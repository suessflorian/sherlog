package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/jroimartin/gocui"
)

const (
	MAIN_VIEW   = "main"
	ZOOM_VIEW   = "zoomed"
	SEARCH_VIEW = "search"
)

func main() {
	file, err := os.OpenFile("main.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic("cannot spin up log file: " + err.Error())
	}
	defer file.Close()

	lg := slog.New(slog.NewJSONHandler(file, nil))
	slog.SetDefault(lg)

	ui, err := gocui.NewGui(gocui.Output256)
	if err != nil {
		slog.Default().With("err", err.Error()).Error("failed to start new GUI")
		os.Exit(1)
	}
	defer ui.Close()

	ui.SetManagerFunc(live)
	ui.InputEsc = true

	if err := keybindings(ui); err != nil {
		slog.Default().With("err", err.Error()).Error("failed to set keybindings")
		os.Exit(1)
	}

	go func() {
		scanner := bufio.NewScanner(os.Stdin)

		for scanner.Scan() {
			var entry log
			err := json.Unmarshal([]byte(scanner.Text()), &entry)
			if err != nil {
				continue // skip lines that are not valid JSON
			}

			feed = append(feed, entry)

			ui.Update(func(g *gocui.Gui) error {
				if focused == nil {
					if err := renderLogs(g, feed, 0); err != nil {
						return err
					}
				}

				return nil
			})
		}
	}()

	if err := ui.MainLoop(); err != nil && !errors.Is(err, gocui.ErrQuit) {
		slog.Default().With("err", err.Error()).Error("failed to set keybindings")
		os.Exit(1)
	}

	os.Exit(0)
}

var (
	feed    []log
	focused []log

	zoomed    bool
	searching bool

	// log cursor position
	cursor int
)

func live(ui *gocui.Gui) error {
	maxX, maxY := ui.Size()
	if v, err := ui.SetView(MAIN_VIEW, 0, 0, maxX-1, maxY-1); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = "Logs"
		v.Autoscroll = true
	}

	return nil
}

func renderLogs(ui *gocui.Gui, logs []log, cursor int) error {
	v, err := ui.View(MAIN_VIEW)
	if err != nil {
		return err
	}

	v.Clear()

	for i, log := range logs {
		if len(logs)-(cursor+1) == i {
			fmt.Fprintf(v, "> ")
		}
		fmt.Fprintln(v, renderPacked(log))
	}

	return nil
}

func zoomLog(ui *gocui.Gui, log log) error {
	zoomed = true

	maxX, maxY := ui.Size()
	width, height := maxX/2, maxY/2

	// central
	startX := (maxX - width) / 2
	startY := (maxY - height) / 2
	endX := startX + width
	endY := startY + height

	if v, err := ui.SetView(ZOOM_VIEW, startX, startY, endX, endY); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = log.Message
		v.Wrap = true
		v.Autoscroll = true

		fmt.Fprintln(v, renderUnpacked(log))
	}

	return nil
}

func openSearch(ui *gocui.Gui, _ *gocui.View) error {
	searching = true

	maxX, maxY := ui.Size()
	width, height := int(float64(maxX)*0.8), 2

	// bottom central
	startY := int(float64(maxY) * 0.8)
	startX := (maxX - width) / 2
	endX := startX + width
	endY := startY + height

	v, err := ui.SetView(SEARCH_VIEW, startX, startY, endX, endY)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Editable = true
	}

	searchPrompt := "Search Pattern: "
	fmt.Fprint(v, searchPrompt)

	if err := v.SetCursor(len(searchPrompt), 0); err != nil {
		return err
	}

	if _, err := ui.SetCurrentView(SEARCH_VIEW); err != nil {
		return err
	}

	if focused == nil {
		focused = feed[:]
		return renderLogs(ui, focused, 0)
	}

	return nil
}

func keybindings(ui *gocui.Gui) error {
	if err := ui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}
	if err := ui.SetKeybinding("", 'q', gocui.ModNone, quit); err != nil {
		return err
	}
	if err := ui.SetKeybinding("", 'k', gocui.ModNone, moveUp); err != nil {
		return err
	}
	if err := ui.SetKeybinding("", 'j', gocui.ModNone, moveDown); err != nil {
		return err
	}
	if err := ui.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, enterKey); err != nil {
		return err
	}
	if err := ui.SetKeybinding("", '?', gocui.ModNone, openSearch); err != nil {
		return err
	}
	if err := ui.SetKeybinding("", gocui.KeyEsc, gocui.ModNone, escapeKey); err != nil {
		return err
	}

	return nil
}

func enterKey(g *gocui.Gui, v *gocui.View) error {
	if searching {
		view, err := g.View(SEARCH_VIEW)
		if err != nil {
			return err
		}

		pattern := strings.TrimPrefix(view.Buffer(), "Search Pattern: ")
		pattern = strings.TrimSpace(pattern)

		if err := g.DeleteView(SEARCH_VIEW); err != nil {
			return err
		}

		var matched []log
		for _, log := range focused {
			if log.Matches(pattern) {
				matched = append(matched, log)
			}
		}

		searching = false
		focused = matched[:]

		if cursor > (len(focused) - 1) {
			cursor = len(focused) - 1
		}

		return renderLogs(g, focused, 0)
	}

	if focused == nil {
		return nil
	}

	logIndex := len(focused) - (cursor + 1)
	if logIndex < 0 || logIndex >= len(focused) {
		return nil
	}

	return zoomLog(g, feed[logIndex])
}

func moveDown(g *gocui.Gui, v *gocui.View) error {
	if searching || zoomed {
		return nil
	}

	if focused == nil {
		focused = feed[:]
	}
	if cursor > 0 {
		cursor--
	}
	return renderLogs(g, focused, cursor)
}

func moveUp(g *gocui.Gui, v *gocui.View) error {
	if searching || zoomed {
		return nil
	}

	if focused == nil {
		focused = feed[:]
	}
	if cursor < (len(focused) - 1) {
		cursor++
	}
	return renderLogs(g, focused, cursor)
}

func escapeKey(ui *gocui.Gui, v *gocui.View) error {
	if zoomed {
		zoomed = false
		return ui.DeleteView(ZOOM_VIEW)
	}

	if searching {
		searching = false
		return ui.DeleteView(SEARCH_VIEW)
	}

	focused = nil // remove focus
	cursor = 0    // reset cursor
	return renderLogs(ui, feed, cursor)
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
