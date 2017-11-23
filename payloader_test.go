package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type helper struct {
	t *testing.T
}

func (h helper) tempDir() string {
	h.t.Helper()
	dest, err := ioutil.TempDir("", "payloader-tests")
	if err != nil {
		h.t.Fatalf("can not create temp dir %s", err)
	}
	return dest
}

func (h helper) assertDirectoryContentsEqual(src, dest string) {

	h.t.Helper()

	err := filepath.Walk(src, func(srcPath string, srcFI os.FileInfo, err error) error {

		// skip the first file (this is the working directory, not
		// part of the archive)
		if src == srcPath {
			return nil
		}

		destPath := filepath.Join(dest, strings.TrimPrefix(srcPath, src))
		destFI, err := os.Stat(destPath)

		if err != nil {
			h.t.Fatalf("assertDirectoryContentsEqual: source file %q not found at %q", srcPath, destPath)
		}

		if srcFI.Mode() != destFI.Mode() {
			h.t.Fatalf("assertDirectoryContentsEqual: source file %q mode (%q) did not match dest file mode %q (%q)", srcPath, srcFI.Mode(), destPath, destFI.Mode())
		}

		if srcFI.IsDir() != destFI.IsDir() {
			h.t.Fatalf("assertDirectoryContentsEqual: source file %q is dir (%q) did not match dest file is dir %q (%q)", srcPath, srcFI.IsDir(), destPath, destFI.IsDir())
		}

		if srcFI.Size() != destFI.Size() {
			h.t.Fatalf("assertDirectoryContentsEqual: source file %q size (%q) did not match dest file size %q (%q)", srcPath, srcFI.Size(), destPath, destFI.Size())
		}

		return nil
	})

	if err != nil {
		h.t.Fatal("assertDirectoryContentsEqual: error %s", err)
	}

}

func Test_Payloader_RaisesErrorIfSourceDoesNotExist(t *testing.T) {
	t.Skip("not implemented")
}

func Test_Payloader_CorrectlyTarsADirectory(t *testing.T) {

	assertDirEql := helper{t}.assertDirectoryContentsEqual
	dest := helper{t}.tempDir()

	var src = "fixtures/happy-path"

	p := Payloader{}
	res, err := p.DirToBase64EncTarGz(src)
	if err != nil {
		log.Fatal(err)
	}

	err = p.Base64EncTarGzToDir(dest, res)
	if err != nil {
		fmt.Println(err)
	}

	assertDirEql(src, dest)

}

// _ = Gantry{ctx: context.TODO(), src: mockSrc{}, logger: NoopLogger{}}
// _ = mockSrc{mockErr: errors.Errorf("hello world, I'm ded.")}
