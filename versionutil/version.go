// Package versionutil provides utility functions
// for working with versions of Docker including
// parsing, comparing, and retrieving information.
package versionutil

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Version represents a specific release or build of
// Docker.
type Version struct {
	Name          string
	versionNumber [3]int
	Tag           string
	Commit        string
}

func (v Version) String() string {
	s := v.Name
	if v.Commit != "" {
		s += "@" + v.Commit
	}
	return s
}

func (v Version) VersionString() string {
	return versionString(v.versionNumber[0], v.versionNumber[1], v.versionNumber[2])
}

func (v Version) downloadURL(os, arch string) string {
	// downloadLocation
	// Install stable
	// https://download.docker.com/linux/static/stable/x86_64/
	// Install test
	// https://download.docker.com/linux/static/test/x86_64/
	// Install edge
	// https://download.docker.com/linux/static/edge/x86_64/
	// Install release (pre 17.03)
	// https://get.docker.com/builds/Linux/x86_64/docker-1.9.0
	suffix := ".tgz"
	tarVersion := StaticVersion(1, 11, 0)
	tarVersion.Tag = "rc1"
	if v.LessThan(tarVersion) {
		suffix = ""
	}

	if v.Tag == "" {
		if v.versionNumber[0] < 17 {
			return fmt.Sprintf("https://get.docker.com/builds/%s/%s/docker-%s%s", os, arch, v.VersionString(), suffix)
		}
	}

	if strings.HasPrefix(v.Tag, "ce") {
		channel := "stable"
		if strings.HasPrefix(v.Tag, "ce-rc") {
			channel = "test"
		}
		// TODO: Support edge channel

		return fmt.Sprintf("https://download.docker.com/%s/static/%s/%s/docker-%s-%s%s", os, channel, arch, v.VersionString(), v.Tag, suffix)
	}

	return ""

}

var (
	versionRegexp = regexp.MustCompile(`v?([0-9]+).([0-9]+).([0-9]+)-((?:[a-z][a-z0-9]+)(?:-[a-z0-9_]+)*)(?:@([a-f0-9]+(?:-dirty)?))?`)
)

// ParseVersion parses a version string as used by
// Docker version command and git tags.
func ParseVersion(s string) (v Version, err error) {
	submatches := versionRegexp.FindStringSubmatch(s)
	if len(submatches) != 6 {
		return Version{}, errors.New("no version match")
	}
	v.Name = submatches[0]
	v.versionNumber[0], err = strconv.Atoi(submatches[1])
	if err != nil {
		return
	}
	v.versionNumber[1], err = strconv.Atoi(submatches[2])
	if err != nil {
		return
	}
	v.versionNumber[2], err = strconv.Atoi(submatches[3])
	if err != nil {
		return
	}
	v.Tag = submatches[4]
	v.Commit = submatches[5]

	if v.Commit != "" {
		v.Name = v.Name[0 : len(v.Name)-len(v.Commit)-1]
	}

	return
}

// LessThan returns true if the provided version is less
// than the version.
func (v Version) LessThan(v2 Version) bool {
	if v.versionNumber[0] != v2.versionNumber[0] {
		return v.versionNumber[0] < v2.versionNumber[0]
	}
	if v.versionNumber[1] != v2.versionNumber[1] {
		return v.versionNumber[1] < v2.versionNumber[1]
	}
	if v.versionNumber[2] != v2.versionNumber[2] {
		return v.versionNumber[2] < v2.versionNumber[2]
	}
	if v.Tag != v2.Tag {
		if v.Tag == "" {
			// Final release always latest for version number
			return false
		}
		if v2.Tag == "" {
			return true
		}
		if v.Tag == "dev" {
			// Dev branch is considered before a tag name is assigned
			return true
		}
		if strings.HasPrefix(v.Tag, "rc") && !strings.HasPrefix(v2.Tag, "rc") {
			// rc is always last tag before final release
			return false
		}
		return v.Tag < v2.Tag
	}

	// This is only for consistent sort order, not
	// for which version is newer/older. Need full commit
	// history to make decision if on same branch
	return v.Commit < v2.Commit
}

var versionOutput = regexp.MustCompile(`Docker version ([a-z0-9-.]+), build ([a-f0-9]+(?:-dirty)?)`)

// BinaryVersion gets the Docker version for the provided Docker binary
func BinaryVersion(executable string) (Version, error) {
	cmd := exec.Command(executable, "--version")
	out, err := cmd.Output()
	if err != nil {
		return Version{}, err
	}

	matches := versionOutput.FindStringSubmatch(strings.TrimSpace(string(out)))
	if len(matches) != 3 {
		return Version{}, fmt.Errorf("unexpected response from version: %s", string(out))
	}
	v, err := ParseVersion(matches[1])
	if err != nil {
		return Version{}, err
	}
	v.Commit = matches[2]

	return v, nil
}

// StaticVersion returns a version object for the given
// version number. This can be useful to compare a version
// against a specific release.
func StaticVersion(major, minor, release int) Version {
	return Version{
		Name:          fmt.Sprintf("v%s", versionString(major, minor, release)),
		versionNumber: [3]int{major, minor, release},
	}
}

func versionString(major, minor, release int) string {
	if major < 17 {
		return fmt.Sprintf("%d.%d.%d", major, minor, release)
	}
	// Version 17 represents point at which format switching to YY.MM, require 2 digits with leading 0
	return fmt.Sprintf("%02d.%02d.%d", major, minor, release)
}
