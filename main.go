package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	colour "github.com/fatih/color"
	"github.com/sirupsen/logrus"
)

type minimal struct {
	Level   logrus.Level `json:"level"`
	Message string       `json:"msg"`

	extras map[string]json.RawMessage
}

func (m *minimal) UnmarshalJSON(data []byte) error {
	extra := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &extra); err != nil {
		return err
	}

	if err := json.Unmarshal(extra["level"], &m.Level); err != nil {
		return err
	}

	if err := json.Unmarshal(extra["msg"], &m.Message); err != nil {
		return err
	}

	delete(extra, "level")
	delete(extra, "msg")

	m.extras = extra

	return nil
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		var entry minimal
		line := scanner.Text()

		err := json.Unmarshal([]byte(line), &entry)
		if err != nil {
			continue // skip lines that are not valid JSON
		}

		switch entry.Level {
		case logrus.ErrorLevel:
			var data map[string]any
			err := json.Unmarshal([]byte(line), &data)
			if err != nil {
				panic(err)
			}

			marshalled, err := json.MarshalIndent(data, "", "  ")
			if err != nil {
				panic(err)
			}

			fmt.Println(string(defaultColorize(marshalled, errord)))
		case logrus.DebugLevel:
			marshalled, err := json.Marshal(entry)
			if err != nil {
				panic(err)
			}
			fmt.Print(string(defaultColorize(marshalled, debug)))
			fmt.Print(debug.value(fmt.Sprintf(" ... hid %d fields\n", len(entry.extras))))
		default:
			marshalled, err := json.Marshal(entry)
			if err != nil {
				panic(err)
			}

			fmt.Print(string(defaultColorize(marshalled, standard)))
			fmt.Print(debug.value(fmt.Sprintf(" ... hid %d fields\n", len(entry.extras))))
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
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
