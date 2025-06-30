package build

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/marcospedro/gocker/internal/dockerfile"
	"github.com/marcospedro/gocker/internal/filesystem"
	"github.com/marcospedro/gocker/internal/image"
)

type Runner struct {
	instructions []dockerfile.Instruction
	rootfsPath   string
	entrypoint   []string
}

func NewRunner(instructions []dockerfile.Instruction) *Runner {
	return &Runner{instructions: instructions}
}

// Runner.Prepare processes the Dockerfile instructions and prepares the root filesystem and entrypoint.
// It returns the path to the root filesystem, the entrypoint command, and any error encountered during processing.
// The root filesystem is built from the layers of the specified image and any additional files copied into it.
// The entrypoint is set based on the ENTRYPOINT instruction in the Dockerfile.
func (r *Runner) Prepare() (string, []string, error) {
	var err error
	lookup := map[string]func(dockerfile.Instruction) error{
		"FromInstruction":       r.handleFrom,
		"CopyInstruction":       r.handleCopy,
		"EntryPointInstruction": r.handleEntrypoint,
	}

	for _, instruction := range r.instructions {
		typeName := fmt.Sprintf("%T", instruction)
		if idx := len("dockerfile."); len(typeName) > idx && typeName[:idx] == "dockerfile." {
			typeName = typeName[idx:]
		}
		handler, ok := lookup[typeName]
		if !ok {
			return "", nil, fmt.Errorf("unsupported instruction type: %T", instruction)
		}
		if err := handler(instruction); err != nil {
			return "", nil, err
		}
	}
	return r.rootfsPath, r.entrypoint, err
}

// handleEntrypoint processes the ENTRYPOINT instruction from the Dockerfile.
// It sets the entrypoint command for the container.
func (r *Runner) handleEntrypoint(inst dockerfile.Instruction) error {
	entry := inst.(dockerfile.EntryPointInstruction)
	if len(entry.Entrypoint) == 0 {
		return fmt.Errorf("entrypoint instruction is empty or not set")
	}

	r.entrypoint = entry.Entrypoint
	return nil
}

// handleFrom processes the FROM instruction from the Dockerfile.
// It downloads the specified image and builds the root filesystem from its layers.
// If the root filesystem already exists, it reuses it instead of downloading again.
// The image is expected to be in the format "imageName:tag"
// where "imageName" is the name of the image and "tag" is the version tag.
func (r *Runner) handleFrom(inst dockerfile.Instruction) error {
	from := inst.(dockerfile.FromInstruction)
	imageName := from.Image
	tag := from.Tag
	fmt.Printf("Building root filesystem for image %s tag:%s...\n", imageName, tag)

	downloadPath := fmt.Sprintf("/tmp/gocker/layers/%s/%s", imageName, tag)
	rootfsPath := fmt.Sprintf("/tmp/gocker/rootfs/%s/%s", imageName, tag)

	_, err := os.Stat(rootfsPath)
	if !os.IsNotExist(err) {
		r.rootfsPath = rootfsPath
		return nil
	}

	_ = os.MkdirAll(downloadPath, 0755)
	_ = os.MkdirAll(rootfsPath, 0755)

	err = image.DownloadImage("library/"+imageName, tag, downloadPath)
	if err != nil {
		return fmt.Errorf("failed to download image %s:%s: %w", imageName, tag, err)
	}

	err = filesystem.BuildFromLayers(downloadPath, rootfsPath)
	if err != nil {
		return fmt.Errorf("failed to build root filesystem: %w", err)
	}

	r.rootfsPath = rootfsPath
	return nil
}

// handleCopy processes the COPY instruction from the Dockerfile.
// It copies files from the host filesystem to the container's root filesystem.
// The source path is relative to the current working directory, and the destination path is relative to the root filesystem.
// If the destination file already exists, it is removed before copying the new file.
// The source file must exist on the host filesystem, or an error is returned.
// The root filesystem path must be set before calling this method.
// It returns an error if the source file does not exist, or if there are issues creating the destination directory or copying the file.
func (r *Runner) handleCopy(inst dockerfile.Instruction) error {
	copy := inst.(dockerfile.CopyInstruction)
	src := copy.Src
	dst := copy.Dst
	rootfsPath := r.rootfsPath
	if rootfsPath == "" || src == "" || dst == "" {
		return fmt.Errorf("invalid rootfs path or source/destination for copy instruction (src: %s, dst: %s, rootfs: %s)", src, dst, rootfsPath)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %v", err)
	}

	srcPath := filepath.Join(cwd, src)
	dstPath := filepath.Join(rootfsPath, dst)

	_, err = os.Stat(dstPath)
	if !os.IsNotExist(err) {
		os.RemoveAll(dstPath)
	}

	_, err = os.Stat(srcPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("source file %s does not exist", srcPath)
	}

	err = os.MkdirAll(filepath.Dir(dstPath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination directory %s: %v", filepath.Dir(dstPath), err)
	}

	err = os.Link(srcPath, dstPath)
	if err != nil {
		return fmt.Errorf("failed to copy %s to %s: %v", srcPath, dstPath, err)
	}
	return nil
}
