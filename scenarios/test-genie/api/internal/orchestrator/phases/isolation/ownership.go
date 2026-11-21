package isolation

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	containerapi "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/go-connections/nat"
)

// Allocation is private owner state, never public evidence. It is persisted
// before allocation so a replacement can retire an uncertain creation exactly.
type Allocation struct {
	Kind        string `json:"kind"`
	RunID       string `json:"run_id"`
	Name        string `json:"name,omitempty"`
	Root        string `json:"root,omitempty"`
	StagingRoot string `json:"staging_root,omitempty"`
	AdminURL    string `json:"admin_url,omitempty"`
	Retired     bool   `json:"retired,omitempty"`
	ImageID     string `json:"image_id,omitempty"`
}

// Only this private namespace may contain interrupted, unmarked preparation.
// Final backing paths always require the completed exact owner marker.
func ensureSQLiteStagingNamespace(parent string) error {
	if err := privateSQLiteDirectory(parent); err != nil {
		return err
	}
	namespace := filepath.Join(parent, ".staging")
	if err := os.Mkdir(namespace, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	return privateSQLiteDirectory(namespace)
}

func privateSQLiteDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return errors.New("sqlite owner namespace must be a private directory without aliases")
	}
	return nil
}

func syncSQLiteDirectory(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return file.Sync()
}

func validateSQLiteStage(a Allocation) error {
	if a.StagingRoot != filepath.Join(filepath.Dir(a.Root), ".staging", a.RunID) || filepath.Base(a.RunID) != a.RunID || a.RunID == "." || a.RunID == ".." || a.RunID == "" {
		return errors.New("sqlite staging path does not match the retained private allocation plan")
	}
	if err := privateSQLiteDirectory(filepath.Dir(a.Root)); err != nil {
		return err
	}
	return privateSQLiteDirectory(filepath.Dir(a.StagingRoot))
}

func prepareSQLiteDirectory(a Allocation, checkpoint func(string) error) error {
	if err := validateSQLiteStage(a); err != nil {
		return err
	}
	if err := os.Mkdir(a.StagingRoot, 0700); err != nil {
		return err
	}
	if checkpoint != nil {
		if err := checkpoint("staging-created"); err != nil {
			return err
		}
	}
	marker, err := os.OpenFile(filepath.Join(a.StagingRoot, ".seed-owner"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, writeErr := marker.WriteString(a.RunID)
	if err := errors.Join(writeErr, marker.Sync(), marker.Close()); err != nil {
		return err
	}
	if err := syncSQLiteDirectory(a.StagingRoot); err != nil {
		return err
	}
	if checkpoint != nil {
		if err := checkpoint("marker-synced"); err != nil {
			return err
		}
	}
	if err := publishSQLiteDirectory(a.StagingRoot, a.Root); err != nil {
		return err
	}
	if checkpoint != nil {
		if err := checkpoint("published"); err != nil {
			return err
		}
	}
	return syncSQLiteDirectory(filepath.Dir(a.Root))
}

func retireSQLiteStaging(a Allocation) error {
	if err := validateSQLiteStage(a); err != nil {
		return err
	}
	if err := privateSQLiteDirectory(a.StagingRoot); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	entries, err := os.ReadDir(a.StagingRoot)
	if err != nil {
		return err
	}
	// No target or seed may see preparation. Its only possible file is the
	// small marker; never recursively delete an unknown or published tree.
	if len(entries) > 1 || (len(entries) == 1 && entries[0].Name() != ".seed-owner") {
		return errors.New("sqlite staging contains content outside its retained preparation plan")
	}
	if len(entries) == 1 {
		marker := filepath.Join(a.StagingRoot, ".seed-owner")
		info, err := os.Lstat(marker)
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > int64(len(a.RunID)) {
			return errors.New("sqlite staging marker is not bounded private preparation")
		}
		value, err := os.ReadFile(marker)
		if err != nil || !strings.HasPrefix(a.RunID, string(value)) {
			return errors.New("sqlite staging marker differs from its retained owner")
		}
		if err := os.Remove(marker); err != nil {
			return err
		}
	}
	if err := os.Remove(a.StagingRoot); err != nil {
		return err
	}
	return syncSQLiteDirectory(filepath.Dir(a.StagingRoot))
}

// startPrivateContainer uses only an already-installed image. Its lifecycle
// is owned by the durable seed journal, with neither AutoRemove nor a
// process-session reaper that could delete backing before target clear.
func (m *Manager) startPrivateContainer(ctx context.Context, runID, kind, image, port string, env []string) (string, string, func(context.Context) error, error) {
	provider, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", "", nil, err
	}
	defer provider.Close()
	host := "127.0.0.1"
	if daemon := provider.DaemonHost(); strings.HasPrefix(daemon, "tcp://") {
		parsed, err := url.Parse(daemon)
		if err != nil {
			return "", "", nil, err
		}
		ip := net.ParseIP(parsed.Hostname())
		if parsed.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback()) {
			return "", "", nil, errors.New("private seed allocation requires a local owned Docker daemon")
		}
		host = parsed.Hostname()
	}
	imageInfo, _, err := provider.ImageInspectWithRaw(ctx, image)
	if err != nil {
		return "", "", nil, fmt.Errorf("private seed image %s is not installed; provision through the resource owner: %w", image, err)
	}
	if !strings.HasPrefix(imageInfo.ID, "sha256:") || len(imageInfo.ID) != 71 {
		return "", "", nil, errors.New("private seed image has no immutable local identity")
	}
	allocation := Allocation{Kind: "container", RunID: runID, Name: "tg-seed-" + runID + "-" + kind, ImageID: imageInfo.ID}
	if _, err := provider.ContainerInspect(ctx, allocation.Name); err == nil {
		return "", "", nil, errors.New("private seed container name is already occupied")
	} else if !errdefs.IsNotFound(err) {
		return "", "", nil, err
	}
	if err := m.track(allocation); err != nil {
		return "", "", nil, err
	}
	cleanup := func(c context.Context) error { return RetireAllocation(c, allocation) }
	resourcePort := nat.Port(port)
	created, err := provider.ContainerCreate(ctx, &containerapi.Config{Image: imageInfo.ID, Env: env, Labels: map[string]string{"vrooli.test-genie.seed-owner": runID}, ExposedPorts: nat.PortSet{resourcePort: {}}}, &containerapi.HostConfig{AutoRemove: false, PortBindings: nat.PortMap{resourcePort: []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: ""}}}}, nil, nil, allocation.Name)
	if err != nil {
		return "", "", cleanup, err
	}
	if err := provider.ContainerStart(ctx, created.ID, types.ContainerStartOptions{}); err != nil {
		return "", "", cleanup, err
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		observed, err := provider.ContainerInspect(ctx, created.ID)
		if err != nil {
			return "", "", cleanup, err
		}
		if observed.Config == nil || observed.Config.Labels["vrooli.test-genie.seed-owner"] != runID || strings.TrimPrefix(observed.Name, "/") != allocation.Name || observed.Image != allocation.ImageID {
			return "", "", cleanup, errors.New("private seed container identity changed during preparation")
		}
		if observed.NetworkSettings != nil {
			for _, binding := range observed.NetworkSettings.Ports[resourcePort] {
				if binding.HostPort == "" {
					continue
				}
				probe, err := net.DialTimeout("tcp", net.JoinHostPort(host, binding.HostPort), 100*time.Millisecond)
				if err == nil {
					probe.Close()
					return host, binding.HostPort, cleanup, nil
				}
			}
		}
		select {
		case <-ctx.Done():
			return "", "", cleanup, ctx.Err()
		case <-ticker.C:
		}
	}
}

func retirePostgresDatabase(ctx context.Context, a Allocation) error {
	// Database names are private UUID-derived identifiers. The additional owner
	// comment prevents a create-name collision from authorizing deletion.
	if len(sanitize(a.RunID)) < 6 || !strings.Contains(a.Name, sanitize(a.RunID)[:6]) {
		return errors.New("postgres allocation name does not match owner")
	}
	query := fmt.Sprintf("SELECT COALESCE(shobj_description(oid,'pg_database'),'') FROM pg_database WHERE datname='%s'", strings.ReplaceAll(a.Name, "'", "''"))
	command := exec.CommandContext(ctx, "psql", "-d", a.AdminURL, "-v", "ON_ERROR_STOP=1", "-At", "-c", query)
	command.WaitDelay = time.Second
	output, err := command.Output()
	if err != nil {
		return err
	}
	marker := strings.TrimSpace(string(output))
	if marker == "" {
		// Distinguish absent from an unmarked/foreign existing database.
		check := exec.CommandContext(ctx, "psql", "-d", a.AdminURL, "-At", "-c", fmt.Sprintf("SELECT count(*) FROM pg_database WHERE datname='%s'", strings.ReplaceAll(a.Name, "'", "''")))
		check.WaitDelay = time.Second
		found, err := check.Output()
		if err != nil {
			return err
		}
		if strings.TrimSpace(string(found)) == "0" {
			return nil
		}
	}
	if marker != "test-genie-seed:"+a.RunID {
		return errors.New("postgres database owner marker does not match seed allocation")
	}
	return execCommandContext(ctx, nil, "psql", a.AdminURL, "-v", "ON_ERROR_STOP=1", "-c", fmt.Sprintf("DROP DATABASE IF EXISTS %q", a.Name))()
}

func (m *Manager) track(a Allocation) error {
	if m.cfg.OnAllocation == nil {
		return nil
	}
	return m.cfg.OnAllocation(a)
}

// RetireAllocation never searches by executable, prefix or shared Redis DB.
// Container ownership is checked against the exact random name and label.
func RetireAllocation(ctx context.Context, a Allocation) error {
	if a.Retired {
		return nil
	}
	if a.RunID == "" {
		return errors.New("allocation has no owner identity")
	}
	switch a.Kind {
	case "sqlite":
		if !filepath.IsAbs(a.Root) || filepath.Base(a.Root) != "test-genie-playbooks-"+a.RunID {
			return errors.New("invalid private sqlite allocation identity")
		}
		if a.StagingRoot != "" {
			if err := retireSQLiteStaging(a); err != nil {
				return err
			}
		}
		marker, err := os.ReadFile(filepath.Join(a.Root, ".seed-owner"))
		if errors.Is(err, os.ErrNotExist) {
			if _, rootErr := os.Lstat(a.Root); errors.Is(rootErr, os.ErrNotExist) {
				return nil
			}
		}
		if err != nil || string(marker) != a.RunID {
			return errors.New("private sqlite allocation owner marker is missing or mismatched")
		}
		return os.RemoveAll(a.Root)
	case "postgres-database":
		if a.Name == "" || !strings.HasPrefix(a.Name, "tg_pb_") || a.AdminURL == "" {
			return errors.New("invalid private postgres allocation identity")
		}
		return retirePostgresDatabase(ctx, a)
	case "container":
		if a.Name != "tg-seed-"+a.RunID+"-postgres" && a.Name != "tg-seed-"+a.RunID+"-redis" {
			return errors.New("invalid container allocation identity")
		}
		provider, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			return err
		}
		defer provider.Close()
		container, err := provider.ContainerInspect(ctx, a.Name)
		if errdefs.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if strings.TrimPrefix(container.Name, "/") != a.Name || container.Config == nil || container.Config.Labels["vrooli.test-genie.seed-owner"] != a.RunID {
			return errors.New("container seed owner identity mismatch")
		}
		if a.ImageID != "" && container.Image != a.ImageID {
			return errors.New("container seed image identity mismatch")
		}
		return provider.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{Force: true, RemoveVolumes: true})
	default:
		return errors.New("unknown seed allocation kind")
	}
}
