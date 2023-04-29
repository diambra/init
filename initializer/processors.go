package initializer

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Processor interface {
	Process(path string) error
}

type ZipProcessor struct {
}

func (u *ZipProcessor) Process(path string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("couldn't open zip file %s: %w", path, err)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("couldn't remove zip file %s: %w", path, err)
	}
	if err := os.Mkdir(path, 0755); err != nil {
		return fmt.Errorf("couldn't create directory %s: %w", path, err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			if err := os.Mkdir(filepath.Join(path, f.Name), 0755); err != nil {
				return fmt.Errorf("couldn't create directory %s: %w", f.Name, err)
			}
			continue
		}
		if err := u.unzipFile(path, f); err != nil {
			return err
		}
	}
	return nil
}

func (u *ZipProcessor) unzipFile(path string, f *zip.File) error {
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("couldn't open file %s in zip file %s: %w", f.Name, path, err)
	}
	defer rc.Close()
	dw, err := os.OpenFile(filepath.Join(path, f.Name), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return fmt.Errorf("couldn't create file %s from zip file %s: %w", f.Name, path, err)
	}
	defer dw.Close()
	_, err = io.Copy(dw, rc)
	if err != nil {
		return fmt.Errorf("couldn't copy file %s from zip file %s: %w", f.Name, path, err)
	}
	return nil
}
