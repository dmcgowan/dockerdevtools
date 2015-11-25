package dockerdevtools

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/dmcgowan/dockerdevtools/versionutil"
)

var (
	ErrCannotDownloadCommit = errors.New("cannot download build by commit")
)

type BuildCache struct {
	root string
}

func NewBuildCache(root string) *BuildCache {
	return &BuildCache{
		root: root,
	}
}

func (bc *BuildCache) versionFile(v versionutil.Version) string {
	if v.Commit != "" {
		panic("cannot get release file with commit")
	}

	versionFile := filepath.Join(bc.root, fmt.Sprintf("%d.%d.%d", v.VersionNumber[0], v.VersionNumber[1], v.VersionNumber[2]))
	if v.Tag != "" {
		versionFile = versionFile + "-" + v.Tag
	}

	return versionFile
}

func (bc *BuildCache) getCached(v versionutil.Version) string {
	if v.Commit != "" {
		commitFile := filepath.Join(bc.root, v.Commit)
		if _, err := os.Stat(commitFile); err == nil {
			return commitFile
		}
		return ""
	}

	versionFile := bc.versionFile(v)
	if _, err := os.Stat(versionFile); err == nil {
		return versionFile
	}

	return ""
}

func initFile(f string) string {
	dir, name := filepath.Split(f)
	if strings.HasPrefix(name, "docker") {
		name = "dockerinit" + name[6:]
	} else {
		name = name + "-init"
	}
	return dir + name

}

func (bc *BuildCache) tempFile() (*os.File, error) {
	return ioutil.TempFile(bc.root, "tmp-")
}

func (bc *BuildCache) cleanupTempFile(tmp *os.File) error {
	if err := tmp.Close(); err != nil {
		log.Printf("Failed to close temp file %v: %s", tmp.Name(), err)
	}
	return os.Remove(tmp.Name())
}

func (bc *BuildCache) saveVersion(tmp *os.File, v versionutil.Version) (string, error) {
	source := tmp.Name()
	if err := tmp.Close(); err != nil {
		log.Printf("Failed to close temp file %v: %s", tmp.Name(), err)
	}
	// TODO: Ensure source version matches

	target := bc.versionFile(v)
	if err := os.Rename(source, target); err != nil {
		return "", err
	}
	return target, nil
}

func (bc *BuildCache) IsCached(v versionutil.Version) bool {
	return bc.getCached(v) != ""
}

func (bc *BuildCache) PutVersion(v versionutil.Version, source string) error {
	cached := bc.getCached(v)
	if err := CopyFile(source, cached, 0755); err != nil {
		return err
	}
	sourceInit := initFile(source)
	if _, err := os.Stat(sourceInit); err == nil {
		cachedInit := initFile(cached)
		if err := CopyFile(sourceInit, cachedInit, 0755); err != nil {
			return err
		}
	}

	return nil
}

func (bc *BuildCache) InstallVersion(v versionutil.Version, target string) error {
	cached := bc.getCached(v)
	var cachedInit string
	if cached == "" {
		if v.Commit != "" {
			return ErrCannotDownloadCommit
		}
		resp, err := http.Get(v.DownloadURL())
		if err != nil {
			return err
		}

		tf, err := bc.tempFile()
		if err != nil {
			return err
		}

		_, err = io.Copy(tf, resp.Body)
		if err != nil {
			if err := bc.cleanupTempFile(tf); err != nil {
				// Just log
				log.Printf("Error cleaning up temp file %v: %s", tf.Name(), err)
			}
			return err
		}

		cached, err = bc.saveVersion(tf, v)
		if err != nil {
			return err
		}

		// Remove any "-init"
		cachedInit = initFile(cached)
		if _, err := os.Stat(cachedInit); err == nil {
			if err := os.Remove(cachedInit); err != nil {
				return err
			}
		}
	} else {
		cachedInit = initFile(cached)
	}

	if err := CopyFile(cached, target, 0755); err != nil {
		return err
	}

	targetInit := initFile(target)
	if _, err := os.Stat(cachedInit); err == nil {
		// Create target file, check if name starts with docker, replace with dockerinit
		return CopyFile(cachedInit, targetInit, 0755)
	} else {
		if _, err := os.Stat(targetInit); err == nil {
			// Truncate file, do not remove since operator may only have access
			// to file and not directory. Future calls may rely on overwriting
			// the content of this file.
			vf, err := os.OpenFile(targetInit, os.O_TRUNC|os.O_WRONLY, 0755)
			if err != nil {
				return err
			}
			return vf.Close()
		}
	}

	return nil
}
