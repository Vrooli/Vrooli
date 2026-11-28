//go:build linux

package hostwireless

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func queryMOKCertificate(ctx context.Context, der []byte) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cert, err := anonymousCMSInput("public-trust-cert", der)
	if err != nil {
		return "", err
	}
	defer cert.Close()
	cmd := exec.CommandContext(ctx, "/usr/bin/mokutil", "--test-key", "/proc/self/fd/3")
	cmd.Env = []string{"LC_ALL=C"}
	cmd.ExtraFiles = []*os.File{cert}
	cmd.WaitDelay = time.Second
	var output mokStatusOutput
	cmd.Stdout, cmd.Stderr = &output, &output
	err = cmd.Run()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if output.exceeded {
		return "", fmt.Errorf("mokutil response exceeds status bound")
	}
	return output.String(), err
}

type mokStatusOutput struct {
	bytes.Buffer
	exceeded bool
}

func (b *mokStatusOutput) Write(data []byte) (int, error) {
	if len(data) > 4096-b.Len() {
		b.exceeded = true
		return 0, fmt.Errorf("mokutil response exceeds status bound")
	}
	return b.Buffer.Write(data)
}
