package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/jroimartin/gocui"
)

const (
	MAIN_VIEW = "main"
	ZOOM_VIEW = "zoomed"
)

// WARN: state variables
var (
	feed    []log
	focused []log

	zoomed bool

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

	startX := (maxX - width) / 2
	startY := (maxY - height) / 2
	endX := startX + width
	endY := startY + height

	if v, err := ui.SetView(ZOOM_VIEW, startX, startY, endX, endY); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = log.Message
		v.Autoscroll = true

		fmt.Fprintln(v, renderUnpacked(log))
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
	if err := ui.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, zoomIntoLog); err != nil {
		return err
	}
	if err := ui.SetKeybinding("", gocui.KeyEsc, gocui.ModNone, clearFocus); err != nil {
		return err
	}
	if err := ui.SetKeybinding("", 'k', gocui.ModNone, moveUp); err != nil {
		return err
	}
	if err := ui.SetKeybinding("", 'j', gocui.ModNone, moveDown); err != nil {
		return err
	}

	return nil
}

func zoomIntoLog(g *gocui.Gui, v *gocui.View) error {
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
	if focused == nil {
		focused = feed[:]
	}
	if cursor > 0 {
		cursor--
	}
	return renderLogs(g, focused, cursor)
}

func moveUp(g *gocui.Gui, v *gocui.View) error {
	if focused == nil {
		focused = feed[:]
	}
	if cursor < (len(focused) - 1) {
		cursor++
	}
	return renderLogs(g, focused, cursor)
}

func clearFocus(ui *gocui.Gui, v *gocui.View) error {
	if zoomed {
		zoomed = false
		return ui.DeleteView(ZOOM_VIEW)
	}

	focused = nil // remove focus
	cursor = 0    // reset cursor
	return renderLogs(ui, feed, cursor)
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func main() {
	ui, err := gocui.NewGui(gocui.Output256)
	if err != nil {
		panic(err)
	}
	defer ui.Close()

	ui.SetManagerFunc(live)
	ui.InputEsc = true

	if err := keybindings(ui); err != nil {
		panic(err)
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
		panic(err)
	}
}
