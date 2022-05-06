package main

import (
	"WrapNGo/parsing"
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// compress creates a tar gzip archive
func compress(path string, overwrite bool) (output string, err error) {
	tm := time.Now()
	_, err = os.Stat(path)
	if err != nil {
		return
	}

	buf := bytes.Buffer{}
	err = compressPath(path, &buf)
	if err != nil {
		return
	}

	parent := filepath.Dir(path)
	dirName := filepath.Base(path)
	t, err := parsing.ParseDate(tm, "YYYY-MM-DD_hhmmssms")
	if err != nil {
		return
	}

	output = fmt.Sprintf("%s/%s-%s.tar.gz", parent, dirName, t)
	_, err = os.Stat(output)
	if err == nil {
		if !overwrite {
			return "", errors.New(ErrArchAlreadyExists)
		} else {
			err = os.Remove(output)
			if err != nil {
				return
			}
		}
	}

	arch, err := os.OpenFile(output, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return
	}

	_, err = io.Copy(arch, &buf)
	return
}

// compressPath creates a tar gzip file of the given source.
func compressPath(src string, buf io.Writer) (err error) {
	gzW := gzip.NewWriter(buf)
	tarW := tar.NewWriter(gzW)
	err = filepath.Walk(src, func(path string, info fs.FileInfo, err error) (wErr error) {
		// Tar header.
		h, wErr := tar.FileInfoHeader(info, path)
		if wErr != nil {
			return
		}

		h.Name = filepath.ToSlash(path)
		wErr = tarW.WriteHeader(h)
		if wErr != nil {
			return
		}

		if !info.IsDir() {
			var data *os.File
			data, wErr = os.Open(path)
			if wErr != nil {
				return wErr
			}

			_, wErr = io.Copy(tarW, data)
			if wErr != nil {
				return wErr
			}
		}
		return
	})
	if err != nil {
		return
	}

	err = tarW.Close()
	if err != nil {
		return
	}
	err = gzW.Close()
	return
}
