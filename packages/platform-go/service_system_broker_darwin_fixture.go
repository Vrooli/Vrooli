package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

// OS-independent engine: native adapters supply ownership, Mach-O/mapped-image
// observations and launchctl. Hermetic tests use the same publication/recovery
// path with isolated roots and injected native observations.
type darwinSystemBrokerEngine struct {
	roots           nativeSystemBrokerRoots
	privileged      func() error
	check           func(string, bool) error
	operator        func(NativeSystemServiceBrokerOptions) error
	closure         func(*os.File) error
	preflight       func() error
	run             func(context.Context, ...string) ([]byte, error)
	query           func(context.Context, string) (darwinJobSnapshot, bool, error)
	runtimeIdentity func(NativeSystemServiceBrokerOptions) error
	validate        func(RenderedArtifact) Verdict
	process         func(context.Context, nativeSystemBrokerPlan, int) error
}

func (e darwinSystemBrokerEngine) plan(o ServiceInstallOptions) (nativeSystemBrokerPlan, error) {
	return buildNativeSystemBrokerPlan("darwin", o, e.roots)
}

func (e darwinSystemBrokerEngine) directory(path string, mode os.FileMode) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return ErrOperationIdentity
	}
	parts := []string{}
	for p := path; ; p = filepath.Dir(p) {
		parts = append(parts, p)
		if p == filepath.Dir(p) {
			break
		}
	}
	for n := len(parts) - 1; n >= 0; n-- {
		p := parts[n]
		if _, err := os.Lstat(p); os.IsNotExist(err) {
			perm := os.FileMode(0o755)
			if p == path {
				perm = mode
			}
			if err = os.Mkdir(p, perm); err != nil {
				return err
			}
			if err = darwinOwnedSyncDir(filepath.Dir(p)); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		if err := e.check(p, true); err != nil {
			return fmt.Errorf("platform: untrusted broker directory %s: %w", p, err)
		}
	}
	return nil
}

func (e darwinSystemBrokerEngine) bounded(path string, limit int64, readonly bool) ([]byte, error) {
	if err := e.check(path, false); err != nil {
		return nil, err
	}
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(before, st) || !st.Mode().IsRegular() || readonly && st.Mode().Perm()&0o222 != 0 {
		return nil, ErrOperationIdentity
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, ErrOperationIdentity
	}
	after, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !os.SameFile(st, after) {
		return nil, ErrOperationIdentity
	}
	if err = e.check(path, false); err != nil {
		return nil, err
	}
	return data, nil
}

func (e darwinSystemBrokerEngine) image(p nativeSystemBrokerPlan) error {
	if err := e.check(p.ImagePath, false); err != nil {
		return err
	}
	f, err := os.Open(p.ImagePath)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	if st.Mode().Perm()&0o222 != 0 || st.Mode().Perm()&0o111 == 0 {
		return ErrOperationIdentity
	}
	if err = verifyNativeSystemBrokerImage(f, p.SHA256); err != nil {
		return err
	}
	return e.closure(f)
}

func (e darwinSystemBrokerEngine) descriptor(path string) (nativeSystemBrokerPlan, error) {
	return e.descriptorWithImageProof(path, true)
}
func (e darwinSystemBrokerEngine) descriptorWithImageProof(path string, imageProof bool) (nativeSystemBrokerPlan, error) {
	data, err := e.bounded(path, 64<<10, true)
	if err != nil {
		return nativeSystemBrokerPlan{}, err
	}
	d, err := decodeNativeSystemServiceBrokerDescriptor(data)
	if err != nil {
		return nativeSystemBrokerPlan{}, err
	}
	p, err := validateNativeSystemBrokerDescriptor(d, "darwin", path, e.roots)
	if err != nil {
		return p, err
	}
	dirs := []string{filepath.Dir(p.ImagePath), e.roots.Install, filepath.Dir(e.roots.Unit)}
	for _, dir := range dirs {
		if err = e.existingDirectory(dir); err != nil {
			return p, err
		}
	}
	if imageProof {
		if err = e.image(p); err != nil {
			return p, err
		}
	}
	if err = e.operator(d.Options); err != nil {
		return p, err
	}
	return p, nil
}

func (e darwinSystemBrokerEngine) existingDirectory(path string) error {
	for p := path; ; p = filepath.Dir(p) {
		if err := e.check(p, true); err != nil {
			return fmt.Errorf("platform: untrusted broker directory %s: %w", p, err)
		}
		if p == filepath.Dir(p) {
			return nil
		}
	}
}

func (e darwinSystemBrokerEngine) publish(path string, data []byte, mode os.FileMode) error {
	if _, err := os.Lstat(path); err == nil {
		old, err := e.bounded(path, int64(len(data))+1, mode&0o222 == 0)
		if err != nil {
			return err
		}
		if !bytes.Equal(old, data) {
			return fmt.Errorf("%w: immutable native broker artifact differs", ErrOperationIdentity)
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".broker-publish-*")
	if err != nil {
		return err
	}
	temp := f.Name()
	defer os.Remove(temp)
	if _, err = f.Write(data); err == nil {
		err = f.Chmod(mode)
	}
	if err == nil {
		err = f.Sync()
	}
	ce := f.Close()
	if err != nil {
		return err
	}
	if ce != nil {
		return ce
	}
	if err = darwinSystemBrokerPublishExclusive(temp, path); err != nil {
		return err
	}
	return darwinOwnedSyncDir(filepath.Dir(path))
}

func (e darwinSystemBrokerEngine) stage(p nativeSystemBrokerPlan, source *os.File) error {
	for _, dir := range []struct {
		path string
		mode os.FileMode
	}{{e.roots.Install, 0o755}, {filepath.Dir(p.ImagePath), 0o755}, {filepath.Dir(p.UnitPath), 0o755}} {
		if err := e.directory(dir.path, dir.mode); err != nil {
			return err
		}
	}
	if _, err := os.Lstat(p.ImagePath); os.IsNotExist(err) {
		if _, err = source.Seek(0, io.SeekStart); err != nil {
			return err
		}
		f, err := os.CreateTemp(filepath.Dir(p.ImagePath), ".image-stage-*")
		if err != nil {
			return err
		}
		temp := f.Name()
		defer os.Remove(temp)
		err = copyNativeSystemBrokerImage(source, f, p.SHA256)
		if err == nil {
			err = f.Chmod(0o555)
		}
		if err == nil {
			err = f.Sync()
		}
		ce := f.Close()
		if err != nil {
			return err
		}
		if ce != nil {
			return ce
		}
		if err = darwinSystemBrokerPublishExclusive(temp, p.ImagePath); err != nil {
			return err
		}
		if err = darwinOwnedSyncDir(filepath.Dir(p.ImagePath)); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if err := e.image(p); err != nil {
		return err
	}
	return e.publish(p.DescriptorPath, p.Descriptor, 0o444)
}

func darwinBrokerDescriptorFromPlist(data []byte) (string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var key string
	var args []string
	label := ""
	inArgs := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "key" {
				if err = decoder.DecodeElement(&key, &t); err != nil {
					return "", err
				}
			}
			if t.Name.Local == "array" && key == "ProgramArguments" {
				if inArgs || len(args) > 0 {
					return "", ErrOperationIdentity
				}
				inArgs = true
			}
			if t.Name.Local == "string" {
				var value string
				if err = decoder.DecodeElement(&value, &t); err != nil {
					return "", err
				}
				if inArgs {
					args = append(args, value)
				} else if key == "Label" {
					if label != "" {
						return "", ErrOperationIdentity
					}
					label = value
				}
			}
		case xml.EndElement:
			if t.Name.Local == "array" {
				inArgs = false
			}
		}
	}
	if label != "com.vrooli.runtime-supervisor" || len(args) != 3 || args[1] != NativeSystemSupervisorVerb || !filepath.IsAbs(args[2]) {
		return "", ErrOperationIdentity
	}
	return args[2], nil
}

func (e darwinSystemBrokerEngine) installed() (nativeSystemBrokerPlan, bool, error) {
	data, err := e.bounded(e.roots.Unit, 1<<20, false)
	if os.IsNotExist(err) {
		return nativeSystemBrokerPlan{}, false, nil
	}
	if err != nil {
		return nativeSystemBrokerPlan{}, false, err
	}
	path, err := darwinBrokerDescriptorFromPlist(data)
	if err != nil {
		return nativeSystemBrokerPlan{}, false, err
	}
	p, err := e.descriptor(path)
	if err != nil {
		return p, false, err
	}
	rendered, err := RenderLaunchd(p.Definition)
	if err != nil {
		return p, false, err
	}
	if !bytes.Equal(data, []byte(rendered.Primary().Content)) {
		return p, false, ErrOperationIdentity
	}
	return p, true, nil
}

func (e darwinSystemBrokerEngine) snapshot(ctx context.Context) (darwinJobSnapshot, bool, error) {
	if e.query == nil {
		return darwinJobSnapshot{}, false, ErrUnsupported
	}
	// A missing slot is still a valid manager absence query. If a job exists
	// without its sealed descriptor, native identity remains unknown.
	expected := ""
	if data, err := e.bounded(e.roots.Unit, 1<<20, false); err == nil {
		expected = string(data)
	} else if !os.IsNotExist(err) {
		return darwinJobSnapshot{}, false, err
	}
	return e.query(ctx, expected)
}

func (e darwinSystemBrokerEngine) bootout(ctx context.Context, p nativeSystemBrokerPlan, present bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Remove(p.UnitPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := darwinOwnedSyncDir(filepath.Dir(p.UnitPath)); err != nil {
		return err
	}
	if present {
		out, err := e.run(ctx, "bootout", "system/com.vrooli.runtime-supervisor")
		if err != nil {
			return fmt.Errorf("platform: broker bootout uncertain; retain sealed version: %w: %s", err, out)
		}
	}
	if _, exists, err := e.snapshot(ctx); err != nil {
		return err
	} else if exists {
		return ErrOperationNotQuiescent
	}
	return nil
}

func sameDarwinBrokerOperator(a, b nativeSystemBrokerPlan) bool {
	da, ea := decodeNativeSystemServiceBrokerDescriptor(a.Descriptor)
	db, eb := decodeNativeSystemServiceBrokerDescriptor(b.Descriptor)
	if ea != nil || eb != nil {
		return false
	}
	da.Options.ExecutableSHA256 = ""
	db.Options.ExecutableSHA256 = ""
	return reflect.DeepEqual(da.Options, db.Options)
}

func (e darwinSystemBrokerEngine) settle(ctx context.Context, p nativeSystemBrokerPlan) error {
	for {
		observed, present, err := e.snapshot(ctx)
		if err != nil {
			return err
		}
		if present && observed.State == "running" && observed.PID > 0 {
			if err = e.process(ctx, p, observed.PID); err != nil {
				return err
			}
			after, present, err := e.snapshot(ctx)
			if err != nil {
				return err
			}
			if !present || after.PID != observed.PID || after.State != "running" {
				return ErrOperationIdentity
			}
			return nil
		}
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (e darwinSystemBrokerEngine) install(ctx context.Context, o ServiceInstallOptions) (ServiceInstallResult, error) {
	p, err := e.plan(o)
	if err != nil {
		return ServiceInstallResult{}, err
	}
	result := ServiceInstallResult{UnitName: "com.vrooli.runtime-supervisor", UnitPath: p.UnitPath, Scope: "system", SealedExecutablePath: p.ImagePath, StateRoot: p.StateRoot, LogPath: p.Definition.Logs.Stdout}
	if err = e.privileged(); err != nil {
		return result, err
	}
	if err = e.operator(*o.SystemBroker); err != nil {
		return result, err
	}
	if err = e.preflight(); err != nil {
		return result, err
	}
	source, err := os.Open(p.Source)
	if err != nil {
		return result, err
	}
	defer source.Close()
	st, err := source.Stat()
	if err != nil || !st.Mode().IsRegular() {
		return result, ErrOperationIdentity
	}
	if err = verifyNativeSystemBrokerImage(source, p.SHA256); err != nil {
		return result, err
	}
	if err = e.closure(source); err != nil {
		return result, err
	}
	unlock, err := e.lock(ctx)
	if err != nil {
		return result, fmt.Errorf("platform: lock broker installation: %w", err)
	}
	defer unlock()
	// Check installed identity before staging any candidate or touching a daemon.
	old, exists, err := e.installed()
	if err != nil {
		return result, err
	}
	observed, present, err := e.snapshot(ctx)
	if err != nil {
		return result, err
	}
	if present && !exists {
		old, err = e.journal()
		if err != nil {
			return result, err
		}
		exists = true
	}
	if exists && !sameDarwinBrokerOperator(old, p) {
		return result, ErrOperationIdentity
	}
	if present && observed.PID > 0 {
		if err = e.process(ctx, old, observed.PID); err != nil {
			return result, err
		}
	}
	if err = e.stage(p, source); err != nil {
		return result, fmt.Errorf("platform: stage sealed broker: %w", err)
	}
	artifact, err := RenderLaunchd(p.Definition)
	if err != nil {
		return result, err
	}
	result.Verdict = e.validate(artifact)
	if result.Verdict.State != VerdictAccepted {
		return result, fmt.Errorf("platform: native broker plist is unqualified: %s", result.Verdict.String())
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	if exists {
		if err = e.saveJournal(old); err != nil {
			return result, err
		}
		if err = e.bootout(ctx, old, present); err != nil {
			return result, err
		}
	}
	if err = e.saveJournal(p); err != nil {
		return result, err
	}
	if err = e.publish(p.UnitPath, []byte(artifact.Primary().Content), 0o644); err != nil {
		return result, err
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	if out, err := e.run(ctx, "enable", "system/com.vrooli.runtime-supervisor"); err != nil {
		return result, fmt.Errorf("platform: broker enable: %w: %s", err, out)
	}
	if out, err := e.run(ctx, "bootstrap", "system", p.UnitPath); err != nil {
		return result, fmt.Errorf("platform: broker bootstrap uncertain; retain sealed artifacts: %w: %s", err, out)
	}
	if observed, present, err := e.snapshot(ctx); err != nil {
		return result, err
	} else if present && observed.State != "running" {
		if out, err := e.run(ctx, "kickstart", "system/com.vrooli.runtime-supervisor"); err != nil {
			return result, fmt.Errorf("platform: broker kickstart: %w: %s", err, out)
		}
	}
	if err = e.settle(ctx, p); err != nil {
		return result, err
	}
	result.Active = true
	return result, nil
}

func (e darwinSystemBrokerEngine) uninstall(ctx context.Context, o ServiceInstallOptions) (ServiceInstallResult, error) {
	p, err := e.plan(o)
	if err != nil {
		return ServiceInstallResult{}, err
	}
	result := ServiceInstallResult{UnitName: "com.vrooli.runtime-supervisor", UnitPath: p.UnitPath, Scope: "system", SealedExecutablePath: p.ImagePath, StateRoot: p.StateRoot, LogPath: p.Definition.Logs.Stdout}
	if err = e.privileged(); err != nil {
		return result, err
	}
	unlock, err := e.lock(ctx)
	if err != nil {
		return result, fmt.Errorf("platform: lock broker installation: %w", err)
	}
	defer unlock()
	old, exists, err := e.installed()
	if err != nil {
		return result, err
	}
	if exists && (!sameDarwinBrokerOperator(old, p) || old.SHA256 != p.SHA256) {
		return result, ErrOperationIdentity
	}
	observed, present, err := e.snapshot(ctx)
	if err != nil {
		return result, err
	}
	if present && !exists {
		old, err = e.journal()
		if err != nil {
			return result, err
		}
		exists = true
		if old.SHA256 != p.SHA256 || !sameDarwinBrokerOperator(old, p) {
			return result, ErrOperationIdentity
		}
	}
	if present && observed.PID > 0 {
		if err = e.process(ctx, old, observed.PID); err != nil {
			return result, err
		}
	}
	if exists {
		if err = e.saveJournal(old); err != nil {
			return result, err
		}
		if err = e.bootout(ctx, old, present); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (e darwinSystemBrokerEngine) start(ctx context.Context, o ServiceInstallOptions) (bool, error) {
	p, err := e.plan(o)
	if err != nil {
		return false, err
	}
	if err = e.privileged(); err != nil {
		return false, err
	}
	unlock, err := e.lock(ctx)
	if err != nil {
		return false, err
	}
	defer unlock()
	old, exists, err := e.installed()
	if err != nil {
		return false, err
	}
	if !exists {
		if _, present, err := e.snapshot(ctx); err != nil {
			return false, err
		} else if present {
			return false, ErrOperationOutcomeLost
		}
		return false, nil
	}
	if old.SHA256 != p.SHA256 || !sameDarwinBrokerOperator(old, p) {
		return false, ErrOperationIdentity
	}
	observed, present, err := e.snapshot(ctx)
	if err != nil {
		return false, err
	}
	if present && observed.PID > 0 {
		if err = e.process(ctx, p, observed.PID); err != nil {
			return false, err
		}
	}
	if out, err := e.run(ctx, "enable", "system/com.vrooli.runtime-supervisor"); err != nil {
		return false, fmt.Errorf("platform: broker enable: %w: %s", err, out)
	}
	if !present {
		if out, err := e.run(ctx, "bootstrap", "system", p.UnitPath); err != nil {
			return false, fmt.Errorf("platform: broker bootstrap uncertain: %w: %s", err, out)
		}
	} else if observed.State != "running" {
		if out, err := e.run(ctx, "kickstart", "system/com.vrooli.runtime-supervisor"); err != nil {
			return false, fmt.Errorf("platform: broker kickstart: %w: %s", err, out)
		}
	}
	if err = e.settle(ctx, p); err != nil {
		return false, err
	}
	return true, nil
}

func darwinBrokerSystemLibrary(path string) bool {
	return filepath.IsAbs(path) && filepath.Clean(path) == path && (path == "/usr/lib" || path == "/System/Library" || strings.HasPrefix(path, "/usr/lib/") || strings.HasPrefix(path, "/System/Library/"))
}

func (e darwinSystemBrokerEngine) lock(ctx context.Context) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := e.directory(e.roots.State, 0o755); err != nil {
		return nil, err
	}
	// This directory holds only a nonsecret readonly descriptor pointer and a
	// private lock. Preserve ordinary operator recovery after boot removal.
	if err := os.Chmod(e.roots.State, 0o755); err != nil {
		return nil, err
	}
	if err := e.check(e.roots.State, true); err != nil {
		return nil, err
	}
	path := filepath.Join(e.roots.State, "native-broker.lock")
	release, err := AcquireFileLockContext(ctx, path)
	if err != nil {
		return nil, err
	}
	if err = e.check(path, false); err != nil {
		release()
		return nil, err
	}
	if err = os.Chmod(path, 0o600); err != nil {
		release()
		return nil, err
	}
	return release, nil
}

func (e darwinSystemBrokerEngine) journal() (nativeSystemBrokerPlan, error) {
	return e.journalWithImageProof(true)
}
func (e darwinSystemBrokerEngine) journalWithImageProof(imageProof bool) (nativeSystemBrokerPlan, error) {
	path := filepath.Join(e.roots.State, "installed-descriptor.json")
	data, err := e.bounded(path, 8192, true)
	if err != nil {
		return nativeSystemBrokerPlan{}, err
	}
	if err = validateNativeBrokerJSON(data); err != nil {
		return nativeSystemBrokerPlan{}, err
	}
	var record struct {
		Descriptor string `json:"descriptor"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&record); err != nil {
		return nativeSystemBrokerPlan{}, err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return nativeSystemBrokerPlan{}, ErrOperationIdentity
	}
	return e.descriptorWithImageProof(record.Descriptor, imageProof)
}

func (e darwinSystemBrokerEngine) saveJournal(p nativeSystemBrokerPlan) error {
	path := filepath.Join(e.roots.State, "installed-descriptor.json")
	if _, err := os.Lstat(path); err == nil {
		if _, err = e.journal(); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	data, err := json.Marshal(struct {
		Descriptor string `json:"descriptor"`
	}{p.DescriptorPath})
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(e.roots.State, ".broker-journal-*")
	if err != nil {
		return err
	}
	temp := f.Name()
	defer os.Remove(temp)
	if _, err = f.Write(data); err == nil {
		err = f.Chmod(0o444)
	}
	if err == nil {
		err = f.Sync()
	}
	ce := f.Close()
	if err != nil {
		return err
	}
	if ce != nil {
		return ce
	}
	if err = os.Rename(temp, path); err != nil {
		return err
	}
	return darwinOwnedSyncDir(e.roots.State)
}

func (e darwinSystemBrokerEngine) runtime(ctx context.Context, path string, pid int) (NativeSystemServiceBrokerOptions, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return NativeSystemServiceBrokerOptions{}, ErrOperationIdentity
	}
	p, err := e.descriptor(path)
	if err != nil {
		return NativeSystemServiceBrokerOptions{}, err
	}
	d, err := decodeNativeSystemServiceBrokerDescriptor(p.Descriptor)
	if err != nil {
		return NativeSystemServiceBrokerOptions{}, err
	}
	if e.runtimeIdentity == nil {
		return NativeSystemServiceBrokerOptions{}, ErrUnsupported
	}
	if err = e.runtimeIdentity(d.Options); err != nil {
		return NativeSystemServiceBrokerOptions{}, err
	}
	installed, exists, err := e.installed()
	if err != nil {
		return NativeSystemServiceBrokerOptions{}, err
	}
	if !exists || installed.DescriptorPath != p.DescriptorPath || installed.SHA256 != p.SHA256 {
		return NativeSystemServiceBrokerOptions{}, ErrOperationIdentity
	}
	observed, present, err := e.snapshot(ctx)
	if err != nil {
		return NativeSystemServiceBrokerOptions{}, err
	}
	if !present || observed.PID != pid || observed.State != "running" {
		return NativeSystemServiceBrokerOptions{}, ErrOperationIdentity
	}
	if err = e.process(ctx, p, pid); err != nil {
		return NativeSystemServiceBrokerOptions{}, err
	}
	after, present, err := e.snapshot(ctx)
	if err != nil {
		return NativeSystemServiceBrokerOptions{}, err
	}
	if !present || after.PID != pid || after.State != "running" {
		return NativeSystemServiceBrokerOptions{}, ErrOperationIdentity
	}
	return d.Options, nil
}

// The sole alias is Apple's fixed root-owned /var -> /private/var. Symlink
// permission bits are conventionally 0777 and do not grant link replacement.
// Its target must still pass the native directory ownership check.
func validateDarwinBrokerMetadata(path string, mode os.FileMode, uid uint32, links uint64, directory bool, target string) error {
	if uid != 0 {
		return ErrOperationIdentity
	}
	if mode&os.ModeSymlink != 0 {
		if directory && path == "/var" && (target == "private/var" || target == "/private/var") {
			return nil
		}
		return ErrOperationIdentity
	}
	if mode.Perm()&0o022 != 0 || mode&(os.ModeSetuid|os.ModeSetgid) != 0 || mode.IsDir() != directory || !directory && (!mode.IsRegular() || links != 1) {
		return ErrOperationIdentity
	}
	return nil
}

// Rights are the native write/append/delete/delete-child/write-xattr/
// write-attribute/write-security/change-owner bits, including inherited and
// inherit-only ALLOW entries. Deny-only Apple protection ACLs stay admissible.
type darwinBrokerACLGrant struct {
	Allow, Root, Owner bool
	Rights             uint32
}

func validateDarwinBrokerACL(entries []darwinBrokerACLGrant) error {
	if len(entries) > 1024 {
		return ErrOperationIdentity
	}
	for _, entry := range entries {
		if entry.Rights&^uint32(255) != 0 {
			return ErrOperationIdentity
		}
		if entry.Allow && !entry.Root && entry.Rights != 0 {
			return ErrOperationIdentity
		}
	}
	return nil
}

// Manager PID observation uses only public sealed slot/image/descriptor data.
// It intentionally requires neither private state access nor foreign process
// image/token/VM observation; the server validates that stronger context itself.
func (e darwinSystemBrokerEngine) supervisorPID(ctx context.Context) (int, error) {
	data, err := e.bounded(e.roots.Unit, 1<<20, false)
	if err != nil {
		return 0, err
	}
	path, err := darwinBrokerDescriptorFromPlist(data)
	if err != nil {
		return 0, err
	}
	plan, err := e.descriptor(path)
	if err != nil {
		return 0, err
	}
	rendered, err := RenderLaunchd(plan.Definition)
	if err != nil {
		return 0, err
	}
	if !bytes.Equal(data, []byte(rendered.Primary().Content)) {
		return 0, ErrOperationIdentity
	}
	before, present, err := e.snapshot(ctx)
	if err != nil {
		return 0, err
	}
	if !present || before.State != "running" || before.PID <= 0 {
		return 0, ErrOperationNotFound
	}
	after, present, err := e.snapshot(ctx)
	if err != nil {
		return 0, err
	}
	if !present || after.State != "running" || after.PID != before.PID {
		return 0, ErrOperationIdentity
	}
	return before.PID, nil
}

// POSIX private modes do not override inherited ALLOW ACLs on Darwin.
func validateDarwinOperatorACL(entries []darwinBrokerACLGrant) error {
	if len(entries) > 1024 {
		return ErrOperationIdentity
	}
	for _, entry := range entries {
		if entry.Rights&^uint32(255) != 0 {
			return ErrOperationIdentity
		}
		if entry.Allow && !entry.Root && !entry.Owner && entry.Rights != 0 {
			return ErrOperationIdentity
		}
	}
	return nil
}
