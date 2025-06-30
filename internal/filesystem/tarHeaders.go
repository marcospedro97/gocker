package filesystem

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
)

func handleSymlink(hdr *tar.Header, r io.Reader, root string) error {
	target := filepath.Join(root, hdr.Name)
	err := os.MkdirAll(filepath.Dir(target), dirPerm)
	if err != nil {
		return err
	}
	_ = os.Remove(target)
	return os.Symlink(hdr.Linkname, target)
}

func handleLink(hdr *tar.Header, r io.Reader, root string) error {
	target := filepath.Join(root, hdr.Name)
	err := os.MkdirAll(filepath.Dir(target), dirPerm)
	if err != nil {
		return err
	}

	linkTarget := filepath.Join(root, hdr.Linkname)
	_ = os.Remove(target)
	return os.Link(linkTarget, target)
}

func handleReg(hdr *tar.Header, r io.Reader, root string) error {
	target := filepath.Join(root, hdr.Name)
	err := os.MkdirAll(filepath.Dir(target), dirPerm)
	if err != nil {
		return err
	}

	outFile, err := os.Create(target)
	if err != nil {
		return err
	}

	defer outFile.Close()

	err = os.Chmod(target, os.FileMode(hdr.Mode))
	if err != nil {
		return err
	}

	_, err = io.Copy(outFile, r)
	return err
}

func handleDir(hdr *tar.Header, r io.Reader, root string) error {
	target := filepath.Join(root, hdr.Name)
	return os.MkdirAll(target, dirPerm)
}
