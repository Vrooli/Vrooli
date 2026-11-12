package platform

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Initial privileged setup publishes an immutable boot image and descriptor.
// The existing supervisor itself runs as the actual operator; ordinary HOME,
// private state and source lifecycle never run under a permanent root broker.
type (
	nativeSystemBrokerRoots struct{ Install, State, Unit, SystemRoot string }
	nativeSystemBrokerPlan  struct {
		Target, Source, SHA256, ImagePath, StateRoot, UnitPath string
		DescriptorPath                                         string
		Descriptor                                             []byte
		Definition                                             ServiceDefinition
		Options                                                NativeSystemServiceBrokerOptions
	}
)

func buildNativeSystemBrokerPlan(target string, o ServiceInstallOptions, roots nativeSystemBrokerRoots) (nativeSystemBrokerPlan, error) {
	var p nativeSystemBrokerPlan
	b := o.SystemBroker
	if o.User || b == nil {
		return p, errors.New("platform: explicit native system broker setup is required")
	}
	if target != "darwin" && target != "windows" {
		return p, ErrUnsupported
	}
	if !filepath.IsAbs(o.Executable) || filepath.Clean(o.Executable) != o.Executable || len(o.Executable) > 4096 || !validNativeBrokerDigest(b.ExecutableSHA256) {
		return p, errors.New("platform: system broker requires an exact checksum-bound absolute executable")
	}
	for _, value := range []string{o.Executable, b.OperatorPrincipal, b.OperatorUsername, b.OperatorHome} {
		if value == "" || len(value) > 4096 || strings.ContainsAny(value, "\x00\r\n") {
			return p, errors.New("platform: invalid native system broker operator identity")
		}
	}
	if !filepath.IsAbs(b.OperatorHome) || filepath.Clean(b.OperatorHome) != b.OperatorHome || o.SourceRoot != "" || o.HomeDir != "" && o.HomeDir != b.OperatorHome || o.LogPath != "" {
		return p, errors.New("platform: native boot supervisor refuses unfrozen source/HOME/log overrides")
	}
	if err := validateNativeGlobalResourceStorageRoot(b.GlobalResourceStorageRoot); err != nil {
		return p, err
	}
	for _, root := range []string{roots.Install, roots.State} {
		if !filepath.IsAbs(root) || filepath.Clean(root) != root {
			return p, errors.New("platform: native system broker roots must be canonical")
		}
	}
	if target == "darwin" && (b.OperatorUID == 0 || b.OperatorPrincipal != "unix:"+strconv.FormatUint(uint64(b.OperatorUID), 10)) {
		return p, errors.New("platform: Darwin broker requires an exact nonroot operator UID")
	}
	if target == "windows" && !canonicalNativeBrokerWindowsUserPrincipal(b.OperatorPrincipal) {
		return p, errors.New("platform: Windows broker requires exact operator SID")
	}
	executableName := "vrooli"
	if target == "windows" {
		executableName += ".exe"
	}
	image := filepath.Join(roots.Install, "versions", b.ExecutableSHA256, executableName)
	descriptorPath := nativeSystemBrokerDescriptorPath(image, *b)
	args := []string{NativeSystemSupervisorVerb, descriptorPath}
	path := "/usr/bin:/bin:/usr/sbin:/sbin"
	if target == "windows" {
		if !filepath.IsAbs(roots.SystemRoot) || filepath.Clean(roots.SystemRoot) != roots.SystemRoot {
			return p, errors.New("platform: native Windows system root required")
		}
		path = filepath.Join(roots.SystemRoot, "System32") + ";" + roots.SystemRoot
	}
	unit, _ := CoreUnitByID(CoreUnitRuntimeSupervisor)
	d := ServiceDefinition{DocumentationURL: DocumentationURL(unit.OwnerPath), Name: "vrooli-runtime-supervisor", Label: "com.vrooli.runtime-supervisor", Description: "Vrooli native runtime supervisor", Executable: image, Args: args, Env: map[string]string{"HOME": b.OperatorHome, "PATH": path, "VROOLI_RUNTIME_SUPERVISOR": "on"}, WorkingDirectory: b.OperatorHome, Kind: KindDaemon, Scope: ScopeSystem, Username: b.OperatorUsername, Restart: RestartPolicy{Mode: RestartAlways, Delay: 5 * time.Second}, StopTimeout: 30 * time.Second}
	if b.GlobalResourceStorageRoot != "" {
		d.Env["VROOLI_RESOURCE_STORAGE_ROOT"] = b.GlobalResourceStorageRoot
	}
	if e := d.Validate(); e != nil {
		return p, fmt.Errorf("platform: native broker definition: %w", e)
	}
	if target == "darwin" && (!filepath.IsAbs(roots.Unit) || filepath.Clean(roots.Unit) != roots.Unit) {
		return p, errors.New("platform: canonical LaunchDaemon path required")
	}
	if target == "windows" && roots.Unit != "VrooliRuntimeSupervisor" {
		return p, errors.New("platform: fixed SCM broker service required")
	}
	descriptor, e := json.Marshal(nativeSystemBrokerDescriptor{Version: nativeSystemBrokerSchemaVersion, Options: *b, ExecutablePath: image})
	if e != nil {
		return p, e
	}
	return nativeSystemBrokerPlan{Target: target, Source: o.Executable, SHA256: b.ExecutableSHA256, ImagePath: image, StateRoot: b.OperatorHome, UnitPath: roots.Unit, Definition: d, Options: *b, DescriptorPath: descriptorPath, Descriptor: append(descriptor, '\n')}, nil
}

func validNativeBrokerDigest(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			if c < 'a' || c > 'f' {
				return false
			}
		}
	}
	return true
}

const (
	NativeSystemSupervisorVerb      = "__vrooli-runtime-system-broker"
	nativeSystemBrokerSchemaVersion = 2
	maxNativeSystemBrokerImageBytes = int64(512 << 20)
)

type nativeSystemBrokerDescriptor struct {
	Version        int                              `json:"version"`
	Options        NativeSystemServiceBrokerOptions `json:"options"`
	ExecutablePath string                           `json:"executable_path"`
}

func decodeNativeSystemServiceBrokerDescriptor(data []byte) (nativeSystemBrokerDescriptor, error) {
	var d nativeSystemBrokerDescriptor
	if len(data) == 0 || len(data) > 64<<10 {
		return d, errors.New("platform: broker descriptor exceeds bound")
	}
	if e := validateNativeBrokerJSON(data); e != nil {
		return d, e
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if e := decoder.Decode(&d); e != nil {
		return d, e
	}
	if decoder.Decode(new(any)) != io.EOF {
		return d, errors.New("platform: trailing broker descriptor data")
	}
	if d.Version != nativeSystemBrokerSchemaVersion {
		return d, ErrUnsupported
	}
	return d, nil
}

func validateNativeSystemBrokerDescriptor(d nativeSystemBrokerDescriptor, target, path string, roots nativeSystemBrokerRoots) (nativeSystemBrokerPlan, error) {
	p, e := buildNativeSystemBrokerPlan(target, ServiceInstallOptions{Executable: d.ExecutablePath, SystemBroker: &d.Options}, roots)
	if e != nil {
		return p, e
	}
	if d.Version != nativeSystemBrokerSchemaVersion || p.ImagePath != d.ExecutablePath || p.DescriptorPath != path {
		return nativeSystemBrokerPlan{}, ErrOperationIdentity
	}
	return p, nil
}

func copyNativeSystemBrokerImage(source io.Reader, dest io.Writer, expected string) error {
	if !validNativeBrokerDigest(expected) {
		return ErrOperationIdentity
	}
	hash := sha256.New()
	n, e := io.Copy(io.MultiWriter(dest, hash), io.LimitReader(source, maxNativeSystemBrokerImageBytes+1))
	if e != nil {
		return e
	}
	if n == 0 || n > maxNativeSystemBrokerImageBytes || hex.EncodeToString(hash.Sum(nil)) != expected {
		return errors.New("platform: system broker image checksum mismatch")
	}
	return nil
}

func verifyNativeSystemBrokerImage(file *os.File, expected string) error {
	if file == nil {
		return ErrOperationIdentity
	}
	if _, e := file.Seek(0, io.SeekStart); e != nil {
		return e
	}
	if e := copyNativeSystemBrokerImage(file, io.Discard, expected); e != nil {
		return e
	}
	_, e := file.Seek(0, io.SeekStart)
	return e
}

// ReadNativeSystemServiceBrokerDescriptor requires native machine ownership,
// immutable image bytes and the actual native service manager's current job.
func ReadNativeSystemServiceBrokerDescriptor(path string) (NativeSystemServiceBrokerOptions, error) {
	return readNativeSystemServiceBrokerDescriptor(path)
}

func RunNativeSystemSupervisor(ctx context.Context, options NativeSystemServiceBrokerOptions, run func(context.Context) error) error {
	if ctx == nil || run == nil || !validNativeBrokerDigest(options.ExecutableSHA256) {
		return errors.New("platform: exact native broker runtime required")
	}
	if e := ctx.Err(); e != nil {
		return e
	}
	return runNativeSystemSupervisor(ctx, options, func(ctx context.Context) error {
		if err := applyNativeOperatorResourceStorageRoot(options.GlobalResourceStorageRoot); err != nil {
			return err
		}
		return run(ctx)
	})
}

func nativeSystemBrokerDescriptorPath(image string, o NativeSystemServiceBrokerOptions) string {
	raw, _ := json.Marshal(o)
	// An old root-broker descriptor must never collide with the operator-host
	// descriptor for the same image/options. Publication remains no-replace.
	digest := sha256.Sum256(append([]byte(fmt.Sprintf("native-supervisor-v%d\x00", nativeSystemBrokerSchemaVersion)), raw...))
	return filepath.Join(filepath.Dir(image), fmt.Sprintf("broker-%x.json", digest))
}

func canonicalNativeBrokerWindowsUserPrincipal(value string) bool {
	if !strings.HasPrefix(value, "windows:S-1-5-21-") && !strings.HasPrefix(value, "windows:S-1-12-1-") {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(value, "windows:"), "-")
	if len(parts) != 8 {
		return false
	}
	for _, part := range parts[1:] {
		n, e := strconv.ParseUint(part, 10, 32)
		if e != nil || strconv.FormatUint(n, 10) != part {
			return false
		}
	}
	return true
}

// A descriptor is a tiny object of typed operator metadata. Reject duplicate
// keys (including Go's case-insensitive field aliases) before typed decoding.
func validateNativeBrokerJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var read func(int) error
	read = func(depth int) error {
		if depth > 8 {
			return errors.New("platform: broker JSON nesting exceeds bound")
		}
		token, e := decoder.Token()
		if e != nil {
			return e
		}
		delimiter, compound := token.(json.Delim)
		if !compound {
			return nil
		}
		if delimiter != '{' && delimiter != '[' {
			return ErrOperationIdentity
		}
		closing := json.Delim(']')
		if delimiter == '{' {
			closing = '}'
		}
		seen := map[string]bool{}
		for decoder.More() {
			if delimiter == '{' {
				key, e := decoder.Token()
				if e != nil {
					return e
				}
				name, ok := key.(string)
				name = strings.ToLower(name)
				if !ok || seen[name] {
					return errors.New("platform: duplicate broker descriptor field")
				}
				seen[name] = true
			}
			if e = read(depth + 1); e != nil {
				return e
			}
		}
		end, e := decoder.Token()
		if e != nil || end != closing {
			return ErrOperationIdentity
		}
		return nil
	}
	if e := read(0); e != nil {
		return e
	}
	if _, e := decoder.Token(); e != io.EOF {
		return errors.New("platform: trailing broker descriptor data")
	}
	return nil
}
