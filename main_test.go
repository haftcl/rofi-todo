package main

import (
	"testing"
	"time"
)

var tests = map[string][]any{
	"+":         {[]string{"", "+Create"}, "Create"},
	"!":         {[]string{"", "!1"}, "1"},
	"?":         {[]string{"", "?1"}, "1"},
	"-":         {[]string{"", "-1"}, "1"},
	">":         {[]string{"", ">1 test"}, "1 test"},
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

func TestParseTags(t *testing.T) {
	todo := NewTodo("a:2024-06-01 13:53,Test:a p:1:p Prueba")
	todoText := "Prueba"
	alarmTime, _ := time.Parse("2006-01-02 15:04", "2024-06-01 13:53")
	alarmText := "Test"
	err := todo.ExtractTags()

	if err != nil {
		t.Errorf("Error: %v", err)
	}

	if todo.Priority != 1 {
		t.Errorf("Priority: %d != 1", todo.Priority)
	}

	if todo.AlarmTime.Compare(alarmTime) != 0 {
		t.Errorf("AlarmTime: %v != %v", todo.AlarmTime, alarmTime)
	}

	if todo.AlarmText == nil || *todo.AlarmText != alarmText {
		t.Errorf("AlarmText: %v != %s. todo: %+v", *todo.AlarmText, alarmText, todo)
	}

	if todo.Title != todoText {
		t.Errorf("Text: %s != %s", todo.Title, todoText)
	}
}
