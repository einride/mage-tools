package sgconvco

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
)

const version = "0.3.7"

// nolint: gochecknoglobals
var commandPath string

func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, commandPath, args...)
}

func PrepareCommand(ctx context.Context) error {
	const toolName = "convco"
	binDir := sg.FromToolsDir(toolName, version)
	binary := filepath.Join(binDir, toolName)
	var hostOS string
	switch strings.Split(runtime.GOOS, "/")[0] {
	case "linux":
		hostOS = "ubuntu"
	case "darwin":
		hostOS = "macos"
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	binURL := fmt.Sprintf(
		"https://github.com/convco/convco/releases/download/v%s/convco-%s.zip",
		version,
		hostOS,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(binDir),
		sgtool.WithUnzip(),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", toolName, err)
	}
	commandPath = binary
	return os.Chmod(binary, 0o755)
}
