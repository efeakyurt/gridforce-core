package container

import (
	"bytes"
	"context"
	"io"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// RunContainer pulls an image (if needed), runs a container, waits for it, and returns logs.
func RunContainer(ctx context.Context, imageName string, cmd []string) (string, error) {
	// Initialize Docker client with fixed API version 1.44 as requested
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.44"))
	if err != nil {
		return "", err
	}
	defer cli.Close()

	// 1. Pull Image
	log.Printf("Pulling image %s...\n", imageName)
	reader, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return "", err
	}
	io.Copy(io.Discard, reader) // Consume pull output
	reader.Close()

	// 2. Create Container
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imageName,
		Cmd:   cmd,
	}, nil, nil, nil, "")
	if err != nil {
		return "", err
	}

	containerID := resp.ID
	defer func() {
		// 6. Cleanup
		if err := cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{}); err != nil {
			log.Printf("Failed to remove container %s: %v\n", containerID, err)
		}
	}()

	// 3. Start Container
	if err := cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}

	// 4. Wait for completion
	statusCh, errCh := cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return "", err
		}
	case <-statusCh:
	}

	// 5. Get Logs
	out, err := cli.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return "", err
	}
	defer out.Close()

	var stdout, stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, out); err != nil {
		return "", err
	}

	return stdout.String(), nil
}
