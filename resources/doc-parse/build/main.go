package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	resourceRoot, err := filepath.Abs(filepath.Join(".."))
	if err != nil {
		return err
	}
	artifactDir := os.Getenv("DOC_PARSE_ARTIFACT_DIR")
	if artifactDir == "" {
		artifactDir = filepath.Join(resourceRoot, "artifacts")
	}
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return fmt.Errorf("create artifact directory: %w", err)
	}
	if _, err := exec.LookPath("cargo"); err == nil {
		if _, err := exec.LookPath("rustup"); err == nil {
			if err := command(resourceRoot, "rustup", "target", "add", "wasm32-wasip1"); err != nil {
				return err
			}
		}
		if err := command(resourceRoot, "cargo", "build", "--locked", "--release", "--target", "wasm32-wasip1"); err != nil {
			return err
		}
	} else {
		if _, err := exec.LookPath("docker"); err != nil {
			return errors.New("doc-parse build requires cargo/rustup or Docker")
		}
		image := os.Getenv("DOC_PARSE_RUST_IMAGE")
		if image == "" {
			image = "rust:1.89.0"
		}
		workspace, err := filepath.Abs(resourceRoot)
		if err != nil {
			return err
		}
		if err := command(resourceRoot, "docker", "run", "--rm", "-v", workspace+":/workspace", "-w", "/workspace", image, "/usr/local/cargo/bin/rustup", "target", "add", "wasm32-wasip1"); err != nil {
			return err
		}
		if err := command(resourceRoot, "docker", "run", "--rm", "-v", workspace+":/workspace", "-w", "/workspace", image, "/usr/local/cargo/bin/cargo", "build", "--locked", "--release", "--target", "wasm32-wasip1"); err != nil {
			return err
		}
	}
	module := filepath.Join(resourceRoot, "target", "wasm32-wasip1", "release", "vrooli-doc-parse-shim.wasm")
	data, err := os.ReadFile(module)
	if err != nil {
		return fmt.Errorf("read build output: %w", err)
	}
	output := filepath.Join(artifactDir, "doc-parse.wasm")
	if err := os.WriteFile(output, data, 0o755); err != nil {
		return fmt.Errorf("write artifact: %w", err)
	}
	digest := sha256.Sum256(data)
	digestText := hex.EncodeToString(digest[:]) + "  doc-parse.wasm\n"
	if err := os.WriteFile(output+".sha256", []byte(digestText), 0o644); err != nil {
		return fmt.Errorf("write checksum: %w", err)
	}
	fmt.Printf("built %s (%s)\n", output, hex.EncodeToString(digest[:]))
	return nil
}

func command(dir, name string, args ...string) error {
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CARGO_TERM_COLOR=never")
	if runtime.GOOS == "windows" {
		cmd.Env = append(cmd.Env, "RUSTFLAGS=-C target-feature=+crt-static")
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr := new(exec.ExitError); errors.As(err, &exitErr) {
			return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), exitErr)
		}
		return err
	}
	return nil
}
