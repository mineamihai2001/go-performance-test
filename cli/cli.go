package cli

import (
	"fmt"
	"os"
)

type Command struct {
	Name        string
	Short       string
	Description string
	Handler     func(params ...string)
}

type Manager struct {
	commands []*Command
}

func Create() Manager {
	return Manager{
		commands: make([]*Command, 0),
	}
}

func (m *Manager) Add(c *Command) {
	temp := *c
	temp.Name = "--" + temp.Name
	temp.Short = "-" + temp.Short

	m.commands = append(m.commands, &temp)
}

func (m *Manager) Start() {
	args := os.Args[1:]

	commands := getCommands(&args)
	for comm, params := range commands {
		m.execute(comm, params...)
	}
}

func getCommands(args *[]string) map[string][]string {
	commands := make(map[string][]string, 0)

	var currentArg string
	for _, arg := range *args {
		if string(arg[0]) == "-" {
			currentArg = arg
			commands[currentArg] = make([]string, 0)
		} else {
			commands[currentArg] = append(commands[currentArg], arg)
		}
	}
	return commands
}

func (m *Manager) execute(commandName string, params ...string) {
	found := false
	if commandName == "--help" || commandName == "-h" {
		m.help()
		return
	}
	for _, command := range m.commands {
		if command.Name == commandName || command.Short == commandName {
			found = true
			command.Handler(params...)
		}
	}
	if found {
		return
	}
	fmt.Printf("command <%s> not found (use --help to find more details)\n", commandName)
}

func (m *Manager) help(params ...string) {
	for _, command := range m.commands {
		fmt.Printf("%s, %s\n", command.Name, command.Description)
	}
}
