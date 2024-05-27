package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var DB *sqlx.DB

const ACTION_CREATE = "+"
const ACTION_DO_ID = "!"
const ACTION_UNDO_ID = "?"
const ACTION_CLEAR_ALL = "|"

func main() {
	CheckDbAndConnect()
	defer DB.Close()
	args := os.Args
	var err error
	var selection string

	if len(args) == 2 {
		selection = args[1]

		if len(selection) > 0 {
			err = ManageSelection(selection)
		}

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

func ManageSelection(selection string) error {
	action := selection[:1]
	actionValue := selection[1:]

	switch action {
	case "+":
		return CreateTodo(actionValue)
	case "!":
		intActionValue, err := strconv.Atoi(actionValue)

		if err != nil {
			return err
		}

		return MarkTodoDone(intActionValue)
	case "?":
		intActionValue, err := strconv.Atoi(actionValue)

		if err != nil {
			return err
		}

		return MarkTodoNotDone(intActionValue)
	case "|":
		return ClearAll();
	default:
		return MarkTodoDoneFromSelection(selection)
	}
}

func CheckDbAndConnect() error {
	var err error
	DB, err = sqlx.Connect("sqlite3", "./rofi-todo.db")

	if err != nil {
		return err
	}

	_, err = DB.Exec("CREATE TABLE IF NOT EXISTS todos (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL UNIQUE, done BOOLEAN default false, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, finished_at TIMESTAMP)")

	if err != nil {
		return err
	}

	_, err = DB.Exec("CREATE TABLE IF NOT EXISTS todos_eliminados (deleted_at TIMESTAMP not null, id INTEGER PRIMARY KEY, title TEXT NOT NULL, done BOOLEAN default false, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, finished_at TIMESTAMP)")
	return err
}

type Todo struct {
	ID         int        `db:"id"`
	Title      string     `db:"title"`
	Done       bool       `db:"done"`
	CreatedAt  time.Time  `db:"created_at"`
	FinishedAt *time.Time `db:"finished_at"`
}

func GetTodos() ([]Todo, error) {
	todos := []Todo{}
	err := DB.Select(&todos, "SELECT * FROM todos ORDER BY done asc, created_at DESC")

	if err != nil {
		return nil, err
	}

	return todos, nil
}

func CreateTodo(title string) error {
	_, err := DB.Exec("INSERT INTO todos (title) VALUES (?)", title)
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

func (t *Todo) Description() string {
	doneText := "✘"
	if t.Done {
		doneText = "✔"
	}

	return fmt.Sprintf("[%v] %v %v %v", t.ID, t.CreatedAt.Format("2006-01-02 15:04"), t.Title, doneText)
}

func ErrorNotify(err error) {
	cmd := exec.Command("notify-send", "-a", "rofi-todo", "Error", err.Error())
	cmd.Run()
}
