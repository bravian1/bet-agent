package prompts

import (
	"embed"
	"fmt"
)

//go:embed *.txt
var promptFiles embed.FS

// Get returns the content of the specified prompt file
func Get(filename string) (string, error) {
	data, err := promptFiles.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read embedded prompt %s: %w", filename, err)
	}
	return string(data), nil
}
