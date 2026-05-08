package main

import (
	"errors"
	"fmt"
	"os"

	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/driverid"
	"workspace-sandbox/internal/driverpref"
)

func main() {
	if err := run(os.Args[1:], defaultDeps()); err != nil {
		fmt.Fprintf(os.Stderr, "workspace-sandbox-launcher: %v\n", err)
		os.Exit(1)
	}
}

type deps struct {
	goos            string
	geteuid         func() int
	getenv          func(string) string
	probePlain      func() error
	probeAppArmor   func() error
	execProcess     func(string, []string, []string) error
	runAndWait      func(string, []string, []string) error
	defaultBaseDir  func() string
	writeDiagnostic func(string, ...any)
}

func defaultDeps() deps {
	return deps{
		goos:            runtimeGOOS(),
		geteuid:         currentEUID,
		getenv:          os.Getenv,
		probePlain:      probePlainUserns,
		probeAppArmor:   probeAppArmorUserns,
		execProcess:     execProcess,
		runAndWait:      runAndWait,
		defaultBaseDir:  config.DefaultBaseDir,
		writeDiagnostic: func(format string, args ...any) { fmt.Fprintf(os.Stderr, format+"\n", args...) },
	}
}

func run(args []string, d deps) error {
	if len(args) > 1 {
		return fmt.Errorf("usage: workspace-sandbox-launcher [api-binary]")
	}
	apiPath := "./workspace-sandbox-api"
	if len(args) == 1 && args[0] != "" {
		apiPath = args[0]
	}

	baseDir := resolveBaseDir(d)
	pref, prefState, err := loadPreference(baseDir)
	if err != nil {
		return err
	}
	mode, err := chooseLaunchMode(pref, d)
	if err != nil {
		return err
	}

	argv0, argv := commandForMode(mode, apiPath)
	d.writeDiagnostic("os=%s driverPreference=%s launchMode=%s", d.goos, prefState, mode)
	if d.goos == "windows" {
		return d.runAndWait(argv0, argv, os.Environ())
	}
	return d.execProcess(argv0, argv, os.Environ())
}

func resolveBaseDir(d deps) string {
	if baseDir := d.getenv("SANDBOX_BASE_DIR"); baseDir != "" {
		return baseDir
	}
	return d.defaultBaseDir()
}

func loadPreference(baseDir string) (driverid.ID, string, error) {
	pref, err := driverpref.Load(baseDir)
	if err == nil {
		return pref, string(pref), nil
	}
	if errors.Is(err, driverpref.ErrNotFound) {
		return "", "default", nil
	}
	return "", "", err
}

type launchMode string

const (
	modeDirect   launchMode = "direct"
	modeUnshare  launchMode = "unshare"
	modeAppArmor launchMode = "apparmor-unshare"
)

func chooseLaunchMode(pref driverid.ID, d deps) (launchMode, error) {
	if d.goos != "linux" {
		return modeDirect, nil
	}

	switch pref {
	case "", driverid.OverlayfsUserNS:
		if err := d.probeAppArmor(); err == nil {
			return modeAppArmor, nil
		}
		if err := d.probePlain(); err == nil {
			return modeUnshare, nil
		}
		return "", fmt.Errorf("overlayfs-userns requires a working `unshare -U -m -r` launch path; run `vrooli setup status --environment development --resources none --scenarios workspace-sandbox`, then privileged setup when approved so the workspace_sandbox_userns safeguard can install the AppArmor profile")
	case driverid.FuseOverlayfs, driverid.Copy:
		return modeDirect, nil
	case driverid.OverlayfsRoot:
		if d.geteuid() != 0 {
			return "", fmt.Errorf("overlayfs-root preference requires running the API as root or with CAP_SYS_ADMIN; choose another driver with /api/v1/driver/select or run from a suitably privileged service")
		}
		return modeDirect, nil
	default:
		return "", fmt.Errorf("unknown driver ID: %s", pref)
	}
}

func commandForMode(mode launchMode, apiPath string) (string, []string) {
	switch mode {
	case modeAppArmor:
		return "aa-exec", []string{"aa-exec", "-p", "vrooli-workspace-sandbox", "--", "unshare", "-U", "-m", "-r", apiPath}
	case modeUnshare:
		return "unshare", []string{"unshare", "-U", "-m", "-r", apiPath}
	default:
		return apiPath, []string{apiPath}
	}
}
