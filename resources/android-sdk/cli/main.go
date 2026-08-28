package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vrooli/envkit-go"
	"time"
)

const (
	resourceVersion  = "2"
	defaultAPI       = "36"
	defaultCLIURL    = "https://dl.google.com/android/repository/commandlinetools-%s-11076708_latest.zip"
	defaultGradleURL = "https://services.gradle.org/distributions/gradle-8.10.2-bin.zip"
	defaultJDKURL    = "https://api.adoptium.net/v3/binary/version/jdk-17.0.15%%2B6/%s/%s/jdk/hotspot/normal/eclipse"
	defaultGradleSHA = "31c55713e40233a8303827ceb42ca48a47267a0ad4bab9177123121e71524c26"
	defaultJDKSHA    = "9616877c733c9249328ea9bd83a5c8c30e0f9a7af180cac8ffda9034161c2df2"
)

// InstallPlan is intentionally pure data so the resource can be tested
// without downloading a multi-gigabyte Android image.
type InstallPlan struct {
	PlatformToolsURL string
	CommandLineURL   string
	PlatformPackage  string
	BuildTools       string
	EmulatorPackage  string
	SystemImage      string
	PlatformLevel    string
	JDKURL           string
	JDKSHA256        string
	JDKArchiveFormat string
	GradleURL        string
	GradleSHA256     string
}

type sdkLayout struct {
	Root          string
	SDKRoot       string
	PlatformTools string
	CmdlineTools  string
	BinDir        string
	ToolchainRoot string
	JDKRoot       string
	GradleRoot    string
}

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		fmt.Println("resource-android-sdk: install, status, kvm-check, avd-create, avd-start, avd-stop, avd-delete, toolchain-install, version")
		return
	}
	command := "status"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	var err error
	switch command {
	case "version":
		fmt.Println("resource-android-sdk", resourceVersion)
		return
	case "install":
		err = install()
	case "status":
		err = status()
	case "kvm-check":
		err = reportKVM()
	case "avd-create":
		err = avdCreate(os.Getenv("AVD_NAME"), os.Getenv("SYSTEM_IMAGE"))
	case "avd-start":
		err = avdStart(os.Getenv("AVD_NAME"))
	case "avd-stop":
		err = avdStop()
	case "avd-delete":
		err = avdDelete(os.Getenv("AVD_NAME"))
	case "toolchain-install":
		err = installHostToolchain()
	default:
		err = fmt.Errorf("unknown command %q (want install, status, kvm-check, avd-create, avd-start, avd-stop, avd-delete, toolchain-install, or version)", command)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "android-sdk", command, "failed:", err)
		os.Exit(1)
	}
}

func installPlan() (InstallPlan, error) {
	api := strings.TrimSpace(os.Getenv("ANDROID_API_LEVEL"))
	if api == "" {
		api = defaultAPI
	}
	if api != defaultAPI && api < defaultAPI {
		return InstallPlan{}, fmt.Errorf("ANDROID_API_LEVEL %s is below the Play target API floor %s", api, defaultAPI)
	}
	arch := runtime.GOARCH
	imageArch := map[string]string{"amd64": "x86_64", "arm64": "arm64-v8a"}[arch]
	if imageArch == "" {
		return InstallPlan{}, fmt.Errorf("no Android system image mapping for host architecture %s", arch)
	}
	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "mac"
	}
	if osName == "windows" {
		osName = "win"
	}
	platformURL := os.Getenv("ANDROID_PLATFORM_TOOLS_URL")
	if platformURL == "" {
		platformURL = "https://dl.google.com/android/repository/platform-tools-latest-" + runtime.GOOS + ".zip"
	}
	cliURL := os.Getenv("ANDROID_CMDLINE_TOOLS_URL")
	if cliURL == "" {
		cliURL = fmt.Sprintf(defaultCLIURL, osName)
	}
	jdkURL := strings.TrimSpace(os.Getenv("ANDROID_JDK_URL"))
	if jdkURL == "" {
		jdkURL = fmt.Sprintf(defaultJDKURL, runtime.GOOS, map[string]string{"amd64": "x64", "arm64": "aarch64"}[arch])
	}
	jdkFormat := "tar.gz"
	if runtime.GOOS == "windows" {
		jdkFormat = "zip"
	}
	jdkSHA := strings.TrimSpace(os.Getenv("ANDROID_JDK_SHA256"))
	if jdkSHA == "" && runtime.GOOS == "linux" && arch == "amd64" {
		jdkSHA = defaultJDKSHA
	}
	gradleURL := strings.TrimSpace(os.Getenv("ANDROID_GRADLE_URL"))
	if gradleURL == "" {
		gradleURL = defaultGradleURL
	}
	gradleSHA := strings.TrimSpace(os.Getenv("ANDROID_GRADLE_SHA256"))
	if gradleSHA == "" {
		gradleSHA = defaultGradleSHA
	}
	return InstallPlan{
		PlatformToolsURL: platformURL,
		CommandLineURL:   cliURL,
		PlatformPackage:  "platforms;android-" + api,
		BuildTools:       "build-tools;35.0.0",
		EmulatorPackage:  "emulator",
		SystemImage:      "system-images;android-" + api + ";google_apis;" + imageArch,
		PlatformLevel:    api,
		JDKURL:           jdkURL,
		JDKSHA256:        jdkSHA,
		JDKArchiveFormat: jdkFormat,
		GradleURL:        gradleURL,
		GradleSHA256:     gradleSHA,
	}, nil
}

func install() error {
	plan, err := installPlan()
	if err != nil {
		return err
	}
	layout, err := currentLayout()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(layout.Root, 0o700); err != nil {
		return fmt.Errorf("create install root: %w", err)
	}

	// A custom platform-tools URL is useful for hermetic unit tests and
	// mirrors. It intentionally does not imply that the rest of the SDK is
	// present; production installs always use the complete bootstrap below.
	if os.Getenv("ANDROID_SDK_SKIP_COMPONENTS") == "1" {
		if err := installArchive(plan.PlatformToolsURL, os.Getenv("ANDROID_PLATFORM_TOOLS_SHA256"), layout.PlatformTools, "platform-tools", "platform-tools", "zip"); err != nil {
			return err
		}
	} else {
		if err := installArchive(plan.CommandLineURL, os.Getenv("ANDROID_CMDLINE_TOOLS_SHA256"), layout.CmdlineTools, "cmdline-tools", "cmdline-tools", "zip"); err != nil {
			return err
		}
		if err := acceptLicensesAndInstall(layout, plan); err != nil {
			return err
		}
		if err := installHostToolchainWithPlan(layout, plan); err != nil {
			return err
		}
	}
	if err := exposeBinaries(layout); err != nil {
		return err
	}
	if err := validateTools(layout); err != nil {
		return err
	}
	fmt.Printf("installed android-sdk: api=%s system_image=%s sdk_root=%s jdk=%s gradle=%s\n", plan.PlatformLevel, plan.SystemImage, layout.SDKRoot, layout.JDKRoot, layout.GradleRoot)
	return nil
}

func installHostToolchain() error {
	plan, err := installPlan()
	if err != nil {
		return err
	}
	layout, err := currentLayout()
	if err != nil {
		return err
	}
	if err := installHostToolchainWithPlan(layout, plan); err != nil {
		return err
	}
	if err := exposeBinaries(layout); err != nil {
		return err
	}
	return validateTools(layout)
}

func installHostToolchainWithPlan(layout sdkLayout, plan InstallPlan) error {
	if err := installArchive(plan.JDKURL, plan.JDKSHA256, layout.JDKRoot, "JDK 17", "", plan.JDKArchiveFormat); err != nil {
		return err
	}
	if err := installArchive(plan.GradleURL, plan.GradleSHA256, layout.GradleRoot, "Gradle 8.10.2", "", "zip"); err != nil {
		return err
	}
	return nil
}

func currentLayout() (sdkLayout, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return sdkLayout{}, fmt.Errorf("resolve operator home directory: %w", err)
	}
	root := filepath.Join(home, ".vrooli", "resources", "android-sdk")
	if configured := strings.TrimSpace(os.Getenv("ANDROID_SDK_ROOT")); configured != "" {
		toolchainRoot := filepath.Join(root, "toolchains")
		return sdkLayout{Root: root, SDKRoot: configured, PlatformTools: filepath.Join(configured, "platform-tools"), CmdlineTools: filepath.Join(configured, "cmdline-tools", "latest"), BinDir: filepath.Join(home, ".vrooli", "bin"), ToolchainRoot: toolchainRoot, JDKRoot: filepath.Join(toolchainRoot, "jdk-17"), GradleRoot: filepath.Join(toolchainRoot, "gradle-8.10.2")}, nil
	}
	toolchainRoot := filepath.Join(root, "toolchains")
	return sdkLayout{Root: root, SDKRoot: root, PlatformTools: filepath.Join(root, "platform-tools"), CmdlineTools: filepath.Join(root, "cmdline-tools", "latest"), BinDir: filepath.Join(home, ".vrooli", "bin"), ToolchainRoot: toolchainRoot, JDKRoot: filepath.Join(toolchainRoot, "jdk-17"), GradleRoot: filepath.Join(toolchainRoot, "gradle-8.10.2")}, nil
}

func installArchive(source, expectedDigest, destination, component, archiveRoot, format string) error {
	if strings.TrimSpace(source) == "" {
		return fmt.Errorf("%s archive URL is empty", component)
	}
	resp, err := http.Get(source)
	if err != nil {
		return fmt.Errorf("download %s: %w", component, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download %s returned HTTP %d", component, resp.StatusCode)
	}
	tmp, err := os.CreateTemp("", "android-sdk-*.zip")
	if err != nil {
		return fmt.Errorf("create %s download: %w", component, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	digest := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, digest), resp.Body); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("download %s archive: %w", component, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close %s archive: %w", component, err)
	}
	actual := hex.EncodeToString(digest.Sum(nil))
	if expected := strings.ToLower(strings.TrimSpace(expectedDigest)); expected != "" && expected != actual {
		return fmt.Errorf("%s archive checksum mismatch: expected %s, got %s", component, expected, actual)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return fmt.Errorf("prepare %s destination: %w", component, err)
	}
	stage, err := os.MkdirTemp(filepath.Dir(destination), ".android-sdk-stage-")
	if err != nil {
		return fmt.Errorf("stage %s: %w", component, err)
	}
	defer os.RemoveAll(stage)
	if err := extractDownloadedArchive(tmpName, stage, format); err != nil {
		return fmt.Errorf("extract %s: %w", component, err)
	}
	sourceRoot := filepath.Join(stage, archiveRoot)
	if archiveRoot == "" {
		sourceRoot, err = singleArchiveRoot(stage)
		if err != nil {
			return fmt.Errorf("resolve %s archive root: %w", component, err)
		}
	}
	if _, err := os.Stat(sourceRoot); err != nil {
		return fmt.Errorf("%s archive did not contain %s: %w", component, archiveRoot, err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return err
	}
	_ = os.RemoveAll(destination)
	if err := os.Rename(sourceRoot, destination); err != nil {
		return fmt.Errorf("activate %s: %w", component, err)
	}
	return nil
}

func extractDownloadedArchive(path, stage, format string) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "zip":
		archive, err := zip.OpenReader(path)
		if err != nil {
			return err
		}
		defer archive.Close()
		return extractArchive(archive, stage)
	case "tar.gz", "tgz":
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		compressed, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer compressed.Close()
		return extractTarArchive(tar.NewReader(compressed), stage)
	default:
		return fmt.Errorf("unsupported archive format %q", format)
	}
}

func singleArchiveRoot(stage string) (string, error) {
	entries, err := os.ReadDir(stage)
	if err != nil {
		return "", err
	}
	if len(entries) != 1 || !entries[0].IsDir() {
		return "", fmt.Errorf("archive must contain exactly one top-level directory")
	}
	return filepath.Join(stage, entries[0].Name()), nil
}

func extractTarArchive(reader *tar.Reader, stage string) error {
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		clean := filepath.Clean(header.Name)
		if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("archive contains unsafe path %q", header.Name)
		}
		path := filepath.Join(stage, clean)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0o700); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				return err
			}
			out, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o700)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, reader)
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		case tar.TypeSymlink:
			if err := validateArchiveLink(clean, header.Linkname); err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				return err
			}
			_ = os.Remove(path)
			if err := os.Symlink(header.Linkname, path); err != nil {
				return err
			}
		case tar.TypeLink:
			link, err := archivePath(stage, header.Linkname)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				return err
			}
			if err := os.Link(link, path); err != nil {
				return err
			}
		}
	}
}

func validateArchiveLink(name, target string) error {
	if filepath.IsAbs(target) {
		return fmt.Errorf("archive link %q points outside extraction root", name)
	}
	resolved := filepath.Clean(filepath.Join(filepath.Dir(name), target))
	if resolved == ".." || strings.HasPrefix(resolved, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("archive link %q points outside extraction root", name)
	}
	return nil
}

func archivePath(stage, name string) (string, error) {
	clean := filepath.Clean(name)
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("archive contains unsafe link %q", name)
	}
	return filepath.Join(stage, clean), nil
}

func extractArchive(archive *zip.ReadCloser, stage string) error {
	for _, file := range archive.File {
		clean := filepath.Clean(file.Name)
		if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("archive contains unsafe path %q", file.Name)
		}
		path := filepath.Join(stage, clean)
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
	return nil
}

func acceptLicensesAndInstall(layout sdkLayout, plan InstallPlan) error {
	sdkmanager := filepath.Join(layout.CmdlineTools, "bin", executable("sdkmanager"))
	if _, err := os.Stat(sdkmanager); err != nil {
		return fmt.Errorf("sdkmanager missing after cmdline-tools install: %w", err)
	}
	args := []string{"--sdk_root=" + layout.SDKRoot, "platform-tools", plan.PlatformPackage, plan.BuildTools, plan.EmulatorPackage, plan.SystemImage}
	cmd := exec.Command(sdkmanager, args...)
	cmd.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.Resource, envkit.Env{"ANDROID_SDK_ROOT=" + layout.SDKRoot})
	cmd.Stdin = bytes.NewBufferString(strings.Repeat("y\n", 128))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sdkmanager install failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func exposeBinaries(layout sdkLayout) error {
	if err := os.MkdirAll(layout.BinDir, 0o700); err != nil {
		return err
	}
	paths := map[string]string{"adb": filepath.Join(layout.PlatformTools, executable("adb"))}
	if os.Getenv("ANDROID_SDK_SKIP_COMPONENTS") != "1" {
		paths["sdkmanager"] = filepath.Join(layout.CmdlineTools, "bin", executable("sdkmanager"))
		paths["avdmanager"] = filepath.Join(layout.CmdlineTools, "bin", executable("avdmanager"))
		paths["emulator"] = filepath.Join(layout.SDKRoot, "emulator", executable("emulator"))
		paths["java"] = filepath.Join(layout.JDKRoot, "bin", executable("java"))
		paths["javac"] = filepath.Join(layout.JDKRoot, "bin", executable("javac"))
	}
	for name, source := range paths {
		if _, err := os.Stat(source); err != nil {
			return fmt.Errorf("%s missing from SDK: %w", name, err)
		}
		link := filepath.Join(layout.BinDir, executable(name))
		_ = os.Remove(link)
		if runtime.GOOS == "windows" {
			// Windows does not provide a portable symlink without elevated
			// privileges; the resource exposes the SDK directory through PATH.
			continue
		}
		if err := os.Symlink(source, link); err != nil {
			return fmt.Errorf("expose %s: %w", name, err)
		}
	}
	if os.Getenv("ANDROID_SDK_SKIP_COMPONENTS") != "1" {
		if err := exposeGradle(layout); err != nil {
			return err
		}
	}
	return nil
}

func exposeGradle(layout sdkLayout) error {
	gradle := filepath.Join(layout.GradleRoot, "bin", executable("gradle"))
	if _, err := os.Stat(gradle); err != nil {
		return fmt.Errorf("gradle missing from toolchain: %w", err)
	}
	path := filepath.Join(layout.BinDir, executable("gradle"))
	if runtime.GOOS == "windows" {
		content := "@echo off\r\nset JAVA_HOME=" + layout.JDKRoot + "\r\n\"" + gradle + "\" %*\r\n"
		if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
			return fmt.Errorf("expose gradle: %w", err)
		}
		return nil
	}
	content := "#!/bin/sh\nexport JAVA_HOME=\"" + strings.ReplaceAll(layout.JDKRoot, "\"", "\\\"") + "\"\nexport PATH=\"$JAVA_HOME/bin:$PATH\"\nexec \"" + strings.ReplaceAll(gradle, "\"", "\\\"") + "\" \"$@\"\n"
	if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
		return fmt.Errorf("expose gradle: %w", err)
	}
	return nil
}

func validateTools(layout sdkLayout) error {
	names := []string{"adb"}
	if os.Getenv("ANDROID_SDK_SKIP_COMPONENTS") != "1" {
		names = append(names, "sdkmanager", "avdmanager", "emulator", "java", "javac", "gradle")
	}
	for _, name := range names {
		path := filepath.Join(layout.BinDir, executable(name))
		if runtime.GOOS == "windows" {
			path = filepath.Join(layout.SDKRoot, toolRelativePath(name))
		}
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("%s unavailable after install: %w", name, err)
		}
	}
	adb := filepath.Join(layout.PlatformTools, executable("adb"))
	output, err := exec.Command(adb, "version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("installed adb failed validation: %s: %w", strings.TrimSpace(string(output)), err)
	}
	if os.Getenv("ANDROID_SDK_SKIP_COMPONENTS") != "1" {
		for _, command := range []string{"javac", "gradle"} {
			probe := exec.Command(filepath.Join(layout.BinDir, executable(command)), "--version")
			probe.Env = sdkEnvironment(layout)
			if output, err := probe.CombinedOutput(); err != nil {
				return fmt.Errorf("installed %s failed validation: %s: %w", command, strings.TrimSpace(string(output)), err)
			}
		}
	}
	return nil
}

func status() error {
	layout, err := currentLayout()
	if err != nil {
		return err
	}
	names := []string{"adb", "sdkmanager", "avdmanager", "emulator", "java", "javac", "gradle"}
	if os.Getenv("ANDROID_SDK_SKIP_COMPONENTS") == "1" {
		names = []string{"adb"}
	}
	for _, name := range names {
		path := filepath.Join(layout.BinDir, executable(name))
		if _, statErr := os.Stat(path); statErr != nil {
			fmt.Printf("%s: unavailable (run vrooli resource install android-sdk)\n", name)
			continue
		}
		fmt.Printf("%s: %s\n", name, path)
	}
	present, writable, reason := kvmStatus()
	fmt.Printf("kvm: present=%t writable=%t reason=%s\n", present, writable, reason)
	if _, err := os.Stat(filepath.Join(layout.BinDir, executable("adb"))); err != nil {
		return errors.New("android-sdk is not installed")
	}
	return nil
}

func reportKVM() error {
	present, writable, reason := kvmStatus()
	fmt.Printf("kvm: present=%t writable=%t reason=%s\n", present, writable, reason)
	if !present || !writable {
		return fmt.Errorf("/dev/kvm unavailable: %s; next action: enable KVM access or use a capable bridge host", reason)
	}
	return nil
}

func kvmStatus() (bool, bool, string) {
	if runtime.GOOS != "linux" {
		return false, false, "KVM probe is only supported on Linux hosts"
	}
	info, err := os.Stat("/dev/kvm")
	if err != nil {
		return false, false, "device is absent"
	}
	if info.Mode()&os.ModeDevice == 0 {
		return true, false, "path exists but is not a device"
	}
	file, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0)
	if err != nil {
		return true, false, "device exists but is not writable"
	}
	_ = file.Close()
	return true, true, "device is present and writable"
}

func avdCreate(name, image string) error {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(image) == "" {
		return errors.New("AVD_NAME and SYSTEM_IMAGE are required")
	}
	layout, err := currentLayout()
	if err != nil {
		return err
	}
	return runSDKToolInput(context.Background(), layout, "no\n", filepath.Join(layout.CmdlineTools, "bin", executable("avdmanager")), "create", "avd", "--name", name, "--package", image, "--force")
}

func avdStart(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("AVD_NAME is required")
	}
	if err := reportKVM(); err != nil {
		return err
	}
	layout, err := currentLayout()
	if err != nil {
		return err
	}
	emulator := filepath.Join(layout.SDKRoot, "emulator", executable("emulator"))
	cmd := exec.Command(emulator, "-avd", name, "-no-snapshot", "-no-audio", "-no-boot-anim", "-no-window")
	cmd.Env = sdkEnvironment(layout)
	cmd.SysProcAttr = androidSDKDetachedProcessAttrs()
	logPath := filepath.Join(layout.Root, "avd", name+".log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o700); err != nil {
		return fmt.Errorf("prepare emulator log directory: %w", err)
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open emulator log: %w", err)
	}
	defer logFile.Close()
	cmd.Stdout, cmd.Stderr = logFile, logFile
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start emulator: %w", err)
	}
	if err := waitForBoot(context.Background(), "emulator-5554", 180*time.Second); err != nil {
		return err
	}
	fmt.Println("avd ready:", name)
	return nil
}

func avdStop() error {
	layout, err := currentLayout()
	if err != nil {
		return err
	}
	return runSDKTool(context.Background(), layout, filepath.Join(layout.PlatformTools, executable("adb")), "emu", "kill")
}

func avdDelete(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("AVD_NAME is required")
	}
	layout, err := currentLayout()
	if err != nil {
		return err
	}
	return runSDKTool(context.Background(), layout, filepath.Join(layout.CmdlineTools, "bin", executable("avdmanager")), "delete", "avd", "--name", name)
}

func waitForBoot(ctx context.Context, serial string, timeout time.Duration) error {
	layout, err := currentLayout()
	if err != nil {
		return err
	}
	adb := filepath.Join(layout.PlatformTools, executable("adb"))
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		waitErr := runSDKTool(probeCtx, layout, adb, "-s", serial, "wait-for-device")
		cancel()
		if waitErr == nil {
			propertyCtx, propertyCancel := context.WithTimeout(ctx, 5*time.Second)
			cmd := exec.CommandContext(propertyCtx, adb, "-s", serial, "shell", "getprop", "sys.boot_completed")
			cmd.Env = sdkEnvironment(layout)
			output, propertyErr := cmd.CombinedOutput()
			propertyCancel()
			if propertyErr == nil && strings.TrimSpace(string(output)) == "1" {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return fmt.Errorf("emulator %s did not report sys.boot_completed=1 within %s", serial, timeout)
}

func runSDKTool(ctx context.Context, layout sdkLayout, path string, args ...string) error {
	return runSDKToolInput(ctx, layout, "", path, args...)
}

func runSDKToolInput(ctx context.Context, layout sdkLayout, input, path string, args ...string) error {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = sdkEnvironment(layout)
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %s: %w", path, strings.Join(args, " "), strings.TrimSpace(string(output)), err)
	}
	return nil
}

func sdkEnvironment(layout sdkLayout) []string {
	env := append([]string{}, os.Environ()...)
	env = append(env, "ANDROID_HOME="+layout.SDKRoot, "ANDROID_SDK_ROOT="+layout.SDKRoot)
	if _, err := os.Stat(filepath.Join(layout.JDKRoot, "bin", executable("java"))); err == nil {
		env = append(env, "JAVA_HOME="+layout.JDKRoot)
		path := os.Getenv("PATH")
		separator := string(os.PathListSeparator)
		env = append(env, "PATH="+filepath.Join(layout.JDKRoot, "bin")+separator+layout.BinDir+separator+path)
	}
	return env
}

func executable(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func toolRelativePath(name string) string {
	switch name {
	case "adb":
		return filepath.Join("platform-tools", executable(name))
	case "sdkmanager", "avdmanager":
		return filepath.Join("cmdline-tools", "latest", "bin", executable(name))
	case "java", "javac":
		return filepath.Join("toolchains", "jdk-17", "bin", executable(name))
	case "gradle":
		return filepath.Join("toolchains", "gradle-8.10.2", "bin", executable(name))
	default:
		return filepath.Join("emulator", executable(name))
	}
}
