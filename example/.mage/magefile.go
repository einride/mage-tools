//go:build mage
// +build mage

package main

import (
	"context"

	"github.com/magefile/mage/mg"
	"go.einride.tech/mage-tools/mglog"
	"go.einride.tech/mage-tools/mgmake"
	"go.einride.tech/mage-tools/mgpath"
	"go.einride.tech/mage-tools/targets/mggitverifynodiff"
	"go.einride.tech/mage-tools/targets/mgyamlfmt"
	"go.einride.tech/mage-tools/tools/mgconvco"
	"go.einride.tech/mage-tools/tools/mggo"
	"go.einride.tech/mage-tools/tools/mggolangcilint"
	"go.einride.tech/mage-tools/tools/mggoreview"
	"go.einride.tech/mage-tools/tools/mgmarkdownfmt"
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
		ConvcoCheck,
		GolangciLint,
		Goreview,
		GoTest,
		FormatMarkdown,
	)
	mg.SerialDeps(
		GoModTidy,
		mggitverifynodiff.GitVerifyNoDiff,
	)
}

func FormatYaml() error {
	return mgyamlfmt.FormatYAML()
}

func GoModTidy() error {
	mglog.Logger("go-mod-tidy").Info("tidying Go module files...")
	return mggo.GoModTidy().Run()
}

func Goreview(ctx context.Context) error {
	mglog.Logger("goreview").Info("running...")
	return mggoreview.Command(ctx, "-c", "1", "./...").Run()
}

func GoTest() error {
	mglog.Logger("go-test").Info("running Go unit tests..")
	return mggo.GoTest().Run()
}

func GolangciLint(ctx context.Context) error {
	mglog.Logger("golangci-lint").Info("running...")
	return mggolangcilint.LintCommand(ctx).Run()
}

func FormatMarkdown(ctx context.Context) error {
	mglog.Logger("format-markdown").Info("formatting..")
	return mgmarkdownfmt.Command(ctx, "-w", ".").Run()
}

func ConvcoCheck(ctx context.Context) error {
	mglog.Logger("convco-check").Info("checking...")
	return mgconvco.Command(ctx, "check", "origin/master..HEAD").Run()
}
