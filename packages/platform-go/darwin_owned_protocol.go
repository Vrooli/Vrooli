package platform

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Snapshot is derived from typed ServiceManagement dictionaries. No diagnostic
// launchctl print field is interpreted as manager authority.
type darwinJobSnapshot struct {
	PID   int
	State string
}

func darwinOwnedLabel(i OperationIntent) string {
	return "com.vrooli.owned." + strings.TrimSuffix(operationUnitName(i), ".service")
}

func darwinOwnedPlist(i OperationIntent, s OperationSpec) string {
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?><plist version=\"1.0\"><dict>\n")
	for _, v := range [][2]string{{"Label", darwinOwnedLabel(i)}, {"WorkingDirectory", s.Dir}, {"ProcessType", "Background"}} {
		fmt.Fprintf(&b, "<key>%s</key><string>%s</string>\n", v[0], xmlText(v[1]))
	}
	b.WriteString("<key>ProgramArguments</key><array>")
	for _, v := range append([]string{s.Path}, s.Args...) {
		fmt.Fprintf(&b, "<string>%s</string>", xmlText(v))
	}
	b.WriteString("</array>\n<key>EnvironmentVariables</key><dict>")
	for _, v := range s.Env {
		k, value, _ := strings.Cut(v, "=")
		fmt.Fprintf(&b, "<key>%s</key><string>%s</string>", xmlText(k), xmlText(value))
	}
	b.WriteString("</dict>\n<key>RunAtLoad</key><true/><key>KeepAlive</key><false/><key>AbandonProcessGroup</key><false/><key>ExitTimeOut</key><integer>10</integer>\n")
	// Native memory limits are advisory per process. Never apply NPROC to
	// the actual operator: it is an account-wide ceiling, not role containment.
	// The final kernel no-children policy owns the ProcessOnly contract.
	fmt.Fprintf(&b, "<key>Nice</key><integer>%d</integer>", niceForWeight(s.Limits.CPUWeight))
	for _, kind := range []string{"SoftResourceLimits", "HardResourceLimits"} {
		fmt.Fprintf(&b, "<key>%s</key><dict><key>ResidentSetSize</key><integer>%d</integer></dict>\n", kind, s.Limits.MemoryMaxBytes)
	}
	b.WriteString("</dict></plist>\n")
	return b.String()
}

// A finite, owned contract with the maintained CLI. Adding another entrypoint
// requires deliberate implementation and qualification, never a registration.
func validateDarwinMaintainedLaunchShape(i OperationIntent, s OperationSpec) error {
	if i.Owner != "resource-initial" && i.Owner != "resource-update" {
		return ErrUnsupported
	}
	if len(s.Args) != 2 || s.Args[0] != "__vrooli-openrgb-native-capsule" || len(s.Env) != 0 || s.Dir != filepath.Dir(s.Path) {
		return ErrUnsupported
	}
	capsule := s.Args[1]
	if len(capsule) > 4096 || !filepath.IsAbs(capsule) || filepath.Clean(capsule) != capsule || strings.ContainsAny(capsule, "\x00\r\n") || filepath.Base(capsule) != "capsule.json" || filepath.Base(filepath.Dir(capsule)) != i.AttemptID || filepath.Base(filepath.Dir(filepath.Dir(capsule))) != "launch" || !filepath.IsAbs(s.Path) || filepath.Clean(s.Path) != s.Path {
		return ErrOperationIdentity
	}
	// The maintained early reader binds this canonical path to the frozen
	// material namespace/account/intent. Custom resource roots are supported;
	// no runtime entrypoint registration or generic executable is admitted.

	return nil
}
