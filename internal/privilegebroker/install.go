package privilegebroker

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/config"
)

const (
	DefaultSocketPath = "/run/vrooli/privilege-broker.sock"
	defaultAuditPath  = "/var/log/vrooli/privilege-broker.jsonl"
	serviceName       = "vrooli-privilege-broker.service"
	servicePath       = "/etc/systemd/system/" + serviceName
	installedBinary   = "/usr/local/lib/vrooli/vrooli-privilege-broker"
)

// SocketPath resolves the broker endpoint from the platform runtime directory.
// User-session platforms must not put a privileged control socket in a
// repository or home directory; Linux keeps /run as the system fallback.
func SocketPath() string {
	return platformgo.PrivilegeBrokerSocketPath()
}

type SetupStatus struct {
	Available  bool   `json:"available"`
	Supported  bool   `json:"supported"`
	SocketPath string `json:"socket_path"`
	Reason     string `json:"reason,omitempty"`
	Recovery   string `json:"recovery,omitempty"`
}

type Installer struct {
	Executable string
	RepoRoot   string
	Run        func(context.Context, string, ...string) ([]byte, error)
	Copy       func(string, string) error
}

func DefaultInstaller(executable string) Installer {
	return Installer{Executable: executable, Run: func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return shell.NewCommandContext(ctx, name, args...).CombinedOutput()
	}, Copy: copyRootOwned}
}

func DefaultInstallerForRepo(executable, repoRoot string) Installer {
	installer := DefaultInstaller(executable)
	installer.RepoRoot = repoRoot
	return installer
}

func (i Installer) Install(ctx context.Context) (SetupStatus, error) {
	if runtime.GOOS != string(hostreqspec.PlatformLinux) {
		return SetupStatus{Supported: false, SocketPath: SocketPath(), Reason: "privilege broker is currently supported only on Linux with systemd", Recovery: "Use a supported Linux host, then re-run `vrooli setup --sudo-mode=ask`."}, nil
	}
	if os.Geteuid() != 0 {
		return SetupStatus{Supported: true, SocketPath: SocketPath(), Reason: "setup was not elevated", Recovery: "Re-run `vrooli setup --sudo-mode=ask` to install the privilege broker."}, nil
	}
	uid, gid, err := invokingIdentity()
	if err != nil {
		return SetupStatus{Supported: true, SocketPath: SocketPath(), Reason: err.Error(), Recovery: "Re-run `vrooli setup --sudo-mode=ask` from the owner account that will run Vrooli."}, nil
	}
	runtimeRoot, err := installedRuntimeHomeRoot(i.RepoRoot, uint32(uid))
	if err != nil {
		return SetupStatus{}, fmt.Errorf("resolve broker runtime-home root: %w", err)
	}
	if strings.TrimSpace(i.Executable) == "" {
		return SetupStatus{}, fmt.Errorf("resolve broker executable: empty path")
	}
	if i.Copy == nil {
		i.Copy = copyRootOwned
	}
	if err := i.Copy(i.Executable, installedBinary); err != nil {
		return SetupStatus{}, fmt.Errorf("install broker executable: %w", err)
	}
	if err := os.WriteFile(servicePath, []byte(systemdUnitWithRuntimeHome(installedBinary, uid, gid, runtimeRoot)), tuning.PermFile); err != nil {
		return SetupStatus{}, fmt.Errorf("write broker systemd unit: %w", err)
	}
	if err := os.Chown(servicePath, 0, 0); err != nil {
		return SetupStatus{}, fmt.Errorf("own broker unit: %w", err)
	}
	if i.Run == nil {
		i.Run = DefaultInstaller(i.Executable).Run
	}
	for _, args := range brokerServiceCommands() {
		if out, err := i.Run(ctx, "systemctl", args...); err != nil {
			return SetupStatus{}, fmt.Errorf("systemctl %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
		}
	}
	return SetupStatus{Available: true, Supported: true, SocketPath: SocketPath()}, nil
}

// brokerServiceCommands deliberately restarts the broker after an install or
// upgrade. `systemctl enable --now` only starts an inactive service; it leaves
// an already-running process on its old binary and old unit sandbox settings.
func brokerServiceCommands() [][]string {
	return [][]string{{"daemon-reload"}, {"enable", serviceName}, {"restart", serviceName}, {"is-active", "--quiet", serviceName}}
}

func Inspect() SetupStatus {
	if runtime.GOOS != string(hostreqspec.PlatformLinux) {
		return SetupStatus{Supported: false, SocketPath: SocketPath(), Reason: "privilege broker is currently supported only on Linux with systemd"}
	}
	socket := SocketPath()
	conn, err := net.Dial("unix", socket)
	if err != nil {
		return SetupStatus{Supported: true, SocketPath: socket, Reason: "broker socket is unavailable", Recovery: "Re-run `vrooli setup --sudo-mode=ask` to install or repair the privilege broker."}
	}
	_ = conn.Close()
	return SetupStatus{Available: true, Supported: true, SocketPath: socket}
}

func systemdUnit(executable string, uid, gid int) string {
	return systemdUnitWithRuntimeHome(executable, uid, gid, "")
}

func systemdUnitWithRuntimeHome(executable string, uid, gid int, runtimeRoot string) string {
	runtimeArg := ""
	if strings.TrimSpace(runtimeRoot) != "" {
		runtimeArg = " --runtime-home-root " + strconv.Quote(filepath.Clean(runtimeRoot))
	}
	return fmt.Sprintf(`[Unit]
Description=Vrooli setup-managed privilege broker
After=network.target

[Service]
Type=simple
User=root
ExecStart=%s __privilege-broker serve --socket %s --allowed-uid %d --socket-gid %d --audit-path %s%s
Restart=on-failure
RestartSec=2s
RuntimeDirectory=vrooli
RuntimeDirectoryMode=755
LogsDirectory=vrooli
LogsDirectoryMode=750
NoNewPrivileges=true
PrivateTmp=true
ProtectHome=false
ProtectSystem=strict
# UFW creates this lock even for read-only status requests, and writes its
# managed rules below /etc/ufw. The lock does not exist until UFW is used, so
# its path must be optional or systemd refuses to start the broker.
ReadWritePaths=/run/vrooli -/run/ufw.lock /etc/ufw /var/log/vrooli

[Install]
WantedBy=multi-user.target
`, strconv.Quote(executable), SocketPath(), uid, gid, defaultAuditPath, runtimeArg)
}

func invokingIdentity() (int, int, error) {
	uidText := strings.TrimSpace(os.Getenv("SUDO_UID"))
	if uidText == "" {
		return 0, 0, fmt.Errorf("elevated setup needs SUDO_UID to identify the authorized broker caller")
	}
	uid, err := strconv.Atoi(uidText)
	if err != nil || uid < 1 {
		return 0, 0, fmt.Errorf("invalid SUDO_UID")
	}
	u, err := user.LookupId(uidText)
	if err != nil {
		return 0, 0, fmt.Errorf("lookup invoking user: %w", err)
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return 0, 0, fmt.Errorf("parse invoking user gid: %w", err)
	}
	return uid, gid, nil
}

func copyRootOwned(source, destination string) error {
	return copyAtomically(source, destination, os.Chown)
}

// copyAtomically replaces destination with a fully-written sibling inode. A
// running systemd service may still execute the old inode, while future starts
// resolve destination to the new binary; opening destination with O_TRUNC would
// instead fail with ETXTBSY and leave setup unable to repair the broker.
func copyAtomically(source, destination string, chown func(string, int, int) error) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := config.WriteOwnedFileAtomic(destination, data, tuning.PermDir); err != nil {
		return err
	}
	return chown(destination, 0, 0)
}
