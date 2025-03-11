# Rofi-todo

## Description
A simple todo backend to be managed with a rofi like interface.

## Build

```bash
go build
```
Then optionally move the rofi-todo binary to your PATH so it can be used on the cli as well.

After that you can initialize rofi in todo mode with the following command:
```bash
rofi -modi GODO:[PATH TO rofi-todo binary] -show GODO
```

## Usage
- The main command is `rofi-todo` wich prints all active todos.
- rofi-todo is used with text command only, no flags. What's written in the input is considered the command and is parsed into the actions.
- All todos are stored in a sqlite database in the user's config directory.
- Removed todos are soft deleted.

### Actions
- `+ "Tags" "Todo Text"` Add a new todo.
- `- "Todo Id"` Remove a todo.
- `! "Todo Id"` Mark a todo as done.
- `? "Todo Id"` Mark a todo as not done.
- `> "Todo Id" "Todo New Text"` Edit a todo.
- `"Todo Text"` Mark a todo as done.

### Tags

> Tags are extra information that can be added to a todo.

#### Alarm

- [a:"Time","Alarm text (Optional)"] Set an alarm for the todo.

#### Priority

- [p:"Priority"] Set a priority for the todo, todos are sorted by priority then id.
