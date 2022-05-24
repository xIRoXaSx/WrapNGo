package main

import (
	"WrapNGo/parsing"
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
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

	parent := filepath.Dir(path)
	dirName := filepath.Base(path)
	t, err := parsing.ParseDate(tm, "YYYY-MM-DD_hhmmssms")
	if err != nil {
		return
	}

	// In order to use in-memory compression, the dir size should be less than or equal to 1GB.
	exceeds, err := calcMaxFileSize(path, 1073741824)
	if err != nil {
		return
	}

	// Remove file if already existing and overwrite flag is true.
	removeOrErr := func() (err error) {
		_, err = os.Stat(output)
		if err == nil {
			if !overwrite {
				return errors.New(ErrArchAlreadyExists)
			} else {
				err = os.Remove(output)
				if err != nil {
					return
				}
			}
		}
		return nil
	}

	output = filepath.Join(parent, dirName+"-"+t+".tar.gz")
	if exceeds {
		err = removeOrErr()
		if err != nil {
			return "", err
		}

		var f *os.File
		f, err = os.OpenFile(output, os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			return
		}

		err = compressPath(path, f)
		if err != nil {
			return
		}
		return
	}

	buf := bytes.Buffer{}
	err = compressPath(path, &buf)
	err = removeOrErr()
	if err != nil {
		return "", err
	}

	arch, err := os.OpenFile(output, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return
	}

	_, err = io.Copy(arch, &buf)
	err = arch.Close()
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

			wErr = data.Close()
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

// calcMaxFileSize calculates the size of the directory.
// If the size is greater than max (in bytes), it returns false.
func calcMaxFileSize(path string, max int64) (exceedsMax bool, err error) {
	var size int64
	err = filepath.Walk(path, func(_ string, info fs.FileInfo, err error) (wErr error) {
		if !info.IsDir() {
			size += info.Size()
		}
		if (size / 1024) > max {
			exceedsMax = true
			return io.EOF
		}
		return
	})
	if errors.Is(err, io.EOF) {
		err = nil
		return
	}

	return size > max, err
}
