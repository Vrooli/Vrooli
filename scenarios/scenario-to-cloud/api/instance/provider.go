package instance

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const instanceDiskName = "disk.qcow2"

var (
	ErrInvalidRequest       = errors.New("instance request is invalid")
	ErrProviderUnavailable  = errors.New("instance provider is unavailable")
	ErrInstanceNotFound     = errors.New("instance was not found")
	ErrUnsupportedOperation = errors.New("instance operation is unsupported")
)

type State string

const (
	StatePlanned  State = "planned"
	StateStopped  State = "stopped"
	StateRunning  State = "running"
	StateFailed   State = "failed"
	StateDeleting State = "deleting"
)

type Profile string

const (
	ProfileHeadlessLinux Profile = "headless-linux"
	ProfileDesktopLinux  Profile = "desktop-linux"
)

type Instance struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Provider  string    `json:"provider"`
	State     State     `json:"state"`
	Image     string    `json:"image"`
	Workdir   string    `json:"workdir"`
	Address   string    `json:"address,omitempty"`
	SSHPort   int       `json:"ssh_port,omitempty"`
	Profile   Profile   `json:"profile"`
	PID       int       `json:"pid,omitempty"`
	Command   []string  `json:"command,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	LastError string    `json:"last_error,omitempty"`
}

type Request struct {
	ID            string  `json:"id,omitempty"`
	Name          string  `json:"name"`
	Image         string  `json:"image"`
	Workdir       string  `json:"workdir"`
	Memory        string  `json:"memory,omitempty"`
	CPUs          int     `json:"cpus,omitempty"`
	Profile       Profile `json:"profile"`
	User          string  `json:"user"`
	AuthorizedKey string  `json:"authorized_key"`
	SSHPort       int     `json:"ssh_port,omitempty"`
}

type Plan struct {
	Provider      string   `json:"provider"`
	Available     bool     `json:"available"`
	Executable    string   `json:"executable"`
	ImageTool     string   `json:"image_tool"`
	CloudInitTool string   `json:"cloud_init_tool"`
	Command       []string `json:"command,omitempty"`
	Checks        []string `json:"checks"`
	Warnings      []string `json:"warnings,omitempty"`
	Profile       Profile  `json:"profile"`
}

type Provider interface {
	Name() string
	Plan(context.Context, Request) (Plan, error)
	Create(context.Context, Request) (Instance, error)
	Start(context.Context, Instance) (int, error)
	WaitForSSH(context.Context, Instance) error
	Snapshot(context.Context, Instance, string) error
	Reset(context.Context, Instance, string) error
	Stop(context.Context, Instance) error
	Destroy(context.Context, Instance) error
	Status(context.Context, Instance) (State, error)
}

// LocalQEMUProvider owns the host-local disposable VM lane. It deliberately
// reports missing host tools as a typed readiness failure; callers must not
// silently substitute a container, because systemd and credential-wrap tests
// depend on a real VM.
type LocalQEMUProvider struct {
	Binary         string
	LookPath       func(string) (string, error)
	StartProcess   func(string, []string, string) (int, error)
	StopProcess    func(int) error
	RunImage       func(...string) error
	BuildCloudInit func(string, string, string, string) error
}

func (p LocalQEMUProvider) Name() string { return "local-qemu" }

func (p LocalQEMUProvider) executable() string {
	if value := strings.TrimSpace(p.Binary); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv("VROOLI_QEMU_BINARY")); value != "" {
		return value
	}
	return "qemu-system-x86_64"
}

func (p LocalQEMUProvider) lookPath(name string) (string, error) {
	if p.LookPath != nil {
		return p.LookPath(name)
	}
	return exec.LookPath(name)
}

func (p LocalQEMUProvider) Plan(_ context.Context, request Request) (Plan, error) {
	if err := validate(request); err != nil {
		return Plan{Provider: p.Name(), Available: false}, err
	}
	plan := Plan{Provider: p.Name(), Executable: p.executable(), Profile: normalizedProfile(request.Profile), Checks: []string{"image-readable", "qemu-executable", "qemu-img-executable", "cloud-localds-executable", "provisioning-input"}}
	if _, err := os.Stat(request.Image); err != nil {
		plan.Warnings = append(plan.Warnings, "image is not readable: "+err.Error())
	} else {
		plan.Checks[0] = "image-readable: ok"
	}
	executable, err := p.lookPath(plan.Executable)
	if err != nil {
		plan.Warnings = append(plan.Warnings, "qemu executable is unavailable: "+err.Error())
		return plan, fmt.Errorf("%w: %s: %v", ErrProviderUnavailable, plan.Executable, err)
	}
	plan.Executable = executable
	plan.Checks[1] = "qemu-executable: ok"
	imageTool, err := p.lookPath("qemu-img")
	if err != nil {
		plan.Warnings = append(plan.Warnings, "qemu-img executable is unavailable: "+err.Error())
		return plan, fmt.Errorf("%w: qemu-img: %v", ErrProviderUnavailable, err)
	}
	plan.ImageTool = imageTool
	plan.Checks[2] = "qemu-img-executable: ok"
	cloudInitTool, err := p.lookPath("cloud-localds")
	if err != nil {
		plan.Warnings = append(plan.Warnings, "cloud-localds executable is unavailable: "+err.Error())
		return plan, fmt.Errorf("%w: cloud-localds: %v", ErrProviderUnavailable, err)
	}
	plan.CloudInitTool = cloudInitTool
	plan.Checks[3] = "cloud-localds-executable: ok"
	if !validLinuxUsername(request.User) || strings.TrimSpace(request.AuthorizedKey) == "" {
		plan.Warnings = append(plan.Warnings, "a non-root user and authorized key are required")
	} else {
		plan.Checks[4] = "provisioning-input: ok"
	}
	plan.Available = len(plan.Warnings) == 0
	if !plan.Available {
		return plan, fmt.Errorf("%w: local-qemu prerequisites are incomplete", ErrProviderUnavailable)
	}
	plan.Command = commandFor(plan.Executable, request)
	return plan, nil
}

func (p LocalQEMUProvider) Create(ctx context.Context, request Request) (Instance, error) {
	if err := ctx.Err(); err != nil {
		return Instance{}, err
	}
	plan, err := p.Plan(ctx, request)
	if err != nil {
		return Instance{}, err
	}
	if err := validateProvisioning(request); err != nil {
		return Instance{}, err
	}
	now := time.Now().UTC()
	if err := os.MkdirAll(request.Workdir, 0o700); err != nil {
		return Instance{}, fmt.Errorf("create instance workdir: %w", err)
	}
	diskImage := filepath.Join(request.Workdir, instanceDiskName)
	if _, err := os.Stat(diskImage); err == nil {
		return Instance{}, fmt.Errorf("%w: instance disk already exists at %s", ErrInvalidRequest, diskImage)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Instance{}, fmt.Errorf("inspect instance disk: %w", err)
	}
	// Specify the backing format explicitly. Newer qemu-img versions reject an
	// implicit backing format because probing an untrusted image can be unsafe
	// and can produce a disk whose backing relationship is not reproducible.
	if err := p.runImage("create", "-f", "qcow2", "-F", "qcow2", "-b", filepath.Clean(request.Image), diskImage); err != nil {
		return Instance{}, fmt.Errorf("create instance disk: %w", err)
	}
	cleanupDisk := true
	defer func() {
		if cleanupDisk {
			_ = os.Remove(diskImage)
		}
	}()
	userData := filepath.Join(request.Workdir, "cloud-init-user-data")
	metaData := filepath.Join(request.Workdir, "cloud-init-meta-data")
	cloudInitISO := filepath.Join(request.Workdir, "cloud-init.iso")
	cloudConfig := "#cloud-config\nusers:\n  - name: " + yamlString(request.User) + "\n    sudo: ALL=(ALL) NOPASSWD:ALL\n    groups: [sudo]\n    shell: /bin/bash\n    lock_passwd: true\n    ssh_authorized_keys:\n      - " + yamlString(request.AuthorizedKey) + "\nssh_pwauth: false\n"
	if err := os.WriteFile(userData, []byte(cloudConfig), 0o600); err != nil {
		return Instance{}, fmt.Errorf("write cloud-init user data: %w", err)
	}
	meta := "instance-id: " + yamlString(request.Name) + "\nlocal-hostname: " + yamlString(request.Name) + "\n"
	if err := os.WriteFile(metaData, []byte(meta), 0o600); err != nil {
		return Instance{}, fmt.Errorf("write cloud-init metadata: %w", err)
	}
	buildCloudInit := p.BuildCloudInit
	if buildCloudInit == nil {
		buildCloudInit = func(tool, iso, user, meta string) error {
			return exec.Command(tool, iso, user, meta).Run()
		}
	}
	if err := buildCloudInit(plan.CloudInitTool, cloudInitISO, userData, metaData); err != nil {
		return Instance{}, fmt.Errorf("build cloud-init seed: %w", err)
	}
	diskRequest := request
	diskRequest.Image = diskImage
	command := commandFor(plan.Executable, diskRequest)
	command = append(command, "-drive", "file="+cloudInitISO+",if=virtio,format=raw,readonly=on")
	port := request.SSHPort
	if port == 0 {
		port = 2222
	}
	instance := Instance{ID: request.ID, Name: request.Name, Provider: p.Name(), State: StateStopped, Image: diskImage, Workdir: request.Workdir, Address: "127.0.0.1", SSHPort: port, Profile: normalizedProfile(request.Profile), Command: command, CreatedAt: now, UpdatedAt: now}
	cleanupDisk = false
	return instance, nil
}

func (p LocalQEMUProvider) Start(ctx context.Context, instance Instance) (int, error) {
	if instance.State == StateRunning {
		return instance.PID, nil
	}
	if len(instance.Command) == 0 {
		return 0, fmt.Errorf("%w: instance has no launch command", ErrInvalidRequest)
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	start := p.StartProcess
	if start == nil {
		start = func(name string, args []string, workdir string) (int, error) {
			cmd := exec.Command(name, args...)
			cmd.Dir = workdir
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Start(); err != nil {
				return 0, err
			}
			return cmd.Process.Pid, nil
		}
	}
	pid, err := start(instance.Command[0], instance.Command[1:], instance.Workdir)
	if err != nil {
		return 0, fmt.Errorf("start local-qemu: %w", err)
	}
	return pid, nil
}

func (p LocalQEMUProvider) Stop(_ context.Context, instance Instance) error {
	if instance.PID == 0 {
		return nil
	}
	if p.StopProcess != nil {
		return p.StopProcess(instance.PID)
	}
	process, err := os.FindProcess(instance.PID)
	if err != nil {
		return err
	}
	if err := process.Signal(os.Interrupt); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	}
	return nil
}

func (p LocalQEMUProvider) runImage(args ...string) error {
	if p.RunImage != nil {
		return p.RunImage(args...)
	}
	binary, err := p.lookPath("qemu-img")
	if err != nil {
		return err
	}
	return exec.Command(binary, args...).Run()
}

func (p LocalQEMUProvider) WaitForSSH(ctx context.Context, instance Instance) error {
	if instance.Address == "" || instance.SSHPort == 0 {
		return fmt.Errorf("%w: instance has no SSH address", ErrInvalidRequest)
	}
	address := net.JoinHostPort(instance.Address, fmt.Sprint(instance.SSHPort))
	for {
		conn, err := net.DialTimeout("tcp", address, time.Second)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func (p LocalQEMUProvider) Snapshot(_ context.Context, instance Instance, name string) error {
	if strings.TrimSpace(name) == "" || instance.Image == "" {
		return fmt.Errorf("%w: snapshot name and image are required", ErrInvalidRequest)
	}
	if instance.State == StateRunning {
		return fmt.Errorf("%w: stop the instance before creating a snapshot", ErrInvalidRequest)
	}
	return p.runImage("snapshot", "-c", name, filepath.Clean(instance.Image))
}

func (p LocalQEMUProvider) Reset(_ context.Context, instance Instance, name string) error {
	if strings.TrimSpace(name) == "" || instance.Image == "" {
		return fmt.Errorf("%w: reset snapshot and image are required", ErrInvalidRequest)
	}
	if instance.State == StateRunning {
		return fmt.Errorf("%w: stop the instance before resetting a snapshot", ErrInvalidRequest)
	}
	return p.runImage("snapshot", "-a", name, filepath.Clean(instance.Image))
}

func (p LocalQEMUProvider) Destroy(ctx context.Context, instance Instance) error {
	if err := p.Stop(ctx, instance); err != nil {
		return err
	}
	diskImage := filepath.Join(instance.Workdir, instanceDiskName)
	if filepath.Clean(instance.Image) != filepath.Clean(diskImage) {
		return fmt.Errorf("%w: instance image is not an owned local-QEMU disk", ErrInvalidRequest)
	}
	for _, path := range []string{
		diskImage,
		filepath.Join(instance.Workdir, "cloud-init.iso"),
		filepath.Join(instance.Workdir, "cloud-init-user-data"),
		filepath.Join(instance.Workdir, "cloud-init-meta-data"),
	} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove instance artifact %s: %w", path, err)
		}
	}
	return nil
}

func (p LocalQEMUProvider) Status(_ context.Context, instance Instance) (State, error) {
	return instance.State, nil
}

func validate(request Request) error {
	if strings.TrimSpace(request.Name) == "" || strings.TrimSpace(request.Image) == "" || strings.TrimSpace(request.Workdir) == "" {
		return fmt.Errorf("%w: name, image, and workdir are required", ErrInvalidRequest)
	}
	if strings.ContainsAny(request.Name+request.User+request.AuthorizedKey, "\r\n") {
		return fmt.Errorf("%w: name, user, and authorized key must be single-line values", ErrInvalidRequest)
	}
	if request.AuthorizedKey != "" && len(strings.Fields(request.AuthorizedKey)) < 2 {
		return fmt.Errorf("%w: authorized_key must contain an SSH key type and key", ErrInvalidRequest)
	}
	if request.CPUs < 0 {
		return fmt.Errorf("%w: cpus cannot be negative", ErrInvalidRequest)
	}
	if normalizedProfile(request.Profile) != ProfileHeadlessLinux && normalizedProfile(request.Profile) != ProfileDesktopLinux {
		return fmt.Errorf("%w: unsupported profile %q", ErrInvalidRequest, request.Profile)
	}
	if request.SSHPort < 0 || request.SSHPort > 65535 {
		return fmt.Errorf("%w: ssh_port must be between 1 and 65535", ErrInvalidRequest)
	}
	return nil
}

func validateProvisioning(request Request) error {
	if !validLinuxUsername(request.User) {
		return fmt.Errorf("%w: user must be a valid non-root Linux username", ErrInvalidRequest)
	}
	if strings.TrimSpace(request.AuthorizedKey) == "" {
		return fmt.Errorf("%w: authorized_key is required", ErrInvalidRequest)
	}
	return nil
}

func validLinuxUsername(value string) bool {
	if value == "" || value == "root" || len(value) > 32 {
		return false
	}
	for index, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9' && index > 0) || (char == '_' || char == '-' && index > 0) {
			continue
		}
		return false
	}
	return value[0] >= 'a' && value[0] <= 'z'
}

func yamlString(value string) string {
	return strconv.Quote(value)
}

func normalizedProfile(profile Profile) Profile {
	if profile == "" {
		return ProfileHeadlessLinux
	}
	return profile
}

func commandFor(executable string, request Request) []string {
	memory := request.Memory
	if memory == "" {
		memory = "2048"
	}
	cpus := request.CPUs
	if cpus == 0 {
		cpus = 2
	}
	port := request.SSHPort
	if port == 0 {
		port = 2222
	}
	command := []string{executable, "-machine", "q35,accel=kvm:tcg", "-m", memory, "-smp", fmt.Sprint(cpus), "-drive", "file=" + filepath.Clean(request.Image) + ",if=virtio,format=qcow2", "-netdev", fmt.Sprintf("user,id=net0,hostfwd=tcp::%d-:22", port), "-device", "virtio-net-pci,netdev=net0"}
	if normalizedProfile(request.Profile) == ProfileDesktopLinux {
		return append(command, "-display", "gtk")
	}
	return append(command, "-nographic")
}
