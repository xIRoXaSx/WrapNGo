package main

import (
	"WrapNGo/config"
	"WrapNGo/logger"
	"WrapNGo/parsing"
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// compress creates a tar gzip archive
func compress(opts *config.CompressionOptions) (output string, err error) {
	tm := time.Now()
	_, err = os.Stat(opts.CompressPathToTarBeforeHand)
	if err != nil {
		return
	}

	parent := filepath.Dir(opts.CompressPathToTarBeforeHand)
	dirName := filepath.Base(opts.CompressPathToTarBeforeHand)
	t, err := parsing.ParseDate(tm, "YYYY-MM-DD_hhmmssms")
	if err != nil {
		return
	}

	// Get the maximum allowed buffer size for in-memory compression.
	sizeReg := regexp.MustCompile("(\\d*)([bBmMkKgG])")
	match := sizeReg.FindStringSubmatch(opts.InMemoryCompressionLimit)
	size := 0
	if len(match) > 1 {
		size, err = strconv.Atoi(match[1])
		if err != nil {
			logger.Error(err)
		}
	}

	// In order to use in-memory compression, the dir size should be greater than or equal to 1 byte.
	exceeds := false
	if size > 0 {
		mp := 1024
		unit := strings.ToLower(match[2])
		switch unit {
		case "k":
			size = size * mp
		case "m":
			size = size * mp * mp
		case "g":
			size = size * mp * mp * mp
		}

		exceeds, err = calcMaxFileSize(opts.CompressPathToTarBeforeHand, int64(size))
		if err != nil {
			return
		}
	}

	// Remove file if already existing and overwrite flag is true.
	removeOrErr := func() (err error) {
		_, err = os.Stat(output)
		if err == nil {
			if !opts.OverwriteCompressed {
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

		err = compressPath(opts.CompressPathToTarBeforeHand, opts.RetainStructure, f)
		if err != nil {
			return
		}
		return
	}

	buf := bytes.Buffer{}
	err = compressPath(opts.CompressPathToTarBeforeHand, opts.RetainStructure, &buf)
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
func compressPath(src string, retainStructure bool, buf io.Writer) (err error) {
	gzW := gzip.NewWriter(buf)
	tarW := tar.NewWriter(gzW)
	err = filepath.Walk(src, func(path string, info fs.FileInfo, err error) (wErr error) {
		// Tar header.
		h, wErr := tar.FileInfoHeader(info, path)
		if wErr != nil {
			return
		}

		if retainStructure {
			h.Name = filepath.ToSlash(path)
		} else {
			h.Name = filepath.Base(path)
		}
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
