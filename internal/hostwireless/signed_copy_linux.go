//go:build linux

package hostwireless

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"golang.org/x/sys/unix"
)

func verifyKernelCMS(ctx context.Context, payload, signature, cert []byte) error {
	sigFile, err := anonymousCMSInput("signature", signature)
	if err != nil {
		return err
	}
	defer sigFile.Close()
	certFile, err := anonymousCMSInput("public-cert", cert)
	if err != nil {
		return err
	}
	defer certFile.Close()

	// -nointern makes the supplied certificate the sole signer source.
	// -noverify skips chain trust. CMS still verifies the signature and content.
	cmd := exec.CommandContext(ctx, "/usr/bin/openssl", "cms", "-verify", "-binary", "-inform", "DER",
		"-nointern", "-noverify", "-in", "/proc/self/fd/3", "-certfile", "/proc/self/fd/4",
		"-content", "/proc/self/fd/0", "-out", "/dev/null")
	cmd.Env = []string{"LC_ALL=C", "OPENSSL_CONF=/dev/null"}
	cmd.ExtraFiles = []*os.File{sigFile, certFile}
	cmd.Stdin = bytes.NewReader(payload)
	cmd.WaitDelay = time.Second
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("OpenSSL could not verify the module signature with the supplied certificate: %w", err)
	}
	return nil
}

func anonymousCMSInput(name string, data []byte) (*os.File, error) {
	fd, err := unix.MemfdCreate("vrooli-wireless-"+name, unix.MFD_CLOEXEC)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), name)
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return nil, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}
