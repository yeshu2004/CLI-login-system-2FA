package cli

import (
	"github/yeshu2004/cli-login/model"
	"strings"
)

type CommandCompleter struct{}

// only return the part that still needs to be typed.
func (c *CommandCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	input := string(line[:pos])

	commands := []string{
		model.HELP,
		model.REGISTER,
		model.LOGIN,
		model.WHOAMI,
		model.ENABLE2FA,
		model.DISABLE2FA,
		model.LOGOUT,
		model.EXIT,
	}

	for _, cmd := range commands {
		if strings.HasPrefix(cmd, input) {
			newLine = append(newLine, []rune(cmd[len(input):]))
		}
	}

	return newLine, len(input)
}
