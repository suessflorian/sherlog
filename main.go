package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	colour "github.com/fatih/color"
	"github.com/jroimartin/gocui"
)

const (
	MAIN_VIEW = "main"
)

// WARN: state variables
var (
	logs  []log
	focus int
)

func frame(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView(MAIN_VIEW, 0, 0, maxX-1, maxY-1); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = "Logs"
		v.Autoscroll = true
	}

	if err := displayLogs(g, logs); err != nil {
		panic(err)
	}

	return nil
}

func keybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'k', gocui.ModNone, moveUp); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'j', gocui.ModNone, moveDown); err != nil {
		return err
	}

	return nil
}

func moveDown(g *gocui.Gui, v *gocui.View) error {
	if focus < len(logs)-1 {
		focus++
		g.Update(func(g *gocui.Gui) error {
			return frame(g)
		})
	}
	return nil
}

func moveUp(g *gocui.Gui, v *gocui.View) error {
	if focus > 0 {
		focus--
		g.Update(func(g *gocui.Gui) error {
			return frame(g)
		})
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func displayLogs(g *gocui.Gui, logs []log) error {
	v, err := g.View(MAIN_VIEW)
	if err != nil {
		return err
	}

	v.Clear()
	for i, log := range logs {
		if focus == i {
			fmt.Fprintf(v, "> ")
		}
		fmt.Fprintln(v, renderedLog(log))
	}

	return nil
}

func main() {
	ui, err := gocui.NewGui(gocui.Output256)
	if err != nil {
		panic(err)
	}
	defer ui.Close()

	ui.SetManagerFunc(frame)

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

			logs = append(logs, entry)

			ui.Update(func(g *gocui.Gui) error {
				if err := displayLogs(g, logs); err != nil {
					panic(err)
				}

				return nil
			})
		}
	}()

	if err := ui.MainLoop(); err != nil && !errors.Is(err, gocui.ErrQuit) {
		panic(err)
	}
}

type colouriser struct {
	key     func(a ...interface{}) string
	value   func(a ...interface{}) string
	bracket func(a ...interface{}) string
}

var (
	errord = colouriser{
		key:     colour.New(colour.FgRed, colour.Bold).SprintFunc(),
		value:   colour.New(colour.FgGreen).SprintFunc(),
		bracket: colour.New(colour.FgWhite, colour.Bold).SprintFunc(),
	}
	standard = colouriser{
		key:     colour.New(colour.FgBlue, colour.Bold).SprintFunc(),
		value:   colour.New(colour.FgGreen).SprintFunc(),
		bracket: colour.New(colour.FgWhite, colour.Bold).SprintFunc(),
	}
	debug = colouriser{
		key:     colour.New(colour.FgHiWhite, colour.Faint).SprintFunc(),
		value:   colour.New(colour.FgHiWhite, colour.Faint).SprintFunc(),
		bracket: colour.New(colour.FgHiWhite, colour.Faint).SprintFunc(),
	}
)

func defaultColorize(raw json.RawMessage, colouriser colouriser) string {
	var (
		insideQuotes bool
		parsingKey   bool = true
	)

	var result strings.Builder

	for _, rune := range raw {
		switch {
		case rune == '"':
			insideQuotes = !insideQuotes

			if parsingKey {
				result.WriteString(colouriser.key(string(rune)))
			} else {
				result.WriteString(colouriser.value(string(rune)))
			}
		case !insideQuotes && (rune == '{' || rune == '}' || rune == '[' || rune == ']' || rune == ':'):
			if rune == ':' {
				parsingKey = !parsingKey
			}
			result.WriteString(colouriser.bracket(string(rune)))
		case !insideQuotes && rune == ',':
			parsingKey = true
			result.WriteString(colouriser.bracket(string(rune)))
		case insideQuotes && parsingKey:
			result.WriteString(colouriser.key(string(rune)))
		case insideQuotes && !parsingKey:
			result.WriteString(colouriser.value(string(rune)))
		default:
			result.WriteString(string(rune))
		}
	}

	return result.String()
}
