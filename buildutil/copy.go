package buildutil

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// CopyFile copies the source file into the destination file
func CopyFile(source, dest string, mode os.FileMode) error {
	if _, err := os.Stat(source); os.IsNotExist(err) {
		return fmt.Errorf("source file not found at %q", source)
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("error creating directory for %q: %s", dest, err)
	}

	vf, err := os.OpenFile(dest, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, mode)
	if err != nil || vf == nil {
		return fmt.Errorf("error opening target file %q: %s", dest, err)
	}
	defer vf.Close()

	bv, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("error opening source file %q: %s", source, err)
	}
	defer bv.Close()

	_, err = io.Copy(vf, bv)
	if err != nil {
		return fmt.Errorf("error copying file: %s", err)
	}

	return nil
}

// CopyBundleBinaries copies binaries out of a binary bundle
// directory. Expected to be a directory of binaries generated
// from a Docker build. The directory parent is expected to
// be the version number and next parent the "bundles" directory.
func CopyBundleBinaries(source, target string) error {
	suffix := versionSuffix(source)
	fis, err := ioutil.ReadDir(source)
	if err != nil {
		return err
	}
	for _, fi := range fis {
		name := fi.Name()
		ext := filepath.Ext(name)
		if ext != ".sha256" {
			continue
		}
		name = name[:len(name)-7]
		targetName := name
		if strings.HasSuffix(targetName, suffix) {
			targetName = targetName[:len(targetName)-len(suffix)]
		}
		if err := copyBinary(name, targetName, source, target); err != nil {
			return fmt.Errorf("copy failed: %v", err)
		}
	}

	return nil
}

// versionSuffix gets the version suffix for a source
// directory, the expected format is "*/bundles/<version>/binarydir/"
// If the source directory is different the version will be empty.
func versionSuffix(source string) string {
	bundleDir, version := filepath.Split(filepath.Dir(source))
	if filepath.Base(filepath.Clean(bundleDir)) == "bundles" {
		return "-" + version
	}
	return ""
}

func hashCheck(file string, expected []byte, h hash.Hash) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	hb := h.Sum(nil)
	if bytes.Compare(expected, hb) != 0 {
		return errors.New("hash mismatch")
	}
	return nil
}

func copyBinary(sourceName, targetName, sourceDir, targetDir string) error {
	source := filepath.Join(sourceDir, sourceName)
	dest := filepath.Join(targetDir, targetName)
	if _, err := os.Stat(source); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("missing file %s", source)
		}
		return err
	}
	if err := CopyFile(source, dest, 0755); err != nil {
		return err
	}
	if b, err := ioutil.ReadFile(source + ".sha256"); err == nil {
		if i := bytes.IndexRune(b, ' '); i > 0 {
			b = b[:i]
		}
		expectedHash := make([]byte, hex.DecodedLen(len(b)))
		if _, err := hex.Decode(expectedHash, b); err != nil {
			return err
		}
		if err := hashCheck(dest, expectedHash, sha256.New()); err != nil {
			return err
		}
	}

	return nil
}
