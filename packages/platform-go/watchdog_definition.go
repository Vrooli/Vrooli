package platform

import (
	"fmt"
	"path/filepath"
	"strings"
)

// WatchdogDefinitionOptions contains the platform-neutral inputs for a boot
// supervisor definition. Rendering belongs here with the native lifecycle
// backends; scenario packages should not carry service-manager templates.
type WatchdogDefinitionOptions struct {
	Root          string
	Home          string
	LoopBinary    string
	VrooliBinary  string
	Username      string
	SystemService bool
}

// RenderWatchdogDefinition renders the native boot-supervisor definition for
// a target platform. The target argument is explicit so injected platform
// tests can exercise every renderer on one host.
func RenderWatchdogDefinition(target string, options WatchdogDefinitionOptions) (string, error) {
	switch target {
	case "linux":
		return renderSystemdDefinition(options), nil
	case "macos":
		return renderLaunchdDefinition(options), nil
	case "windows":
		return renderWindowsTaskDefinition(options), nil
	default:
		return "", fmt.Errorf("platform: unsupported watchdog target %q", target)
	}
}

func renderSystemdDefinition(options WatchdogDefinitionOptions) string {
	wantedBy := "default.target"
	userDirective := ""
	if options.SystemService {
		userDirective = "User=root\n"
		wantedBy = "multi-user.target"
	}
	return fmt.Sprintf(`[Unit]
Description=Vrooli Autoheal - Self-healing infrastructure supervisor
After=network-online.target docker.service docker.socket
Wants=network-online.target
Wants=docker.service
StartLimitIntervalSec=300
StartLimitBurst=5

[Service]
Type=simple
ExecStart=%s
Restart=always
RestartSec=15
%sEnvironment=VROOLI_LIFECYCLE_MANAGED=true
Environment=HOME=%s
Environment=VROOLI_ROOT=%s
Environment=VROOLI_BIN=%s
Environment=PATH=/usr/local/bin:/usr/bin:/bin:%s/.local/bin:%s/.vrooli/bin
WorkingDirectory=%s
TimeoutStopSec=30

[Install]
WantedBy=%s
`, options.LoopBinary, userDirective, options.Home, options.Root, options.VrooliBinary, options.Home, options.Home, filepath.Join(options.Root, "scenarios", "vrooli-autoheal"), wantedBy)
}

func renderLaunchdDefinition(options WatchdogDefinitionOptions) string {
	loopBinary := xmlEscape(options.LoopBinary)
	root := xmlEscape(options.Root)
	home := xmlEscape(options.Home)
	logPath := xmlEscape(filepath.Join(options.Home, "Library", "Logs", "vrooli-autoheal.log"))
	errPath := xmlEscape(filepath.Join(options.Home, "Library", "Logs", "vrooli-autoheal.error.log"))
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>Label</key><string>com.vrooli.autoheal</string>
  <key>ProgramArguments</key><array><string>%s</string></array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>EnvironmentVariables</key><dict>
    <key>VROOLI_LIFECYCLE_MANAGED</key><string>true</string>
    <key>VROOLI_ROOT</key><string>%s</string>
    <key>HOME</key><string>%s</string>
    <key>PATH</key><string>/usr/local/bin:/usr/bin:/bin:%s/.local/bin:%s/.vrooli/bin</string>
  </dict>
  <key>WorkingDirectory</key><string>%s</string>
  <key>StandardOutPath</key><string>%s</string>
  <key>StandardErrorPath</key><string>%s</string>
  <key>ThrottleInterval</key><integer>15</integer>
</dict></plist>
`, loopBinary, root, home, home, home, xmlEscape(filepath.Join(options.Root, "scenarios", "vrooli-autoheal")), logPath, errPath)
}

func renderWindowsTaskDefinition(options WatchdogDefinitionOptions) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Task version="1.4" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo><Description>Vrooli Autoheal - Self-healing infrastructure supervisor</Description><Author>Vrooli</Author></RegistrationInfo>
  <Triggers><BootTrigger><Enabled>true</Enabled><Delay>PT30S</Delay></BootTrigger></Triggers>
  <Principals><Principal id="Author"><UserId>%s</UserId><LogonType>S4U</LogonType><RunLevel>HighestAvailable</RunLevel></Principal></Principals>
  <Settings><MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy><StartWhenAvailable>true</StartWhenAvailable><AllowStartOnDemand>true</AllowStartOnDemand><Enabled>true</Enabled><ExecutionTimeLimit>PT0S</ExecutionTimeLimit><RestartOnFailure><Interval>PT1M</Interval><Count>999</Count></RestartOnFailure></Settings>
  <Actions Context="Author"><Exec><Command>%s</Command><WorkingDirectory>%s</WorkingDirectory></Exec></Actions>
</Task>
`, xmlEscape(options.Username), xmlEscape(options.LoopBinary), xmlEscape(filepath.Join(options.Root, "scenarios", "vrooli-autoheal")))
}

func xmlEscape(value string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&apos;").Replace(value)
}
