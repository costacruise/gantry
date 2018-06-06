package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// A Payloader defines convenient methods to write a directory of data into a
// gzipped base64 string tar and back
type Payloader struct {
	logger Logger
}

// DirToTarGz encodes the given payload with base64, zips it with
// gzip and writes it into a tar file.
func (p Payloader) DirToTarGz(src string) ([]byte, error) {

	// TODO: maybe make sure logger is never nil?
	if p.logger == nil {
		p.logger = noopLogger{}
	}

	var b = new(bytes.Buffer)

	gzw := gzip.NewWriter(b)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	if src == "" {
		return nil, errors.Errorf("payloader: source dir not specificed")
	}

	if _, err := os.Stat(src); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("payloader: can not stat file %s", src))
	}

	err := filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {

		// return early incase walk has errors
		if err != nil {
			return errors.Wrap(err, "payloader: can not walk file tree")
		}

		// create a new dir/file tar header
		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("payloader: can not create tar file info header (%s)", file))
		}

		// overwrite header.name to include path, otherwise
		// all files land in root of tar archive
		header.Name = filepath.Join(filepath.Dir(file), fi.Name())

		// Remove the src path from the tar archive (ensures
		// we get the *contents* of the target path, in our
		// archive root
		header.Name = strings.TrimPrefix(header.Name, filepath.Clean(src))
		// TODO: test for this                        ^^^^^^^^^^^^^^
		// remove it and run Gantry tests to see difference

		// write the tar header
		if err := tw.WriteHeader(header); err != nil {
			return errors.Wrap(err, fmt.Sprintf("payloader: can not write tar file info header (%s)", file))
		}

		// don't try and read directory body contents
		if !fi.Mode().IsRegular() {
			return nil
		}

		// open file for reading the body
		f, err := os.Open(file)
		defer f.Close()
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("can not open file for copying body to tar (%s)", file))
		}

		// copy file data into tar writer
		if _, err := io.Copy(tw, f); err != nil {
			return errors.Wrap(err, fmt.Sprintf("can not write tar file body (%s)", file))
		}

		return nil
	})

	tw.Close()
	gzw.Close()

	return b.Bytes(), err
}

// ExtractTarGzToDir extracts payload as a tar file, unzips each entry. It assumes that the tar file represents a directory and writes any file/directory within into dest.
func (p Payloader) ExtractTarGzToDir(dest string, payload []byte) error {
	gzr, err := gzip.NewReader(bytes.NewReader(payload))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("payloader: error making new gzip reader from source"))
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this
		// happens)
		case header == nil:
			continue
		}

		target := filepath.Join(dest, header.Name)

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return errors.Wrap(err, fmt.Sprintf("payloader: error making directory %s", target))
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("payloader: error opening file for writing %s", target))
			}
			defer f.Close()

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return errors.Wrap(err, fmt.Sprintf("payloader: error copying file contents to archive %s", target))
			}
		}
	}
}
