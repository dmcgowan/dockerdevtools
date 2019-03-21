package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/dmcgowan/dockerdevtools/buildutil"
	"github.com/dmcgowan/dockerdevtools/versionutil"
	"github.com/sirupsen/logrus"
)

func main() {
	var targetDir string
	var buildCache string
	var checkCache bool
	var useFile string
	var verbose bool
	flag.StringVar(&targetDir, "t", "", "Directory to install files")
	flag.StringVar(&buildCache, "bc", "", "Directory to cache builds")
	flag.BoolVar(&checkCache, "cc", false, "Whether to only do a cache check")
	flag.BoolVar(&verbose, "v", false, "Verbose logging")
	flag.StringVar(&useFile, "put", "", "Use the provided file instead of cache and put in cache")
	flag.Parse()
	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}

	version := "latest"
	if flag.NArg() > 1 {
		logrus.Fatalf("Can only install 1 version")
	}
	if flag.NArg() == 1 {
		version = flag.Arg(0)
	}
	if version == "latest" {
		// TODO: Support downloading from
		// "https://experimental.docker.com/builds/Linux/x86_64/docker-latest", nil
		logrus.Fatalf("Experiment build installs not yet supported")
	}
	if targetDir == "" {
		targetDir = filepath.Join(os.Getenv("HOME"), ".bin")
	}
	if buildCache == "" {
		var err error
		buildCache, err = ioutil.TempDir("/tmp", "docker-install-")
		if err != nil {
			logrus.Fatalf("Error creating temp dir: %s", err)
		}
	}

	v, err := versionutil.ParseVersion(version)
	if err != nil {
		logrus.Fatalf("Invalid version: %s", err)
	}

	c := buildutil.NewFSBuildCache(buildCache)
	if checkCache {
		// Only do a cache check
		if c.IsCached(v) {
			fmt.Println("cached")
			os.Exit(0)
		}
		fmt.Println("uncached")
		os.Exit(1)
	}
	if useFile != "" {
		logrus.Fatalf("Putting file not yet supported")
	}
	if err := c.InstallVersion(v, targetDir); err != nil {
		logrus.Fatalf("Error installing %s: %s", version, err)
	}

}
