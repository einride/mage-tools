package mgtool

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/sh"
)

const (
	AMD64 = "amd64"
	X8664 = "x86_64"
)

// Path This should only be used to set a custom value.
// Targets should use path() instead which performs
// validation on whether a path is set.
var mgToolPath = GetGitRootPath(".tools")

func GetCWDPath(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return filepath.Join(cwd, path)
}

func GetGitRootPath(path string) string {
	root, err := sh.Output("git", "rev-parse", "--show-toplevel")
	if err != nil {
		panic(err)
	}
	return filepath.Join(root, path)
}

func GetPath() string {
	if mgToolPath == "" {
		panic("No tools path set")
	}
	return mgToolPath
}

func SetPath(p string) {
	mgToolPath = p
}

func IsSupportedVersion(versions []string, version string, name string) error {
	for _, a := range versions {
		if a == version {
			return nil
		}
	}
	return fmt.Errorf(
		"the following %s versions are supported: %s",
		name,
		strings.Join(versions, ", "),
	)
}
