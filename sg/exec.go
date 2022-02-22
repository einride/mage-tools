package sg

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Command should be used when returning exec.Cmd from tools to set opinionated standard fields.
func Command(ctx context.Context, path string, args ...string) *exec.Cmd {
	// TODO: use exec.CommandContext when we have determined there are no side-effects.
	cmd := exec.Command(path)
	cmd.Args = append(cmd.Args, args...)
	cmd.Dir = FromGitRoot(".")
	cmd.Env = prependPath(os.Environ(), FromBinDir())
	cmd.Stderr = newLogWriter(ctx, os.Stderr)
	cmd.Stdout = newLogWriter(ctx, os.Stdout)
	return cmd
}

func newLogWriter(ctx context.Context, out io.Writer) *logWriter {
	logger := log.New(out, Logger(ctx).Prefix(), 0)
	return &logWriter{logger: logger, out: out}
}

type logWriter struct {
	logger            *log.Logger
	out               io.Writer
	hasFileReferences bool
}

func (l *logWriter) Write(p []byte) (n int, err error) {
	in := bufio.NewScanner(bytes.NewReader(p))
	for in.Scan() {
		line := in.Text()
		if !l.hasFileReferences {
			l.hasFileReferences, line = hasFileReferenceAndEnsurePathRelativeToGitRoot(line)
			if l.hasFileReferences {
				// If line has file reference (e.g. lint errors), print empty line with logger prefix.
				// This enables GitHub to autodetect the file references and print them in the PR review.
				l.logger.Println()
			}
		}
		if l.hasFileReferences {
			// Prints line without logger prefix.
			_, _ = fmt.Fprintln(l.out, line)
		} else {
			l.logger.Print(line)
		}
	}
	if err := in.Err(); err != nil {
		l.logger.Fatal(err)
	}
	return len(p), nil
}

func hasFileReferenceAndEnsurePathRelativeToGitRoot(line string) (bool, string) {
	trimmedLine := strings.TrimSpace(line)
	if i := strings.IndexByte(trimmedLine, ':'); i > 0 {
		filePath, ok := ensurePathIsRelativeFromGitRoot(trimmedLine[:i])
		if !ok {
			return false, line
		}
		if _, err := os.Lstat(FromGitRoot(filePath)); err == nil {
			return true, filePath
		}
	}
	return false, line
}

func ensurePathIsRelativeFromGitRoot(path string) (string, bool) {
	subdir, err := filepath.Rel(FromGitRoot(), FromWorkDir())
	if err != nil {
		return path, false
	}
	if strings.HasPrefix(path, subdir+"/") {
		return path, true
	}
	// Prefix path with subdir to get path relative from git root
	return filepath.Join(subdir, path), true
}

// Output runs the given command, and returns all output from stdout in a neatly, trimmed manner,
// panicking if an error occurs.
func Output(cmd *exec.Cmd) string {
	cmd.Stdout = nil
	output, err := cmd.Output()
	if err != nil {
		panic(fmt.Sprintf("%s failed: %v", cmd.Path, err))
	}
	return strings.TrimSpace(string(output))
}

func prependPath(environ []string, paths ...string) []string {
	for i, kv := range environ {
		if !strings.HasPrefix(kv, "PATH=") {
			continue
		}
		environ[i] = fmt.Sprintf("PATH=%s:%s", strings.Join(paths, ":"), strings.TrimPrefix(kv, "PATH="))
		return environ
	}
	return append(environ, fmt.Sprintf("PATH=%s", strings.Join(paths, ":")))
}
