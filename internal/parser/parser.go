package parser

import "strings"

func ParseInput(input string) (string, []string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return "", nil
	}

	return parts[0], parts[1:]
}
