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
	"github.com/sirupsen/logrus"
)

type log struct {
	Level   logrus.Level `json:"level"`
	Message string       `json:"msg"`

	all map[string]json.RawMessage
}

const (
	MAIN_VIEW = "main"
)

// WARN: state variables
var (
	entries []log
	focus   int
)

func (l *log) UnmarshalJSON(data []byte) error {
	all := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &all); err != nil {
		return err
	}

	l.all = all

	if err := json.Unmarshal(all["level"], &l.Level); err != nil {
		return err
	}
	if err := json.Unmarshal(all["msg"], &l.Message); err != nil {
		return err
	}

	return nil
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView(MAIN_VIEW, 0, 0, maxX-1, maxY-1); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = "Logs"
		v.Autoscroll = true
	}

	for i, entry := range entries {
		if err := displayLogEntry(g, entry, i); err != nil {
			panic(err)
		}
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
	if focus < len(entries)-1 {
		focus++
		g.Update(func(g *gocui.Gui) error {
			return layout(g)
		})
	}
	return nil
}

func moveUp(g *gocui.Gui, v *gocui.View) error {
	if focus > 0 {
		focus--
		g.Update(func(g *gocui.Gui) error {
			return layout(g)
		})
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func displayLogEntry(g *gocui.Gui, entry log, index int) error {
	maxX, _ := g.Size()
	v, err := g.SetView(fmt.Sprintf("%d", index), 1, index+2, maxX-1, (index+2)*3-1)
	if err != nil && !errors.Is(err, gocui.ErrUnknownView) {
		return err
	}

	v.Clear()
	v.Frame = false

	if focus == index {
		fmt.Fprint(v, colour.HiMagentaString("> "))
	}
	fmt.Fprint(v, renderedLog(entry))

	return nil
}

func renderedLog(entry log) string {
	switch entry.Level {
	case logrus.ErrorLevel:
		marshalled, err := json.MarshalIndent(entry.all, "", "  ")
		if err != nil {
			panic(err)
		}

		return string(defaultColorize(marshalled, errord))
	case logrus.DebugLevel:
		marshalled, err := json.Marshal(entry)
		if err != nil {
			panic(err)
		}
		return string(defaultColorize(marshalled, debug) + debug.value(fmt.Sprintf(" ... %d hidden fields", len(entry.all)-2)))
	default:
		marshalled, err := json.Marshal(entry)
		if err != nil {
			panic(err)
		}

		return string(defaultColorize(marshalled, standard)) + debug.value(fmt.Sprintf(" ... %d hidden fields", len(entry.all)-2))
	}
}

func main() {
	ui, err := gocui.NewGui(gocui.Output256)
	if err != nil {
		panic(err)
	}
	defer ui.Close()

	ui.SetManagerFunc(layout)

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

			entries = append(entries, entry)

			ui.Update(func(g *gocui.Gui) error {
				for i, entry := range entries {
					if err := displayLogEntry(g, entry, i); err != nil {
						panic(err)
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
