package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// resource-android-sdk is intentionally small: lifecycle authority remains in
// the Vrooli resource driver while this binary provides a governed, inspectable
// host-tool preflight for operators and CI.
func main() {
	command := "status"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	if command == "version" {
		fmt.Println("resource-android-sdk 1")
		return
	}
	if command == "install" {
		if err := install(); err != nil {
			fmt.Fprintln(os.Stderr, "android-sdk install failed:", err)
			os.Exit(1)
		}
		return
	}
	path, err := exec.LookPath("adb")
	if err != nil {
		fmt.Fprintln(os.Stderr, "adb unavailable: install/start the android-sdk resource")
		os.Exit(1)
	}
	fmt.Println(path)
}

func install() error {
	if _, err := exec.LookPath("adb"); err == nil {
		version, versionErr := exec.Command("adb", "version").CombinedOutput()
		if versionErr == nil {
			fmt.Printf("android-sdk already available: %s", version)
			return nil
		}
	}
	url := os.Getenv("ANDROID_PLATFORM_TOOLS_URL")
	if url == "" {
		url = "https://dl.google.com/android/repository/platform-tools-latest-" + runtime.GOOS + ".zip"
	}
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download %s returned HTTP %d", url, resp.StatusCode)
	}
	tmp, err := os.CreateTemp("", "android-platform-tools-*.zip")
	if err != nil {
		return fmt.Errorf("create download: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, hash), resp.Body); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("download archive: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close archive: %w", err)
	}
	digest := hex.EncodeToString(hash.Sum(nil))
	if expected := strings.ToLower(strings.TrimSpace(os.Getenv("ANDROID_PLATFORM_TOOLS_SHA256"))); expected != "" && expected != digest {
		return fmt.Errorf("archive checksum mismatch: expected %s, got %s", expected, digest)
	}
	target, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve operator home directory: %w", err)
	}
	vrooliRoot := filepath.Join(target, ".vrooli")
	installRoot := filepath.Join(vrooliRoot, "resources", "android-sdk")
	if err := os.MkdirAll(installRoot, 0o700); err != nil {
		return fmt.Errorf("create install directory: %w", err)
	}
	archive, err := zip.OpenReader(tmpName)
	if err != nil {
		return fmt.Errorf("open platform-tools archive: %w", err)
	}
	defer archive.Close()
	stage, err := os.MkdirTemp(installRoot, ".stage-")
	if err != nil {
		return fmt.Errorf("create staging directory: %w", err)
	}
	defer os.RemoveAll(stage)
	for _, file := range archive.File {
		name := filepath.Clean(file.Name)
		if name == "." || strings.HasPrefix(name, ".."+string(os.PathSeparator)) || filepath.IsAbs(name) {
			return fmt.Errorf("archive contains unsafe path %q", file.Name)
		}
		path := filepath.Join(stage, name)
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0o700); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return err
		}
		in, err := file.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o700)
		if err == nil {
			_, err = io.Copy(out, in)
			_ = out.Close()
		}
		_ = in.Close()
		if err != nil {
			return fmt.Errorf("extract %s: %w", file.Name, err)
		}
	}
	platformTools := filepath.Join(stage, "platform-tools")
	adbPath := filepath.Join(platformTools, "adb")
	if runtime.GOOS == "windows" {
		adbPath += ".exe"
	}
	if _, err := os.Stat(adbPath); err != nil {
		return fmt.Errorf("archive did not contain platform-tools/%s", filepath.Base(adbPath))
	}
	final := filepath.Join(installRoot, "platform-tools")
	_ = os.RemoveAll(final)
	if err := os.Rename(platformTools, final); err != nil {
		return fmt.Errorf("activate platform-tools: %w", err)
	}
	binDir := filepath.Join(vrooliRoot, "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		return err
	}
	link := filepath.Join(binDir, filepath.Base(adbPath))
	_ = os.Remove(link)
	if err := os.Symlink(filepath.Join(final, filepath.Base(adbPath)), link); err != nil && runtime.GOOS != "windows" {
		return fmt.Errorf("expose adb in %s: %w", binDir, err)
	}
	version, err := exec.Command(filepath.Join(final, filepath.Base(adbPath)), "version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("installed adb failed validation: %s: %w", strings.TrimSpace(string(version)), err)
	}
	fmt.Printf("installed android-sdk platform-tools (%s)\n", strings.TrimSpace(string(version)))
	return nil
}
