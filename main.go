package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/marcospedro/gocker/internal/build"
	"github.com/marcospedro/gocker/internal/container"
	"github.com/marcospedro/gocker/internal/dockerfile"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("failed to get current working directory: %v\n", err)
		os.Exit(1)
	}

	dockerfilePath := filepath.Join(cwd, "Dockerfile")
	instructions, err := dockerfile.Parse(dockerfilePath)
	if err != nil {
		fmt.Printf("failed to parse Dockerfile: %v\n", err)
		os.Exit(1)
	}

	runner := build.NewRunner(instructions)
	rootfsPath, entrypoint, err := runner.Prepare()
	if err != nil {
		fmt.Printf("failed to prepare runner: %v\n", err)
		os.Exit(1)
	}

	err = container.Run(rootfsPath, entrypoint)
	if err != nil {
		fmt.Printf("container run failed: %v\n", err)
		os.Exit(1)
	}
}
