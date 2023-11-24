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
)

// WARN: state variables
var (
	feed    []log
	focused []log

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

func renderLogs(g *gocui.Gui, logs []log, cursor int) error {
	v, err := g.View(MAIN_VIEW)
	if err != nil {
		return err
	}

	v.Clear()

	for i, log := range logs {
		if len(logs)-(cursor+1) == i {
			fmt.Fprintf(v, "> ")
		}
		fmt.Fprintln(v, renderedLog(log))
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

func clearFocus(g *gocui.Gui, v *gocui.View) error {
	focused = nil // remove focus
	cursor = 0    // reset cursor
	return renderLogs(g, feed, cursor)
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
