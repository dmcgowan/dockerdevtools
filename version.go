package dockerdevtools

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type Version struct {
	Name          string
	VersionNumber [3]int
	Tag           string
	Commit        string
}

func (v Version) downloadURL(os, arch string) string {
	// downloadLocation
	// Install release
	// https://get.docker.com/builds/Linux/x86_64/docker-1.9.0
	// Install non release
	// https://test.docker.com/builds/Linux/x86_64/docker-1.9.0-rc5
	// Install experimental
	// https://experimental.docker.com/builds/Linux/x86_64/docker-latest
	if v.Tag == "" {
		return fmt.Sprintf("https://get.docker.com/builds/%s/%s/docker-%d.%d.%d", os, arch, v.VersionNumber[0], v.VersionNumber[1], v.VersionNumber[2])
	}
	if strings.HasPrefix(v.Tag, "rc") {
		return fmt.Sprintf("https://test.docker.com/builds/%s/%s/docker-%d.%d.%d-%s", os, arch, v.VersionNumber[0], v.VersionNumber[1], v.VersionNumber[2], v.Tag)
	}

	return ""

}

var (
	versionRegexp = regexp.MustCompile(`v?([0-9]+).([0-9]+).([0-9]+)(?:-([a-z][a-z0-9]+))?(?:@([a-f0-9]+(?:-dirty)?))?`)
)

func ParseVersion(s string) (v Version, err error) {
	submatches := versionRegexp.FindStringSubmatch(s)
	if len(submatches) != 6 {
		return Version{}, errors.New("no version match")
	}
	v.Name = submatches[0]
	v.VersionNumber[0], err = strconv.Atoi(submatches[1])
	if err != nil {
		return
	}
	v.VersionNumber[1], err = strconv.Atoi(submatches[2])
	if err != nil {
		return
	}
	v.VersionNumber[2], err = strconv.Atoi(submatches[3])
	if err != nil {
		return
	}
	v.Tag = submatches[4]
	v.Commit = submatches[5]

	return
}

func Less(v1, v2 Version) bool {
	if v1.VersionNumber[0] != v2.VersionNumber[0] {
		return v1.VersionNumber[0] < v2.VersionNumber[0]
	}
	if v1.VersionNumber[1] != v2.VersionNumber[1] {
		return v1.VersionNumber[1] < v2.VersionNumber[1]
	}
	if v1.VersionNumber[2] != v2.VersionNumber[2] {
		return v1.VersionNumber[2] < v2.VersionNumber[2]
	}
	if v1.Tag != v2.Tag {
		if v1.Tag == "" {
			// Final release always latest for version number
			return false
		}
		if v2.Tag == "" {
			return true
		}
		if v1.Tag == "dev" {
			// Dev branch is considered before a tag name is assigned
			return true
		}
		if strings.HasPrefix(v1.Tag, "rc") && !strings.HasPrefix(v2.Tag, "rc") {
			// rc is always last tag before final release
			return false
		}
		return v1.Tag < v2.Tag
	}

	// This is only for consistent sort order, not
	// for which version is newer/older. Need full commit
	// history to make decision if on same branch
	return v1.Commit < v2.Commit
}

var versionOutput = regexp.MustCompile(`Docker version ([a-z0-9-.]+), build ([a-f0-9]+(?:-dirty)?)`)

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

func StaticVersion(major, minor, release int) Version {
	return Version{
		Name:          fmt.Sprintf("v%d.%d.%d", major, minor, release),
		VersionNumber: [3]int{major, minor, release},
	}
}
