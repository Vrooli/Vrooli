package scanners

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func findExternalTool(name string) string {
	if _, err := exec.LookPath(name); err == nil {
		return name
	}
	homeBinPath := filepath.Join(os.Getenv("HOME"), "go", "bin", name)
	if _, err := os.Stat(homeBinPath); err == nil {
		return homeBinPath
	}
	return name
}

func externalToolAvailable(binPath string, versionArgs ...string) bool {
	return exec.Command(binPath, versionArgs...).Run() == nil
}

func externalToolVersion(binPath string, versionArgs ...string) ([]byte, error) {
	return exec.Command(binPath, versionArgs...).Output()
}

// runExternalTool owns the shared process, timeout, and output-capture path for
// command-backed scanners. A scanner supplies its tool-specific exit policy.
func runExternalTool(binPath string, args []string, timeout time.Duration, allowExit func(error) bool, captureStdout bool) (stdout, stderr string, err error) {
	cmd := exec.Command(binPath, args...)
	var out, diagnostic bytes.Buffer
	if captureStdout {
		cmd.Stdout = &out
	}
	cmd.Stderr = &diagnostic

	if timeout > 0 {
		timer := time.AfterFunc(timeout, func() {
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		})
		defer timer.Stop()
	}

	err = cmd.Run()
	if err != nil && (allowExit == nil || !allowExit(err)) {
		return out.String(), diagnostic.String(), fmt.Errorf("%s - stderr: %s", err, diagnostic.String())
	}
	return out.String(), diagnostic.String(), nil
}

func countExternalToolFiles(path string, skipDirectory func(string) bool, includeFile func(string) bool) (int, int) {
	fileCount, lineCount := 0, 0
	_ = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.IsDir() {
			if skipDirectory(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if includeFile(filePath) {
			fileCount++
			if content, err := os.ReadFile(filePath); err == nil {
				lineCount += bytes.Count(content, []byte("\n"))
			}
		}
		return nil
	})
	return fileCount, lineCount
}

func externalScanResult(scanID string, scannerType ScannerType, start time.Time, path string, findings []Finding, filesScanned, linesScanned int) *ScanResult {
	return &ScanResult{
		ScanID:       scanID,
		ScannerType:  scannerType,
		StartTime:    start,
		EndTime:      time.Now(),
		Duration:     time.Since(start),
		ScannedPath:  path,
		Findings:     findings,
		FilesScanned: filesScanned,
		LinesScanned: linesScanned,
	}
}
