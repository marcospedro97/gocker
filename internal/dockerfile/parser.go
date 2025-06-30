package dockerfile

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Instruction represents a parsed Dockerfile instruction.
type Instruction interface{}

type FromInstruction struct {
	Image string
	Tag   string
}

type CopyInstruction struct {
	Src string
	Dst string
}

type EntryPointInstruction struct {
	Entrypoint []string
}

// Parse parses a Dockerfile and returns a slice of instructions.
// It reads the Dockerfile line by line, ignoring comments and empty lines,
// and uses a lookup map to call the appropriate parsing function for each instruction.
func Parse(path string) ([]Instruction, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open Dockerfile: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var instructions []Instruction

	lookup := map[string]func([]string) (Instruction, error){
		"FROM":       parseFrom,
		"COPY":       parseCopy,
		"ENTRYPOINT": parseEntrypoint,
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		parseFn, ok := lookup[parts[0]]
		if !ok {
			return nil, fmt.Errorf("unknown instruction: %s", parts[0])
		}

		instruction, err := parseFn(parts)
		if err != nil {
			return nil, fmt.Errorf("error parsing instruction '%s': %w", line, err)
		}
		instructions = append(instructions, instruction)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading Dockerfile: %w", err)
	}

	return instructions, nil
}

func parseFrom(parts []string) (Instruction, error) {
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid FROM instruction")
	}
	image := strings.Split(parts[1], ":")
	return FromInstruction{Image: image[0], Tag: image[1]}, nil
}

func parseCopy(parts []string) (Instruction, error) {
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid COPY instruction")
	}
	return CopyInstruction{Src: parts[1], Dst: parts[2]}, nil
}

func parseEntrypoint(parts []string) (Instruction, error) {
	parts = parts[1:]

	entrypoint := []string{}

	for _, part := range parts {
		part = strings.Trim(part, `"'[],`)
		if part == "" {
			continue
		}

		entrypoint = append(entrypoint, part)
	}

	return EntryPointInstruction{Entrypoint: entrypoint}, nil
}
