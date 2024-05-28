package main

import (
	"testing"
)

var tests = map[string][]any {
	"+": {[]string{"",  "+Create"}, "Create"},
	"!": {[]string{"", "!1"}, "1"},
	"?": {[]string{"", "?1"}, "1"},
	"-": {[]string{"", "-1"}, "1"},
	">": {[]string{"", ">1 test"}, "1 test"},
	"SELECTION": {[]string{"", "test"}, "test"},
}

func TestCommandFromCmdArgs(t *testing.T) {
	for action, test := range tests {
		command, err := CommandFromCmdArgs(test[0].([]string))

		if err != nil {
			t.Errorf("Error: %v", err)
		}

		if command == nil {
			t.Errorf("Command is nil")
		}

		if command.Action != action {
			t.Errorf("Action: %s != %s", command.Action, action)
		}

		if command.Value != test[1] {
			t.Errorf("Value: %s != %s", command.Value, test[1])
		}
	}
}
