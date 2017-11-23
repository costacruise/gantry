package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type Payloader struct{}

func (p Payloader) EnumerateFileNames(payload []byte) ([]string, error) {
	return nil, nil
}

func (p Payloader) Base64EncTarGzToDir(dest string, payload []byte) error {

	b64 := base64.NewDecoder(base64.StdEncoding, bytes.NewReader(payload))

	gzr, err := gzip.NewReader(b64)
	defer gzr.Close()
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("payloader: error making new gzip reader from b64 source"))
	}

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

func (p Payloader) DirToBase64EncTarGz(src string) ([]byte, error) {

	var b = new(bytes.Buffer)
	b64 := base64.NewEncoder(base64.StdEncoding, b)
	defer b64.Close()

	gzw := gzip.NewWriter(b64)
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
		header.Name = strings.TrimPrefix(header.Name, src)

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
	b64.Close()

	return b.Bytes(), err
}
