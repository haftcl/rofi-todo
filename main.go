package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var DB *sqlx.DB

const ACTION_CREATE = "+"
const ACTION_DO_ID = "!"
const ACTION_UNDO_ID = "?"
const ACTION_CLEAR = "-"
const ACTION_EDIT = ">"
const ACTION_PRIORITY = "p"
const ACTION_SELECTION = "SELECTION"
const DATA_FOLDER = `/.config/rofi-todo`
const PRIORITY_TAG = "p"
const ALARM_TAG = "a"

var actions = []string{
	ACTION_CREATE,
	ACTION_DO_ID,
	ACTION_UNDO_ID,
	ACTION_CLEAR,
	ACTION_EDIT,
	ACTION_PRIORITY,
}

func main() {
	var err error

	err = CheckDbAndConnect()

	if err != nil {
		ErrorNotify(err)
		os.Exit(1)
	}

	defer DB.Close()

	command, err := CommandFromCmdArgs(os.Args)

	if err != nil {
		ErrorNotify(err)
	}

	if command != nil {
		err = command.Run()
		if err != nil {
			ErrorNotify(err)
		}
	}

	todos, err := GetTodos()

	if err != nil {
		ErrorNotify(err)
		os.Exit(1)
	}

	for _, todo := range todos {
		fmt.Printf("%v\n", todo.Description())
	}
}

type TodoCommand struct {
	Action string
	Value  string
}

func CommandFromCmdArgs(args []string) (*TodoCommand, error) {
	if len(args) != 2 {
		return nil, nil
	}

	selection := args[1]

	if len(selection) == 0 {
		return nil, nil
	}

	action := selection[:1]

	if len(action) == 0 {
		return nil, fmt.Errorf("No action found")
	}

	t := &TodoCommand{}

	if slices.Contains(actions, action) {
		actionValue := strings.TrimSpace(selection[1:])

		if len(actionValue) == 0 {
			return nil, fmt.Errorf("No action value found")
		}

		t.Action = action
		t.Value = actionValue
	} else {
		t.Action = ACTION_SELECTION
		t.Value = selection
	}

	return t, nil
}

func (t *TodoCommand) Run() error {
	switch t.Action {
	case ACTION_CREATE:
		return CreateTodo(t.Value)
	case ACTION_DO_ID:
		intActionValue, err := strconv.Atoi(t.Value)

		if err != nil {
			return fmt.Errorf("Error converting id to int")
		}

		return MarkTodoDone(intActionValue)
	case ACTION_UNDO_ID:
		intActionValue, err := strconv.Atoi(t.Value)

		if err != nil {
			return fmt.Errorf("Error converting id to int")
		}

		return MarkTodoNotDone(intActionValue)
	case ACTION_CLEAR:
		if len(t.Value) == 0 {
			return fmt.Errorf("Need to specify an action value for clear action")
		}

		if t.Value == "done" {
			return ClearAllDone()
		}

		if t.Value == "all" {
			return ClearAll()
		}

		intActionValue, err := strconv.Atoi(t.Value)

		if err != nil {
			return fmt.Errorf("Error converting id to int")
		}

		return ClearTodo(intActionValue)
	case ACTION_EDIT:
		return EditTodo(t.Value)
	case ACTION_PRIORITY:
		return EditPriority(t.Value)
	case ACTION_SELECTION:
		return CopySelection(t.Value)
	}

	return fmt.Errorf("Action %v not found", t.Action)
}

func EditPriority(s string) error {
	id, val, err := IdAndValueFromSelection(s)

	if err != nil {
		return err
	}

	if val == "" {
		return fmt.Errorf("No valid priority found")
	}

	priority, err := strconv.Atoi(val)

	if err != nil {
		return fmt.Errorf("Error converting priority to int")
	}

	todo, err := GetTodoById(id)

	if err != nil {
		return err
	}

	todo.Priority = priority
	return UpdateTodo(todo)
}

func CheckDbAndConnect() error {
	var err error
	var dataFolder string

	envFolder := os.Getenv("GODO_DATA_FOLDER")

	if len(envFolder) > 0 {
		dataFolder = envFolder
	} else {
		homeFolder, err := os.UserHomeDir()

		if err != nil {
			return err
		}

		dataFolder = homeFolder + DATA_FOLDER
	}

	if _, err = os.Stat(dataFolder); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(dataFolder, 0755)

			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	file := fmt.Sprintf("%s%s", dataFolder, "/rofi-todo.db")
	DB, err = sqlx.Connect("sqlite3", file)

	if err != nil {
		return err
	}

	_, err = DB.Exec("CREATE TABLE IF NOT EXISTS todos (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL UNIQUE, done BOOLEAN default false, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, finished_at TIMESTAMP, alarm_time TIMESTAMP, alarm_text TEXT, priority INT)")

	if err != nil {
		return err
	}

	_, err = DB.Exec("CREATE TABLE IF NOT EXISTS todos_eliminados (deleted_at TIMESTAMP not null, id INTEGER PRIMARY KEY, title TEXT NOT NULL, done BOOLEAN default false, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, finished_at TIMESTAMP, alarm_time TIMESTAMP, alarm_text TEXT, priority INT)")
	return err
}

type Todo struct {
	ID         int        `db:"id"`
	Title      string     `db:"title"`
	Done       bool       `db:"done"`
	CreatedAt  time.Time  `db:"created_at"`
	FinishedAt *time.Time `db:"finished_at"`
	AlarmTime  *time.Time `db:"alarm_time"`
	AlarmText  *string    `db:"alarm_text"`
	Priority   int        `db:"priority"`
}

func NewTodo(title string) *Todo {
	return &Todo{
		Priority: 0,
		Done:     false,
		Title:    title,
	}
}

func GetTodoById(id int) (*Todo, error) {
	todo := &Todo{}
	err := DB.Get(todo, "SELECT * FROM todos WHERE id = ?", id)

	if err != nil {
		return nil, err
	}

	return todo, nil
}

func GetTodos() ([]Todo, error) {
	todos := []Todo{}
	err := DB.Select(&todos, "SELECT * FROM todos ORDER BY done ASC, priority DESC, created_at ASC")

	if err != nil {
		return nil, err
	}

	return todos, nil
}

func CreateTodo(title string) error {
	todo := NewTodo(title)

	// Parse text and create an alarm
	err := todo.ExtractTags()

	if err != nil {
		return err
	}

	_, err = DB.Exec("INSERT INTO todos (title, alarm_time, alarm_text, priority) VALUES (?, ?, ?, ?)", todo.Title, todo.AlarmTime, todo.AlarmText, todo.Priority)
	return err
}

func (todo *Todo) ExtractTags() error {
	err := todo.ExtractPriority()

	if err != nil {
		return err
	}

	err = todo.ExtractAlarm()

	if err != nil {
		return err
	}

	return nil
}

func (todo *Todo) ExtractPriority() error {
	value, err := todo.ExtractTag(PRIORITY_TAG)

	if err != nil {
		return err
	}

	if value == "" {
		return nil
	}

	todo.Priority, err = strconv.Atoi(value)

	if err != nil {
		return fmt.Errorf("Error converting priority to int")
	}

	return nil
}

func (todo *Todo) ExtractAlarm() error {
	value, err := todo.ExtractTag(ALARM_TAG)

	if err != nil {
		return err
	}

	if value == "" {
		return nil
	}

	alarmData := strings.Split(value, ",")
	alarmTime, err := time.Parse("2006-01-02 15:04", alarmData[0])

	if err != nil {
		return err
	}

	var alarmText string
	if len(alarmData) == 1 {
		alarmText = todo.Title
	} else {
		alarmText = strings.TrimSpace(alarmData[1])
	}

	todo.AlarmTime = &alarmTime
	todo.AlarmText = &alarmText

	return GenerateAlarm(alarmText, alarmTime)
}

func (todo *Todo) ExtractTag(tag string) (string, error) {
	openTag := fmt.Sprintf("%s:", tag)
	closeTag := fmt.Sprintf(":%s", tag)

	openIndex := strings.Index(todo.Title, openTag)

	if openIndex == -1 {
		return "", nil
	}

	closeIndex := strings.Index(todo.Title, closeTag)

	if closeIndex == -1 {
		return "", fmt.Errorf("Tag %s not closed", tag)
	}

	tagValue := todo.Title[openIndex+2 : closeIndex]

	if len(tagValue) == 0 {
		return "", fmt.Errorf("Tag %s has no value", tag)
	}

	todo.Title = strings.TrimSpace(strings.Replace(todo.Title, openTag+tagValue+closeTag, "", 1))
	return tagValue, nil
}

func GenerateAlarm(alarm string, time time.Time) error {
	cmd := exec.Command("alarma", alarm, time.Format("15:04 2006-01-02"))
	err := cmd.Run()

	return err
}

func CopySelection(selection string) error {
	r, err := regexp.Compile(`\[([0-9]+)\]`)

	if err != nil {
		return err
	}

	idFindings := r.FindStringSubmatch(selection)

	if len(idFindings) == 0 {
		return fmt.Errorf("No id found in selection")
	}

	id, err := strconv.Atoi(idFindings[1])

    if err != nil {
		return fmt.Errorf("Error converting id to int")
    }

	todo, err := GetTodoById(id)

	if err != nil {
		return err
	}

	cmd := exec.Command("wl-copy", todo.BuildText())
	err = cmd.Run()

	return err
}

func MarkTodoDoneFromSelection(selection string) error {
	r, err := regexp.Compile(`\[([0-9]+)\]`)

	if err != nil {
		return err
	}

	idFindings := r.FindStringSubmatch(selection)

	if len(idFindings) == 0 {
		return fmt.Errorf("No id found in selection")
	}

	_, err = DB.Exec("UPDATE todos SET done = ?, finished_at = CURRENT_TIMESTAMP WHERE id = ? and DONE = false", true, idFindings[1])

	return err
}

func MarkTodoDone(id int) error {
	_, err := DB.Exec("UPDATE todos SET done = ?, finished_at = CURRENT_TIMESTAMP WHERE id = ? and DONE = false", true, id)
	return err
}

func MarkTodoNotDone(id int) error {
	_, err := DB.Exec("UPDATE todos SET done = ?, finished_at = NULL WHERE id = ? and DONE = true", false, id)
	return err
}

func UpdateTodo(todo *Todo) error {
	_, err := DB.Exec("UPDATE todos SET title = ?, priority = ? WHERE id = ?", todo.Title, todo.Priority, todo.ID)
	return err
}

func EditTodo(selection string) error {
	id, val, err := IdAndValueFromSelection(selection)

	val = strings.TrimSpace(val)

	if len(val) == 0 {
		return fmt.Errorf("No value found in selection")
	}

	todo, err := GetTodoById(id)

	if err != nil {
		return fmt.Errorf("Error retrieving todo, error: %v" , err)
	}

	todo.Title = val
	todo.ExtractTags()

	return UpdateTodo(todo)
}

func ClearTodo(id int) error {
	tx, err := DB.BeginTx(context.Background(), nil)

	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO todos_eliminados SELECT CURRENT_TIMESTAMP, * FROM todos where id = ?", id)

	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM todos where id = ?", id)

	if err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

func ClearAllDone() error {
	tx, err := DB.BeginTx(context.Background(), nil)

	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO todos_eliminados SELECT CURRENT_TIMESTAMP, * FROM todos where done = true")

	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM todos where done = true")

	if err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

func ClearAll() error {
	tx, err := DB.BeginTx(context.Background(), nil)

	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO todos_eliminados SELECT CURRENT_TIMESTAMP, * FROM todos")

	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM todos")

	if err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

func (t *Todo) BuildText() string {
	priority := ""
	if t.Priority > 0 {
		priority = fmt.Sprintf("%s:%v:%s", PRIORITY_TAG, t.Priority, PRIORITY_TAG)
	}

	return fmt.Sprintf("%d %s %s", t.ID, priority, t.Title)
}

func (t *Todo) Description() string {
	doneText := "✘"
	if t.Done {
		doneText = "✔"
	}

	return fmt.Sprintf("[%v] [p:%v] [%v] %v %v", t.ID, t.Priority, t.CreatedAt.Format("2006-01-02 15:04"), doneText, t.Title)
}

func ErrorNotify(err error) {
	cmd := exec.Command("notify-send", "-a", "rofi-todo", "Error", err.Error())
	cmd.Run()
}

func IdAndValueFromSelection(selection string) (int, string, error) {
	values := strings.SplitN(selection, " ", 2)

	if len(values) < 1 {
		return 0, "", fmt.Errorf("No id found in selection")
	}

	id, err := strconv.Atoi(values[0])

	if err != nil {
		return 0, "", fmt.Errorf("Error converting id to int")
	}

	value := ""
	if len(values) > 1 {
		value = values[1]
	}

	return id, value, nil
}
