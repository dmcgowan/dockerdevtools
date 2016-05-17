package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dmcgowan/dockerdevtools/buildutil"
)

var (
	BuildScript = "make.sh"
	BuildType   = []string{"binary-client", "binary-daemon"}
)

func main() {
	var buildDir string
	var targetDir string
	var dynamic bool
	flag.StringVar(&targetDir, "t", "", "Directory to install files")
	flag.StringVar(&buildDir, "b", "", "Directory to build files")
	flag.BoolVar(&dynamic, "dynamic", false, "Whether to build a dynamic binary")
	flag.Parse()
	if dynamic {
		BuildType = []string{"dynbinary"}
	}

	if targetDir == "" {
		targetDir = filepath.Join(os.Getenv("HOME"), ".bin")
	}
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		log.Fatal("Target directory does not exist: %s", targetDir)
	} else if err != nil {
		log.Fatalf("Error calling stat on target dir: %s", err)
	}

	if buildDir == "" {
		var err error
		buildDir, err = ioutil.TempDir("/tmp", "docker-build-")
		if err != nil {
			log.Fatalf("Error creating temp dir: %s", err)
		}
		defer os.RemoveAll(buildDir)
	} else if _, err := os.Stat(buildDir); os.IsNotExist(err) {
		log.Fatalf("Build directory does not exist: %s", buildDir)
	} else if err != nil {
		log.Fatalf("Error calling stat on build dir: %s", err)
	}
	buildGoPath := buildDir
	buildDir = filepath.Join(buildDir, "src", "github.com", "docker", "docker")
	log.Printf("Building in %s", buildDir)

	packageCache := os.Getenv("DBUILDER_PACKAGE_CACHE")
	if packageCache != "" {
		os.Symlink(packageCache, filepath.Join(buildGoPath, "pkg"))
	}

	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		log.Fatal("Must set GOPATH to build Docker")
	}
	goroot := os.Getenv("GOROOT")
	if goroot == "" {
		log.Fatal("Must set GOROOT to build Docker")
	}

	dockerpath := filepath.Join(gopath, "src", "github.com", "docker", "docker")
	if _, err := os.Stat(dockerpath); os.IsNotExist(err) {
		log.Fatalf("Docker not found on path, go get or checkout in GOPATH at : %s", dockerpath)
	}

	buildscript := filepath.Join(dockerpath, "hack", BuildScript)
	if _, err := os.Stat(buildscript); os.IsNotExist(err) {
		log.Fatalf("Build script not found, ensure Docker is checked out correctly and up to date: missing %s", buildscript)
	}

	copyFile(filepath.Join(dockerpath, "VERSION"), filepath.Join(buildDir, "VERSION"))

	copyFileIfExists(filepath.Join(dockerpath, "dockerinit", "dockerinit.go"), filepath.Join(buildDir, "dockerinit", "dockerinit.go"))

	copyFileIfExists(filepath.Join(dockerpath, "dockerversion", "version_lib.go"), filepath.Join(buildDir, "dockerversion", "version_lib.go"))
	copyFileIfExists(filepath.Join(dockerpath, "dockerversion", "useragent.go"), filepath.Join(buildDir, "dockerversion", "useragent.go"))

	//git rev-parse HEAD
	gitCmd := exec.Command("git", "rev-parse", "HEAD")
	gitCmd.Dir = dockerpath
	b, err := gitCmd.Output()
	if err != nil {
		log.Fatalf("Error getting git HEAD: %s", err)
	}
	log.Printf("Git version: %s", b)

	buildCmd := exec.Command(buildscript, BuildType...)
	buildCmd.Dir = buildDir
	buildCmd.Env = []string{
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
		fmt.Sprintf("GOPATH=%s:%s:%s", buildGoPath, filepath.Join(dockerpath, "vendor"), gopath),
		fmt.Sprintf("GOROOT=%s", goroot),
		fmt.Sprintf("DOCKER_GITCOMMIT=%s", strings.TrimSpace(string(b))),
		"DOCKER_BUILDTAGS=exclude_graphdriver_devicemapper",
	}
	buildCmd.Stderr = os.Stderr

	out, err := buildCmd.Output()
	if err != nil {
		log.Fatalf("Build failure")
	}

	buildRegexp := regexp.MustCompile("Created binary:[[:space:]]+([[:graph:]]+)")

	log.Printf("Success, copying\n%s", out)

	matches := buildRegexp.FindAllSubmatch(out, 2)
	if len(matches) == 0 || len(matches) > 2 {
		log.Fatalf("Build failure: could not find binaries")
	}

	sourceDirs := map[string]struct{}{}
	for i := range matches {
		if len(matches[i]) != 2 {
			log.Fatal("Invalid match: %#v", matches[i])
		}
		file := string(matches[i][1])
		if !filepath.IsAbs(file) {
			file = filepath.Join(buildDir, file)
		}
		sourceDirs[filepath.Dir(file)] = struct{}{}
	}

	for sourceDir := range sourceDirs {
		log.Printf("Copying bundle directory %s to %s\n", sourceDir, targetDir)
		buildutil.CopyBundleBinaries(sourceDir, targetDir)
	}

}

func copyFileIfExists(source, dest string) {
	if _, err := os.Stat(source); err == nil {
		copyFile(source, dest)
	}
}

func copyFile(source, dest string) {
	if err := buildutil.CopyFile(source, dest, 0644); err != nil {
		log.Fatal(err)
	}
}
