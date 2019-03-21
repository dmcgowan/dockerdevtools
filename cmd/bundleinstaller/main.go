package main

import (
	"flag"
	"path/filepath"

	"github.com/dmcgowan/dockerdevtools/buildutil"
	"github.com/sirupsen/logrus"
)

func main() {
	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		logrus.Fatal("Expecting source and target directory")
	}

	targetDir, err := filepath.Abs(args[len(args)-1])
	if err != nil {
		logrus.Fatal("Error resolving target directory %s: %v", args[len(args)-1], err)
	}
	targetDir = filepath.Clean(targetDir)

	for _, bundleDir := range args[:len(args)-1] {
		absDir, err := filepath.Abs(bundleDir)
		if err != nil {
			logrus.Fatalf("Error resolving directory %s: %v", bundleDir, err)
		}
		bundleDir = filepath.Clean(absDir)
		logrus.Infof("Copying %s to %s", bundleDir, targetDir)
		if err := buildutil.CopyBundleBinaries(bundleDir, targetDir); err != nil {
			logrus.Fatalf("Error copying binaries: %v", err)
		}
	}
}
