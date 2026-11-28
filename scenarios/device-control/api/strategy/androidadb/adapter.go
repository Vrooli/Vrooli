package androidadb

import (
	"context"
	"strings"
	"sync"

	"device-control/internal/androidartifact"
	"device-control/strategy"
)

type Adapter struct {
	runner           Runner
	serial           string
	identitySerial   string
	endpoint         string
	transport        string
	recordingMu      sync.Mutex
	recordings       map[string]activeRecording
	rendererID       func(context.Context, string) (string, error)
	rendererTarget   func(context.Context, string) (string, string, error)
	signingInspector androidartifact.APKSigningInspector
	probes           *probeCache
}

func newAdapter(runner Runner, serial string) *Adapter {
	return &Adapter{runner: runner, serial: serial, identitySerial: serial, transport: transportForSerial(serial), recordings: map[string]activeRecording{}, rendererTarget: discoverWebViewRenderer, signingInspector: androidartifact.AndroidSDKSigningInspector{ADBPath: "adb"}, probes: newProbeCache()}
}

type Config struct {
	Serial               string
	RequireDefaultServer bool
}

// New constructs an unbound ADB strategy. Production configuration is passed
// by registry.DefaultWithEnvironment.
func New() *Adapter { return NewWithConfig(Config{}) }

func NewWithConfig(config Config) *Adapter {
	return newAdapter(commandRunner{requireDefaultADBServer: config.RequireDefaultServer}, strings.TrimSpace(config.Serial))
}

func NewWithRunner(r Runner, serial string) *Adapter {
	return newAdapter(r, serial)
}

func (a *Adapter) ForDevice(serial string) strategy.Strategy {
	serial = strings.TrimSpace(serial)
	return &Adapter{runner: a.runner, serial: serial, identitySerial: serial, transport: transportForSerial(serial), recordings: map[string]activeRecording{}, rendererID: a.rendererID, rendererTarget: a.rendererTarget, signingInspector: a.signingInspector, probes: a.probes}
}

// NewNetworkTransport returns a fresh, unbound adapter that shares this
// adapter's runner. The control service calls ConnectNetwork on the returned
// adapter to onboard a directly-addressable endpoint without mutating the shared
// registry instance.
func (a *Adapter) NewNetworkTransport() strategy.Strategy {
	return &Adapter{runner: a.runner, recordings: map[string]activeRecording{}, rendererID: a.rendererID, rendererTarget: a.rendererTarget, probes: a.probes}
}

// RestoreWireless rebuilds an endpoint-bound adapter after the control
// service is restarted. The endpoint was previously verified against the
// hardware serial during promotion and is restored only from durable local
// state, never from operator-supplied input.
func (a *Adapter) RestoreWireless(endpoint string) strategy.Strategy {
	identity := a.identitySerial
	if identity == "" {
		identity = a.serial
	}
	return &Adapter{runner: a.runner, serial: identity, identitySerial: identity, endpoint: strings.TrimSpace(endpoint), transport: "wireless", recordings: map[string]activeRecording{}, rendererID: a.rendererID, rendererTarget: a.rendererTarget, probes: a.probes}
}

func (a *Adapter) ID() string { return "android-adb" }

func (a *Adapter) args(args ...string) []string {
	selector := a.serial
	if a.endpoint != "" {
		selector = a.endpoint
	}
	if selector != "" {
		return append([]string{"-s", selector}, args...)
	}
	return args
}

// shellQuote preserves spaces and shell metacharacters when adb serializes
// arguments after the `shell` boundary. Intent string extras are user content
// and must remain one remote argument rather than becoming an accidental
// package or component token.
func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func (a *Adapter) connected(ctx context.Context) (bool, string) {
	if a.endpoint != "" || a.serial != "" {
		out, err := a.runner.Run(ctx, "adb", "devices", "-l")
		if err == nil {
			selector := a.serial
			if a.endpoint != "" {
				selector = a.endpoint
			}
			for _, line := range strings.Split(string(out), "\n") {
				fields := strings.Fields(strings.TrimSpace(line))
				if len(fields) >= 2 && fields[0] == selector && fields[1] == "device" {
					return true, ""
				}
			}
		}
		if a.endpoint != "" {
			return false, "Enable Android wireless debugging, authorize this host, and verify the saved wireless endpoint is reachable."
		}
		return false, "Enable USB debugging, authorize this computer, and verify the Android device is reachable."
	}
	out, err := a.runner.Run(ctx, "adb", a.args("devices")...)
	if err != nil {
		return false, "Install the android-sdk resource (adb/platform-tools) and put adb on PATH."
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == "device" {
			return true, ""
		}
	}
	return false, "Enable USB debugging, authorize this computer, and attach an Android device or boot the emulator."
}
