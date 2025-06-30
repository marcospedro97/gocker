package filesystem

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	layerFileExt = ".gz"
	dirPerm      = 0o755
)

// BuildFromLayers extracts all layer archives (*.gz) from a directory and builds the root filesystem at targetRoot.
// This function is the only one that orchestrates the others: extraction, decompression, untar and file writing.
func BuildFromLayers(layersDir, targetRoot string) error {
	files, err := os.ReadDir(layersDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != layerFileExt {
			continue
		}

		layerPath := filepath.Join(layersDir, file.Name())

		extractLayer(layerPath, targetRoot)
	}
	return nil
}

// extractLayer extracts a single layer from a gzipped tar archive.
// It opens the layer file, creates a gzip reader, and then a tar reader to process
// the contents of the tar archive. It handles different types of entries in the tar file
// such as directories, regular files, symlinks, and hard links.
// The extracted files are written to the targetRoot directory, maintaining the original structure.
// It returns an error if any operation fails, such as opening the file, creating readers,
func extractLayer(layerPath, targetRoot string) error {
	fmt.Printf("Extracting layer %s...\n", layerPath)

	file, err := os.Open(layerPath)
	if err != nil {
		return fmt.Errorf("failed to open layer file: %w", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	err = handleTarHeader(tarReader, targetRoot)

	if err != nil {
		return fmt.Errorf("failed to handle tar header: %w", err)
	}

	fmt.Printf("Layer %s extracted successfully.\n", layerPath)
	return nil
}

// handleTarHeader processes each entry in the tar archive.
// It uses a map of handlers to call the appropriate function based on the type of entry.
// The handlers are responsible for creating directories, writing regular files, creating symlinks, and handling hard links.
func handleTarHeader(tarReader *tar.Reader, targetRoot string) error {
	handlers := map[byte]func(*tar.Header, io.Reader, string) error{
		tar.TypeDir:     handleDir,
		tar.TypeReg:     handleReg,
		tar.TypeSymlink: handleSymlink,
		tar.TypeLink:    handleLink,
	}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		handler, ok := handlers[header.Typeflag]
		if !ok {
			return fmt.Errorf("unknown tar entry type: %c", header.Typeflag)
		}

		err = handler(header, tarReader, targetRoot)
		if err != nil {
			return fmt.Errorf("failed to handle tar entry %s: %w", header.Name, err)
		}
	}

	return nil
}
