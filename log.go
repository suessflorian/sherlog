package main

import (
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
)

type log struct {
	Level   logrus.Level `json:"level"`
	Message string       `json:"msg"`

	all map[string]json.RawMessage
}

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
