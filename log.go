package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	colour "github.com/fatih/color"
)

type log struct {
	Level   logrus.Level `json:"level"`
	Message string       `json:"msg"`

	all map[string]json.RawMessage
	raw []byte
}

func (l *log) Matches(pattern string) bool {
	return strings.Contains(string(l.raw), pattern)
}

func (l *log) UnmarshalJSON(data []byte) error {
	all := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &all); err != nil {
		return err
	}

	if err := json.Unmarshal(all["level"], &l.Level); err != nil {
		return err
	}
	if err := json.Unmarshal(all["msg"], &l.Message); err != nil {
		return err
	}

	l.all = all
	l.raw = data

	return nil
}

func renderPacked(entry log) string {
	switch entry.Level {
	case logrus.ErrorLevel:
		marshalled, err := json.Marshal(entry.all)
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

func renderUnpacked(entry log) string {
	switch entry.Level {
	case logrus.ErrorLevel:
		marshalled, err := json.MarshalIndent(entry.all, "", " ")
		if err != nil {
			panic(err)
		}

		return string(defaultColorize(marshalled, errord))
	default:
		marshalled, err := json.MarshalIndent(entry.all, "", " ")
		if err != nil {
			panic(err)
		}

		return string(defaultColorize(marshalled, standard))
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
