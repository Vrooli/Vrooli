package resources

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/config"
)

func openRGBNativeCapsulePath(id string) (string, error) {
	if !managedUpdateKey.MatchString(id) {
		return "", errors.New("invalid native launch capsule id")
	}
	root, err := OpenRGBNativeResourceRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "launch", id, "capsule.json"), nil
}
func openRGBNativeCapsuleEndpoint(id string) (string, error) {
	if !managedUpdateKey.MatchString(id) {
		return "", errors.New("invalid native launch capsule id")
	}
	root, err := OpenRGBNativeResourceRoot()
	if err != nil {
		return "", err
	}
	return openRGBNativeCapsuleEndpointAt(root, id)
}
func openRGBNativeCapsuleEndpointAt(root, id string) (string, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || !managedUpdateKey.MatchString(id) {
		return "", errors.New("invalid native capsule endpoint namespace")
	}
	hash := sha256.Sum256([]byte(id))
	path := filepath.Join(root, "ipc", fmt.Sprintf("claim-%x.sock", hash[:12]))
	if len(path) > 100 {
		return "", errors.New("native claim endpoint exceeds native Unix socket path limit")
	}
	return path, nil
}
func nativeCapsuleOwnerPrincipal(principal string) string {
	if strings.HasPrefix(principal, "uid:") {
		return "unix:" + strings.TrimPrefix(principal, "uid:")
	}
	if strings.HasPrefix(principal, "sid:") {
		return "windows:" + strings.TrimPrefix(principal, "sid:")
	}
	return ""
}
func readOpenRGBNativeCapsule(ctx context.Context, path string) (openRGBNativeCapsule, error) {
	var c openRGBNativeCapsule
	root, err := openRGBNativeResourceRootFromCapsulePath(path)
	if err != nil {
		return c, err
	}
	owner, err := readOpenRGBNativeOwnerAt(ctx, root)
	if err != nil {
		return c, err
	}
	policy, err := owner.policyFor(path)
	if err != nil {
		return c, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return c, err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 64<<10 {
		return c, errors.New("invalid native capsule file")
	}
	if err = config.VerifyPrivatePath(path, info, policy); err != nil {
		return c, err
	}
	file, err := os.Open(path)
	if err != nil {
		return c, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return c, errors.New("native capsule changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(file, (64<<10)+1))
	if err != nil || len(data) > 64<<10 {
		return c, errors.New("native capsule exceeds size bound")
	}
	if err = validateManagedUpdateJSON(data); err != nil {
		return c, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&c); err != nil {
		return c, err
	}
	if err = decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return c, errors.New("native capsule has trailing data")
	}
	if err = c.validate(); err != nil {
		return c, err
	}
	expected := filepath.Join(root, "launch", c.ID, "capsule.json")
	if !nativeCapsulePathWithin(filepath.Join(root, "runtime"), c.RuntimeRoot) || c.RuntimeRoot == filepath.Join(root, "runtime") {
		return c, errors.New("capsule runtime differs from its frozen resource namespace")
	}
	if expected != path || c.Account != owner.NativeAccount || !nativeCapsuleOwnerAllowed(owner.OwnerPrincipal, c.OwnerPrincipal) {
		return c, errors.New("native capsule differs from enrolled owner")
	}
	endpoint, err := openRGBNativeCapsuleEndpointAt(root, c.ID)
	if err != nil || endpoint != c.Endpoint {
		return c, errors.New("native capsule endpoint changed")
	}
	return c, nil
}

func publishOpenRGBNativeCapsule(c openRGBNativeCapsule) (string, error) {
	if err := c.validate(); err != nil {
		return "", err
	}
	root, err := openRGBNativeResourceRootFromRuntime(c.RuntimeRoot)
	if err != nil {
		return "", err
	}
	path := filepath.Join(root, "launch", c.ID, "capsule.json")
	owner, err := readOpenRGBNativeOwnerAt(context.Background(), root)
	if err != nil {
		return "", err
	}
	if c.Account != owner.NativeAccount || !nativeCapsuleOwnerAllowed(owner.OwnerPrincipal, c.OwnerPrincipal) {
		return "", errors.New("native launch owner differs from enrollment")
	}
	policy, err := owner.policyFor(path)
	if err != nil {
		return "", err
	}
	if _, err = os.Lstat(path); err == nil {
		return "", errors.New("native capsule is immutable and already exists")
	} else if !os.IsNotExist(err) {
		return "", err
	}
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	if err = config.WritePrivateFileDurable(path, data, 0600, policy); err != nil {
		return "", err
	}
	return path, nil
}

func materializeOpenRGBNativeCapsule(material openRGBNativeMaterial, invocation managedUpdateInvocation) error {
	frozenRoot, err := openRGBNativeResourceRootFromRuntime(invocation.RuntimeRoot)
	if err != nil {
		return err
	}
	if material.Runtime.StateDir != filepath.Join(frozenRoot, "state") || material.Invocation.RuntimeRoot != invocation.RuntimeRoot {
		return errors.New("native materialization differs from frozen resource namespace")
	}
	owner, err := readOpenRGBNativeOwnerAt(context.Background(), frozenRoot)
	if err != nil {
		return err
	}
	root := invocation.RuntimeRoot
	for _, path := range append([]string{root, filepath.Dir(invocation.ConfigPath)}, nativeCapsuleRuntimeDirs(invocation)...) {
		if !nativeCapsulePathWithin(root, path) {
			return errors.New("native invocation writes outside admitted runtime")
		}
		policy, err := owner.policyFor(path)
		if err != nil {
			return err
		}
		if policy.RuntimeReadOnly || policy.OwnerPrincipal != owner.RuntimePrincipal {
			return errors.New("native invocation is outside backend-owned configuration")
		}
		if err = config.EnsurePrivateDirectory(path, policy); err != nil {
			return err
		}
	}
	if !nativeCapsulePathWithin(root, invocation.ConfigPath) {
		return errors.New("native config escapes backend runtime")
	}
	policy, err := owner.policyFor(invocation.ConfigPath)
	if err != nil {
		return err
	}
	return config.WritePrivateFileDurable(invocation.ConfigPath, []byte(invocation.ConfigContent), 0600, policy)
}
func nativeCapsuleRuntimeDirs(invocation managedUpdateInvocation) []string {
	values := managedServiceEnvValues(invocation.Env)
	var paths []string
	for _, key := range []string{"HOME", "RESOURCE_CONFIG_DIR", "RESOURCE_CACHE_DIR", "RESOURCE_LOGS_DIR", "RESOURCE_STATE_DIR", "XDG_CONFIG_HOME", "APPDATA", "LOCALAPPDATA"} {
		if path := values[key]; path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}

// The one-shot owner exchange belongs to the exact enrolled operator. Kernel
// peer identity and process incarnation are verified separately.
func nativeCapsuleOwnerAllowed(owner, actual string) bool {
	return actual != "" && actual == nativeCapsuleOwnerPrincipal(owner)
}

func openRGBNativeResourceRootFromCapsulePath(path string) (string, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.ContainsAny(path, "\x00\r\n") || filepath.Base(path) != "capsule.json" {
		return "", errors.New("native capsule path must be canonical launch metadata")
	}
	launch := filepath.Dir(filepath.Dir(path))
	id := filepath.Base(filepath.Dir(path))
	root := filepath.Dir(launch)
	if filepath.Base(launch) != "launch" || !managedUpdateKey.MatchString(id) || filepath.Dir(root) == root {
		return "", errors.New("native capsule path is outside an exact launch/ID namespace")
	}
	return root, nil
}
func openRGBNativeResourceRootFromRuntime(path string) (string, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.ContainsAny(path, "\x00\r\n") {
		return "", errors.New("native runtime path must be canonical")
	}
	cursor := path
	for depth := 0; depth < 4; depth++ {
		if filepath.Base(cursor) == "runtime" {
			root := filepath.Dir(cursor)
			relative, err := filepath.Rel(cursor, path)
			if err != nil {
				return "", err
			}
			parts := strings.Split(relative, string(filepath.Separator))
			valid := len(parts) == 2 && parts[0] == "initial" || len(parts) == 3 && parts[0] == "managed-update"
			if !valid || filepath.Dir(root) == root {
				return "", errors.New("native runtime lacks a bounded role namespace")
			}
			for _, part := range parts[1:] {
				if !managedUpdateKey.MatchString(part) {
					return "", errors.New("native runtime role identity is invalid")
				}
			}
			return root, nil
		}
		cursor = filepath.Dir(cursor)
	}
	return "", errors.New("native runtime path lacks its frozen resource root")
}
func openRGBNativeCapsuleFrozenEndpoint(c openRGBNativeCapsule) (string, error) {
	root, err := openRGBNativeResourceRootFromRuntime(c.RuntimeRoot)
	if err != nil {
		return "", err
	}
	return openRGBNativeCapsuleEndpointAt(root, c.ID)
}
