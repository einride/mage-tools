//go:build mage
// +build mage

package main

import (
	"github.com/magefile/mage/mg"
	"go.einride.tech/mage-tools/mgmake"
	"go.einride.tech/mage-tools/mgpath"

	// mage:import
	"go.einride.tech/mage-tools/targets/mgcocogitto"

	// mage:import
	"go.einride.tech/mage-tools/targets/mggitverifynodiff"
)

func init() {
	mgmake.GenerateMakefiles(
		mgmake.Makefile{
			Path:          mgpath.FromGitRoot("Makefile"),
			DefaultTarget: All,
		},
	)
}

func All() {
	mg.Deps(
		mgcocogitto.CogCheck,
	)
	mg.SerialDeps(
		mggitverifynodiff.GitVerifyNoDiff,
	)
}
