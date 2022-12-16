package sgcloudspanner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/tools/sgdocker"
)

// Cloud Spanner Emulator versions can be found here,
// https://console.cloud.google.com/gcr/images/cloud-spanner-emulator/global/emulator
const (
	cloudbuildNetwork = "cloudbuild"
	url               = "gcr.io/cloud-spanner-emulator/emulator"
	version           = "sha256:5469945589399bd79ead8bed929f5eb4d1c5ee98d095df5b0ebe35f0b7160a84" // 1.4.8
	image             = url + "@" + version
)

// RunEmulator runs the Cloud Spanner emulator in Docker.
func RunEmulator(ctx context.Context) (_ func(), err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("run Cloud Spanner emulator: %w", err)
		}
	}()
	sg.Logger(ctx).Println("starting Cloud Spanner emulator...")
	if emulatorHost, ok := os.LookupEnv("SPANNER_EMULATOR_HOST"); ok {
		sg.Logger(ctx).Printf("a Cloud Spanner emulator is already running on %s", emulatorHost)
		return func() {}, nil
	}
	if !isDockerDaemonRunning(ctx) {
		return nil, fmt.Errorf("the Docker daemon does not seem to be running")
	}
	var dockerRunCmd *exec.Cmd
	if isRunningOnCloudBuild(ctx) {
		dockerRunCmd = sgdocker.Command(ctx, "run", "-d", "--network", cloudbuildNetwork, "--publish-all", image)
	} else {
		dockerRunCmd = sgdocker.Command(ctx, "run", "-d", "--publish-all", image)
	}
	var dockerRunStdout strings.Builder
	dockerRunCmd.Stdout = &dockerRunStdout
	if err := dockerRunCmd.Run(); err != nil {
		return nil, err
	}
	containerID := strings.TrimSpace(dockerRunStdout.String())
	cleanup := func() {
		sg.Logger(ctx).Println("stopping down Cloud Spanner emulator...")
		cmd := sgdocker.Command(ctx, "kill", containerID)
		cmd.Stdout, cmd.Stderr = nil, nil
		if err := cmd.Run(); err != nil {
			sg.Logger(ctx).Printf("failed to kill emulator container: %v", err)
		}
		cmd = sgdocker.Command(ctx, "rm", "-v", containerID)
		cmd.Stdout, cmd.Stderr = nil, nil
		if err := cmd.Run(); err != nil {
			sg.Logger(ctx).Printf("failed to remove emulator container: %v", err)
		}
		if err := os.Unsetenv("SPANNER_EMULATOR_HOST"); err != nil {
			sg.Logger(ctx).Printf("failed to unset SPANNER_EMULATOR_HOST: %v", err)
		}
	}
	var emulatorHost string
	if isRunningOnCloudBuild(ctx) {
		emulatorHost, err = inspectPortAddressCloudbuild(ctx, containerID, "9010/tcp", image)
		if err != nil {
			cleanup()
			return nil, err
		}
	} else {
		emulatorHost, err = inspectPortAddress(ctx, containerID, "9010/tcp")
		if err != nil {
			cleanup()
			return nil, err
		}
	}
	if err := os.Setenv("SPANNER_EMULATOR_HOST", emulatorHost); err != nil {
		cleanup()
		return nil, err
	}
	sg.Logger(ctx).Printf("running Cloud Spanner emulator on %s", emulatorHost)
	if err := awaitReachable(ctx, emulatorHost, 100*time.Millisecond, 10*time.Second); err != nil {
		cleanup()
		return nil, err
	}
	return cleanup, nil
}

func isDockerDaemonRunning(ctx context.Context) bool {
	cmd := sgdocker.Command(ctx, "info")
	cmd.Stdout, cmd.Stderr = nil, nil
	return cmd.Run() == nil
}

func inspectPortAddress(ctx context.Context, containerID, containerPort string) (string, error) {
	var stdout bytes.Buffer
	cmd := sgdocker.Command(ctx, "port", containerID, containerPort)
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}
	output := stdout.String()
	lines := strings.Split(output, "\n")
	// docker port can return ipv6 mapping as well, take the first non ipv6 mapping.
	for _, line := range lines {
		mapping := strings.TrimSpace(line)
		if _, err := net.ResolveTCPAddr("tcp4", mapping); err == nil {
			return mapping, nil
		}
	}
	return "", fmt.Errorf("no mapping found for %s in container %s", containerPort, containerID)
}

func inspectPortAddressCloudbuild(ctx context.Context, containerID, containerPort, image string) (string, error) {
	var stdout bytes.Buffer
	cmd := sgdocker.Command(ctx, "inspect", containerID)
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}
	var containers []struct {
		Config struct {
			Image string
		}
		NetworkSettings struct {
			Ports map[string][]struct {
				HostPort string
			}
			Networks map[string]struct {
				Gateway string
			}
		}
	}
	if err := json.NewDecoder(&stdout).Decode(&containers); err != nil {
		return "", err
	}
	var host, port string

	for _, container := range containers {
		if container.Config.Image == image {
			port = container.NetworkSettings.Ports[containerPort][0].HostPort // prefer first option
		}
		if network, ok := container.NetworkSettings.Networks[cloudbuildNetwork]; ok {
			host = network.Gateway
		}
	}
	if host == "" || port == "" {
		return "", fmt.Errorf("failed to inspect container %s for port %s", containerID, containerPort)
	}
	return fmt.Sprintf("%s:%s", host, port), nil
}

func awaitReachable(ctx context.Context, addr string, wait, maxWait time.Duration) error {
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		if c, err := net.Dial("tcp", addr); err == nil {
			_ = c.Close()
			return nil
		}
		sg.Logger(ctx).Printf("waiting %v for %s to become reachable...", wait, addr)
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return fmt.Errorf("%s was unreachable for %v", addr, maxWait)
}

func isRunningOnCloudBuild(ctx context.Context) bool {
	cmd := sgdocker.Command(ctx, "network", "inspect", cloudbuildNetwork)
	var stdout bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, nil
	return cmd.Run() == nil
}
