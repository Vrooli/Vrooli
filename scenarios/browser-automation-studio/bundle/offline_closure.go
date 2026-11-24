package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	closure "github.com/vrooli/vrooli/packages/browser-closure"
)

func main() {
	if len(os.Args) < 2 {
		fatal("expected package, install, rollback, or remove command")
	}
	switch os.Args[1] {
	case "package":
		fs := flag.NewFlagSet("package", flag.ExitOnError)
		source := fs.String("closure", "", "validated staged closure directory")
		archive := fs.String("archive", "", "offline .tar.gz output")
		manifest := fs.String("manifest", "", "offline manifest output")
		platform := fs.String("platform", "", "expected platform lock")
		version := fs.String("core-version", "", "expected rebrowser-playwright-core version")
		revision := fs.String("chromium-revision", "", "expected browsers.json Chromium revision")
		shellRevision := fs.String("headless-shell-revision", "", "expected browsers.json headless-shell revision")
		ffmpegRevision := fs.String("ffmpeg-revision", "", "expected browsers.json ffmpeg revision")
		hostPlatform := fs.String("playwright-host-platform", "", "expected Playwright host platform")
		_ = fs.Parse(os.Args[2:])
		if *source == "" || *archive == "" || *manifest == "" {
			fatal("package requires --closure, --archive and --manifest")
		}
		result, err := closure.Create(*source, *archive, *manifest, closure.Expected{Platform: *platform, HostPlatform: *hostPlatform, Package: "rebrowser-playwright-core", Version: *version, ChromiumRev: *revision, HeadlessRev: *shellRevision, FFmpegRev: *ffmpegRevision})
		if err != nil {
			fatal(err.Error())
		}
		writeJSON(result)
	case "install":
		fs := flag.NewFlagSet("install", flag.ExitOnError)
		archive := fs.String("archive", "", "offline .tar.gz input")
		manifest := fs.String("manifest", "", "offline manifest input")
		destination := fs.String("destination", "", "BAS resources/playwright install path")
		platform := fs.String("platform", "", "expected platform lock")
		version := fs.String("core-version", "", "expected rebrowser-playwright-core version")
		revision := fs.String("chromium-revision", "", "expected browsers.json Chromium revision")
		shellRevision := fs.String("headless-shell-revision", "", "expected browsers.json headless-shell revision")
		ffmpegRevision := fs.String("ffmpeg-revision", "", "expected browsers.json ffmpeg revision")
		hostPlatform := fs.String("playwright-host-platform", "", "expected Playwright host platform")
		_ = fs.Parse(os.Args[2:])
		if *archive == "" || *manifest == "" || *destination == "" {
			fatal("install requires --archive, --manifest and --destination")
		}
		err := closure.Install(*archive, *manifest, *destination, closure.Expected{Platform: *platform, HostPlatform: *hostPlatform, Package: "rebrowser-playwright-core", Version: *version, ChromiumRev: *revision, HeadlessRev: *shellRevision, FFmpegRev: *ffmpegRevision})
		if err != nil {
			fatal(err.Error())
		}
		writeJSON(map[string]string{"state": "installed", "destination": *destination})
	case "rollback", "remove":
		command := os.Args[1]
		fs := flag.NewFlagSet(command, flag.ExitOnError)
		destination := fs.String("destination", "", "BAS resources/playwright install path")
		platform := fs.String("platform", "", "expected platform lock")
		version := fs.String("core-version", "", "expected rebrowser-playwright-core version")
		revision := fs.String("chromium-revision", "", "expected browsers.json Chromium revision")
		shellRevision := fs.String("headless-shell-revision", "", "expected browsers.json headless-shell revision")
		ffmpegRevision := fs.String("ffmpeg-revision", "", "expected browsers.json ffmpeg revision")
		hostPlatform := fs.String("playwright-host-platform", "", "expected Playwright host platform")
		_ = fs.Parse(os.Args[2:])
		if *destination == "" {
			fatal(command + " requires --destination")
		}
		expected := closure.Expected{Platform: *platform, HostPlatform: *hostPlatform, Package: "rebrowser-playwright-core", Version: *version, ChromiumRev: *revision, HeadlessRev: *shellRevision, FFmpegRev: *ffmpegRevision}
		if command == "rollback" {
			if err := closure.Rollback(*destination, expected); err != nil {
				fatal(err.Error())
			}
			writeJSON(map[string]string{"state": "rolled-back", "destination": *destination})
		} else {
			if err := closure.Remove(*destination, expected); err != nil {
				fatal(err.Error())
			}
			writeJSON(map[string]string{"state": "removed", "destination": *destination})
		}
	default:
		fatal("unknown command %q", os.Args[1])
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "offline-closure: "+format+"\n", args...)
	os.Exit(1)
}
func writeJSON(value any) {
	if err := json.NewEncoder(os.Stdout).Encode(value); err != nil {
		fatal(err.Error())
	}
}
