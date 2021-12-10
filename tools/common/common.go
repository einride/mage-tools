package common

import (
	"fmt"

	"github.com/magefile/mage/sh"
)

func WireGenerate(path string) error {
	fmt.Println("[wire-generate] generating initializers...")
	return sh.RunV("go", "run", "-mod=mod", "github.com/google/wire/cmd/wire", "gen", path)
}

func MockgenGenerate(packageName, destination, moduleName, mocks string) error {
	fmt.Printf("[mockgen] generating %s...", packageName)
	return sh.RunV("go", "run", "-mod=mod", "github.com/golang/mock/mockgen", "-package", packageName, "-destination", destination, moduleName, mocks)
}

func GoTest() error {
	err := sh.RunV("go", "test", "-race", "-cover", "./...")
	if err != nil {
		return err
	}
	return nil
}

func GoModTidy() error {
	err := sh.RunV("go", "mod", "tidy", "-v")
	if err != nil {
		return err
	}
	return nil
}
