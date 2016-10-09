package launch

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func WriteZipToFile(name string, conf *Config) error {
	b, err := ZipWorkingDir(conf)
	if err != nil {
		return err
	}

	file, err := os.Create(fmt.Sprintf("%v.zip", name))
	if err != nil {
		return err
	}

	_, err = file.Write(b.Bytes())
	return err
}

func ZipWorkingDir(conf *Config) (*bytes.Buffer, error) {
	fmt.Println("Zipping files...")
	out := new(bytes.Buffer)
	source := "."

	archive := zip.NewWriter(out)
	defer archive.Close()

	baseDir := filepath.Base(source)

	filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(writer, file)
		return err
	})

	if err := appendShim(archive, conf); err != nil {
		return nil, err
	}

	return out, nil
}

func appendShim(archive *zip.Writer, conf *Config) error {
	shim, err := Shim(conf)
	if err != nil {
		return err
	}

	writer, err := archive.Create("launch_shim.js")
	if err != nil {
		return err
	}

	_, err = writer.Write(shim)
	return err
}
