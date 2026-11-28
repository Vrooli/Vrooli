// Package wordpress owns local WordPress site containers, their database,
// bounded site snapshots, and recipe application. Callers identify a validated
// site slug; they cannot choose Docker objects, images, mounts or commands.
package wordpress

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/capacity"
)

const (
	ImagePeakBytes          int64 = 5 * 1024 * 1024 * 1024
	ImageDiskFloorBytes     int64 = 120 * 1024 * 1024 * 1024
	SnapshotMaxBytes              = 512 << 20
	ContainerStopSeconds          = 10
	readinessTimeout              = 90 * time.Second
	imageReservationOwner         = "resource:wordpress:images:v1"
	wordpressNetworkPrefix        = "vrooli-wp-"
	databaseVolumePrefix          = "vrooli-wp-dbdata-"
	siteVolumePrefix              = "vrooli-wp-sitedata-"
	databaseContainerPrefix       = "vrooli-wp-db-"
	siteContainerPrefix           = "vrooli-wp-site-"
	labelResource                 = "com.vrooli.resource"
	labelObjectKind               = "com.vrooli.object-kind"
	labelSiteID                   = "com.vrooli.wordpress.site-id"
	resourceLabelValue            = "wordpress"
)

var (
	ErrDockerUnavailable = errors.New("wordpress resource: Docker daemon unavailable")
	ErrImageMissing      = errors.New("wordpress resource: pinned images are not installed; run `vrooli resource wordpress acquire`")
	ErrForeignObject     = errors.New("wordpress resource: refusing to modify an object without matching owner labels")
	ErrInvalidSiteID     = errors.New("wordpress resource: site ID must be a lowercase slug")
	ErrRecipeMissing     = errors.New("wordpress resource: recipe package is not staged by Scenario Dependency Analyzer")
	siteIDPattern        = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,30}$`)
)

// CommandRunner is the only Docker execution seam. Production uses the Docker
// CLI; tests use deterministic fake owners without starting containers.
type CommandRunner interface {
	Run(context.Context, ...string) ([]byte, error)
}

type outputRunner interface {
	RunOutput(context.Context, io.Writer, ...string) error
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("docker %s failed: %w: %s", strings.Join(redactSecrets(args), " "), err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func redactSecrets(args []string) []string {
	redacted := append([]string(nil), args...)
	for index, arg := range redacted {
		upper := strings.ToUpper(arg)
		if strings.Contains(upper, "PASSWORD=") || strings.HasPrefix(strings.ToLower(arg), "--admin_password=") {
			if separator := strings.IndexByte(arg, '='); separator >= 0 {
				redacted[index] = arg[:separator+1] + "[redacted]"
			}
		}
	}
	return redacted
}

func (execRunner) RunOutput(ctx context.Context, writer io.Writer, args ...string) error {
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = writer
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker %s failed: %w", strings.Join(args, " "), err)
	}
	return nil
}

// DiskAdmission provides shared host capacity admission for image downloads.
type DiskAdmission interface {
	Reserve(context.Context, string, string, int64, int64) (DiskLease, error)
}

// DiskLease remains live for the duration of one image acquisition.
type DiskLease interface {
	Heartbeat(context.Context) error
	Release(context.Context) error
}

type capacityAdmission struct{ home string }
type capacityLease struct {
	store *capacity.SQLiteStore
	id    string
}

func (a capacityAdmission) Reserve(ctx context.Context, ownerID, path string, peak, floor int64) (DiskLease, error) {
	store, err := capacity.NewSQLiteStore(ctx, capacity.Config{HomeDir: a.home})
	if err != nil {
		return nil, fmt.Errorf("open host capacity admission owner: %w", err)
	}
	reservation, err := store.ReserveDisk(ctx, ownerID, path, peak, floor)
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	return &capacityLease{store: store, id: reservation.ID}, nil
}

func (l *capacityLease) Heartbeat(ctx context.Context) error { return l.store.HeartbeatDisk(ctx, l.id) }
func (l *capacityLease) Release(ctx context.Context) error {
	return errors.Join(l.store.ReleaseDisk(ctx, l.id), l.store.Close())
}

// Owner owns exactly the three pinned images and containers labeled for a
// validated site ID. The caller does not select Docker names or paths.
type Owner struct {
	Home       string
	DataRoot   string
	Profile    ImageSet
	Runner     CommandRunner
	Admission  DiskAdmission
	HTTPClient *http.Client
	PollEvery  time.Duration
}

// NewOwner creates the standard resource owner for a home directory and a
// previously validated image profile.
func NewOwner(home string, profile ImageSet) *Owner {
	return &Owner{
		Home:      home,
		DataRoot:  filepath.Join(home, ".local", "share", "vrooli", "resources", "wordpress"),
		Profile:   profile,
		Admission: capacityAdmission{home: home},
		PollEvery: time.Second,
	}
}

func (o *Owner) runner() CommandRunner {
	if o.Runner != nil {
		return o.Runner
	}
	return execRunner{}
}

func (o *Owner) client() *http.Client {
	if o.HTTPClient != nil {
		return o.HTTPClient
	}
	return &http.Client{Timeout: 2 * time.Second}
}

// ImageReceipt contains bounded facts about the fixed image acquisition.
type ImageReceipt struct {
	References     []string `json:"references"`
	AlreadyPresent bool     `json:"alreadyPresent"`
	DiskPath       string   `json:"diskPath,omitempty"`
	PeakBytes      int64    `json:"peakBytesReserved,omitempty"`
}

// AcquireImages is the only image-pull path. It reserves capacity on Docker's
// actual data-root filesystem, heartbeats during downloads, and pulls only the
// exact references declared by the resource.
func (o *Owner) AcquireImages(ctx context.Context) (ImageReceipt, error) {
	entries := o.Profile.entries()
	receipt := ImageReceipt{References: make([]string, 0, len(entries))}
	for _, entry := range entries {
		if err := validateImageEntry("profile", entry); err != nil {
			return receipt, err
		}
		receipt.References = append(receipt.References, entry.Reference)
	}
	root, err := o.dockerRoot(ctx)
	if err != nil {
		return receipt, err
	}
	missing := make([]ImageEntry, 0, len(entries))
	for _, entry := range entries {
		present, err := o.imagePresent(ctx, entry.Reference)
		if err != nil {
			return receipt, err
		}
		if !present {
			missing = append(missing, entry)
		}
	}
	if len(missing) == 0 {
		receipt.AlreadyPresent = true
		return receipt, nil
	}
	if o.Admission == nil {
		return receipt, errors.New("wordpress resource: disk admission owner is unavailable; refusing image acquisition")
	}
	lease, err := o.Admission.Reserve(ctx, imageReservationOwner, root, ImagePeakBytes, ImageDiskFloorBytes)
	if err != nil {
		return receipt, fmt.Errorf("admit WordPress image acquisition on %s: %w", root, err)
	}
	receipt.DiskPath, receipt.PeakBytes = root, ImagePeakBytes
	defer func() { _ = lease.Release(context.Background()) }()
	for _, entry := range missing {
		if err := runWithHeartbeat(ctx, lease, func(pullCtx context.Context) error {
			_, runErr := o.runner().Run(pullCtx, "pull", entry.Reference)
			return runErr
		}); err != nil {
			return receipt, fmt.Errorf("acquire pinned WordPress %s image: %w", entry.Version, err)
		}
		if present, err := o.imagePresent(ctx, entry.Reference); err != nil || !present {
			if err != nil {
				return receipt, err
			}
			return receipt, fmt.Errorf("Docker did not retain the acquired pinned WordPress image %q", entry.Reference)
		}
	}
	return receipt, nil
}

func runWithHeartbeat(ctx context.Context, lease DiskLease, operation func(context.Context) error) error {
	operationCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	interval := 30 * time.Second
	heartbeatErr := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-operationCtx.Done():
				return
			case <-ticker.C:
				if err := lease.Heartbeat(operationCtx); err != nil {
					select {
					case heartbeatErr <- err:
					default:
					}
					cancel()
					return
				}
			}
		}
	}()
	err := operation(operationCtx)
	cancel()
	<-done
	select {
	case err := <-heartbeatErr:
		return fmt.Errorf("maintain WordPress image capacity admission: %w", err)
	default:
	}
	return err
}

func (o *Owner) dockerRoot(ctx context.Context) (string, error) {
	if _, err := o.runner().Run(ctx, "info"); err != nil {
		return "", errors.Join(ErrDockerUnavailable, err)
	}
	output, err := o.runner().Run(ctx, "info", "--format", "{{.DockerRootDir}}")
	if err != nil {
		return "", errors.Join(ErrDockerUnavailable, err)
	}
	root := strings.TrimSpace(string(output))
	if !filepath.IsAbs(root) {
		return "", fmt.Errorf("wordpress resource: Docker reported an invalid data root %q", root)
	}
	return root, nil
}

func (o *Owner) imagePresent(ctx context.Context, reference string) (bool, error) {
	output, err := o.runner().Run(ctx, "image", "inspect", "--format", "{{.Id}}", reference)
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(string(output)) != "", nil
}

// SiteStatus is a bounded state receipt; it contains no passwords or site
// content.
type SiteStatus struct {
	SiteID    string `json:"siteId"`
	Status    string `json:"status"`
	Database  string `json:"database"`
	WordPress string `json:"wordpress"`
	URL       string `json:"url,omitempty"`
	RecipeID  string `json:"recipeId,omitempty"`
}

// Start creates or starts one named site and returns its private-network URL.
func (o *Owner) Start(ctx context.Context, siteID string) (SiteStatus, error) {
	if err := validateSiteID(siteID); err != nil {
		return SiteStatus{}, err
	}
	if err := o.ensureImages(ctx); err != nil {
		return SiteStatus{}, err
	}
	paths, err := o.instancePaths(siteID, true)
	if err != nil {
		return SiteStatus{}, err
	}
	secrets, err := o.ensureSecrets(paths)
	if err != nil {
		return SiteStatus{}, err
	}
	networkName, databaseName, siteName := objectNames(siteID)
	if err := o.ensureNetwork(ctx, networkName, siteID); err != nil {
		return SiteStatus{}, err
	}
	if err := o.ensureVolume(ctx, databaseVolumePrefix+siteID, siteID, "database-data"); err != nil {
		return SiteStatus{}, err
	}
	if err := o.ensureVolume(ctx, siteVolumePrefix+siteID, siteID, "site-data"); err != nil {
		return SiteStatus{}, err
	}
	_, err = o.ensureContainer(ctx, databaseName, siteID, "database", func() []string {
		return []string{
			"run", "--detach", "--name", databaseName,
			"--network", networkName, "--network-alias", "database",
			"--label", labelResource + "=" + resourceLabelValue,
			"--label", labelObjectKind + "=database",
			"--label", labelSiteID + "=" + siteID,
			"--env", "MARIADB_ROOT_PASSWORD=" + secrets.RootPassword,
			"--env", "MARIADB_DATABASE=wordpress",
			"--env", "MARIADB_USER=wordpress",
			"--env", "MARIADB_PASSWORD=" + secrets.DatabasePassword,
			"--mount", "type=volume,src=" + databaseVolumePrefix + siteID + ",dst=/var/lib/mysql",
			"--memory", "1g", "--cpus", "1", "--pids-limit", "256",
			o.Profile.Database.Reference,
		}
	}())
	if err != nil {
		return SiteStatus{}, err
	}
	if err := o.waitDatabase(ctx, databaseName, secrets.RootPassword); err != nil {
		return SiteStatus{}, err
	}
	site, err := o.ensureContainer(ctx, siteName, siteID, "site", func() []string {
		return []string{
			"run", "--detach", "--name", siteName,
			"--network", networkName, "--network-alias", "wordpress",
			"--label", labelResource + "=" + resourceLabelValue,
			"--label", labelObjectKind + "=site",
			"--label", labelSiteID + "=" + siteID,
			"--mount", "type=volume,src=" + siteVolumePrefix + siteID + ",dst=/var/www/html",
			"--env", "WORDPRESS_DB_HOST=database:3306",
			"--env", "WORDPRESS_DB_USER=wordpress",
			"--env", "WORDPRESS_DB_PASSWORD=" + secrets.DatabasePassword,
			"--env", "WORDPRESS_DB_NAME=wordpress",
			"--env", "WP_ENVIRONMENT_TYPE=local",
			"--memory", "1g", "--cpus", "1", "--pids-limit", "256",
			o.Profile.Site.Reference,
		}
	}())
	if err != nil {
		return SiteStatus{}, err
	}
	url, err := o.siteURL(ctx, siteName, site)
	if err != nil {
		return SiteStatus{}, err
	}
	if err := o.waitHTTP(ctx, url+"/wp-login.php"); err != nil {
		return SiteStatus{}, err
	}
	if err := o.ensureWordPressInstalled(ctx, siteName, url, secrets); err != nil {
		return SiteStatus{}, err
	}
	return SiteStatus{SiteID: siteID, Status: "running", Database: "running", WordPress: "ready", URL: url}, nil
}

func (o *Owner) ensureWordPressInstalled(ctx context.Context, siteName, siteURL string, secrets instanceSecrets) error {
	baseArgs := o.wpCLIArgs(siteName, secrets.DatabasePassword, true)
	args := append(append([]string{}, baseArgs...), "wp", "core", "is-installed", "--allow-root")
	if _, err := o.runner().Run(ctx, args...); err == nil {
		return nil
	}
	install := append(append([]string{}, baseArgs...), "wp", "core", "install", "--url="+siteURL, "--title=K3 WordPress Ground", "--admin_user=site-admin", "--admin_password="+secrets.AdminPassword,
		"--admin_email=site-admin@example.invalid", "--skip-email", "--allow-root")
	if _, err := o.runner().Run(ctx, install...); err != nil {
		return fmt.Errorf("install the isolated local WordPress site through WP-CLI: %w", err)
	}
	if _, err := o.runner().Run(ctx, args...); err != nil {
		return fmt.Errorf("verify local WordPress installation through WP-CLI: %w", err)
	}
	return nil
}

func (o *Owner) wpCLIArgs(siteName, databasePassword string, joinNetwork bool) []string {
	args := []string{"run", "--rm"}
	if joinNetwork {
		args = append(args, "--network", wordpressNetworkPrefix+strings.TrimPrefix(siteName, siteContainerPrefix))
	}
	args = append(args, "--volumes-from", siteName+":ro",
		"--env", "WORDPRESS_DB_HOST=database:3306", "--env", "WORDPRESS_DB_USER=wordpress", "--env", "WORDPRESS_DB_NAME=wordpress", "--env", "WORDPRESS_DB_PASSWORD="+databasePassword,
		o.Profile.CLI.Reference)
	return args
}

func (o *Owner) ensureImages(ctx context.Context) error {
	for _, entry := range o.Profile.entries() {
		present, err := o.imagePresent(ctx, entry.Reference)
		if err != nil {
			return err
		}
		if !present {
			return ErrImageMissing
		}
	}
	return nil
}

func (o *Owner) ensureNetwork(ctx context.Context, networkName, siteID string) error {
	labels, exists, err := o.inspectNetwork(ctx, networkName)
	if err != nil {
		return err
	}
	if exists {
		if labels[labelResource] != resourceLabelValue || labels[labelSiteID] != siteID {
			return ErrForeignObject
		}
		return nil
	}
	_, err = o.runner().Run(ctx, "network", "create", "--internal", "--label", labelResource+"="+resourceLabelValue, "--label", labelSiteID+"="+siteID, networkName)
	if err != nil {
		return fmt.Errorf("create WordPress private site network: %w", err)
	}
	return nil
}

func (o *Owner) ensureVolume(ctx context.Context, name, siteID, kind string) error {
	labels, exists, err := o.inspectVolume(ctx, name)
	if err != nil {
		return err
	}
	if !exists {
		if _, err := o.runner().Run(ctx, "volume", "create", "--label", labelResource+"="+resourceLabelValue, "--label", labelObjectKind+"="+kind, "--label", labelSiteID+"="+siteID, name); err != nil {
			return fmt.Errorf("create owned WordPress %s volume: %w", kind, err)
		}
		labels, exists, err = o.inspectVolume(ctx, name)
		if err != nil {
			return err
		}
	}
	if !exists || labels[labelResource] != resourceLabelValue || labels[labelObjectKind] != kind || labels[labelSiteID] != siteID {
		return ErrForeignObject
	}
	return nil
}

func (o *Owner) inspectVolume(ctx context.Context, name string) (map[string]string, bool, error) {
	output, err := o.runner().Run(ctx, "volume", "inspect", "--format", "{{json .Labels}}", name)
	if err != nil {
		message := strings.ToLower(err.Error())
		if strings.Contains(message, "no such volume") || strings.Contains(message, "no such object") || strings.Contains(message, "volume "+strings.ToLower(name)+" not found") {
			return nil, false, nil
		}
		return nil, false, err
	}
	var labels map[string]string
	if json.Unmarshal(output, &labels) != nil {
		return nil, false, errors.New("Docker returned invalid WordPress volume labels")
	}
	return labels, true, nil
}

func (o *Owner) inspectNetwork(ctx context.Context, name string) (map[string]string, bool, error) {
	output, err := o.runner().Run(ctx, "network", "inspect", "--format", "{{json .Labels}}", name)
	if err != nil {
		message := strings.ToLower(err.Error())
		if strings.Contains(message, "no such network") || strings.Contains(message, "no such object") || strings.Contains(message, "network "+strings.ToLower(name)+" not found") {
			return nil, false, nil
		}
		return nil, false, err
	}
	var labels map[string]string
	if json.Unmarshal(output, &labels) != nil {
		return nil, false, errors.New("Docker returned invalid WordPress network labels")
	}
	return labels, true, nil
}

func (o *Owner) ensureContainer(ctx context.Context, name, siteID, kind string, createArgs []string) (containerInfo, error) {
	info, exists, err := o.inspectContainer(ctx, name)
	if err != nil {
		return containerInfo{}, err
	}
	if exists {
		if !ownsContainer(info, siteID, kind) {
			return containerInfo{}, ErrForeignObject
		}
		if info.State.Status != "running" {
			if _, err := o.runner().Run(ctx, "start", name); err != nil {
				return containerInfo{}, fmt.Errorf("start owned WordPress %s container: %w", kind, err)
			}
			info, exists, err = o.inspectContainer(ctx, name)
			if err != nil {
				return containerInfo{}, err
			}
			if !exists || !ownsContainer(info, siteID, kind) || info.State.Status != "running" {
				return containerInfo{}, fmt.Errorf("restarted WordPress %s container did not attest its owner labels and running state", kind)
			}
		}
		return info, nil
	}
	if _, err := o.runner().Run(ctx, createArgs...); err != nil {
		return containerInfo{}, fmt.Errorf("create owned WordPress %s container: %w", kind, err)
	}
	info, exists, err = o.inspectContainer(ctx, name)
	if err != nil {
		return containerInfo{}, err
	}
	if !exists || !ownsContainer(info, siteID, kind) {
		return containerInfo{}, fmt.Errorf("WordPress %s container did not attest its owner labels", kind)
	}
	return info, nil
}

// Stop stops only the matching site's verified containers and keeps its data,
// network and recipe state for the next start.
func (o *Owner) Stop(ctx context.Context, siteID string) (SiteStatus, error) {
	if err := validateSiteID(siteID); err != nil {
		return SiteStatus{}, err
	}
	_, databaseName, siteName := objectNames(siteID)
	for _, target := range []struct{ name, kind string }{{siteName, "site"}, {databaseName, "database"}} {
		info, exists, err := o.inspectContainer(ctx, target.name)
		if err != nil {
			return SiteStatus{}, err
		}
		if !exists {
			continue
		}
		if !ownsContainer(info, siteID, target.kind) {
			return SiteStatus{}, ErrForeignObject
		}
		if info.State.Status == "running" {
			if _, err := o.runner().Run(ctx, "stop", "--time", fmt.Sprint(ContainerStopSeconds), target.name); err != nil {
				return SiteStatus{}, fmt.Errorf("stop owned WordPress %s container: %w", target.kind, err)
			}
		}
	}
	return SiteStatus{SiteID: siteID, Status: "stopped", Database: "stopped", WordPress: "stopped"}, nil
}

// Reset removes only the site's owner-verified containers, network and exact
// dedicated data directory, then recreates a clean site. Snapshots are kept.
func (o *Owner) Reset(ctx context.Context, siteID string) (SiteStatus, error) {
	if err := validateSiteID(siteID); err != nil {
		return SiteStatus{}, err
	}
	if _, err := o.Stop(ctx, siteID); err != nil {
		return SiteStatus{}, err
	}
	networkName, databaseName, siteName := objectNames(siteID)
	for _, target := range []struct{ name, kind string }{{siteName, "site"}, {databaseName, "database"}} {
		info, exists, err := o.inspectContainer(ctx, target.name)
		if err != nil {
			return SiteStatus{}, err
		}
		if !exists {
			continue
		}
		if !ownsContainer(info, siteID, target.kind) {
			return SiteStatus{}, ErrForeignObject
		}
		if _, err := o.runner().Run(ctx, "rm", "--force", target.name); err != nil {
			return SiteStatus{}, fmt.Errorf("remove owned WordPress %s container: %w", target.kind, err)
		}
	}
	labels, exists, err := o.inspectNetwork(ctx, networkName)
	if err != nil {
		return SiteStatus{}, err
	}
	if exists {
		if labels[labelResource] != resourceLabelValue || labels[labelSiteID] != siteID {
			return SiteStatus{}, ErrForeignObject
		}
		if _, err := o.runner().Run(ctx, "network", "rm", networkName); err != nil {
			return SiteStatus{}, fmt.Errorf("remove owned WordPress site network: %w", err)
		}
	}
	for _, target := range []struct{ name, kind string }{{siteVolumePrefix + siteID, "site-data"}, {databaseVolumePrefix + siteID, "database-data"}} {
		labels, exists, err := o.inspectVolume(ctx, target.name)
		if err != nil {
			return SiteStatus{}, err
		}
		if !exists {
			continue
		}
		if labels[labelResource] != resourceLabelValue || labels[labelObjectKind] != target.kind || labels[labelSiteID] != siteID {
			return SiteStatus{}, ErrForeignObject
		}
		if _, err := o.runner().Run(ctx, "volume", "rm", target.name); err != nil {
			return SiteStatus{}, fmt.Errorf("remove owned WordPress %s volume: %w", target.kind, err)
		}
	}
	paths, err := o.instancePaths(siteID, false)
	if err != nil {
		return SiteStatus{}, err
	}
	if err := o.resetSiteData(ctx, siteID, paths); err != nil {
		return SiteStatus{}, err
	}
	return o.Start(ctx, siteID)
}

func (o *Owner) resetSiteData(ctx context.Context, siteID string, paths sitePaths) error {
	for _, legacy := range []string{filepath.Join(paths.Root, "database"), filepath.Join(paths.Root, "site")} {
		info, err := os.Lstat(legacy)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("refusing to reset legacy WordPress data through unsafe path %s", legacy)
		}
		uid, gid, err := directoryOwnerID(info)
		if err != nil {
			return err
		}
		if _, err := o.runner().Run(ctx, "run", "--rm", "--user", fmt.Sprintf("%d:%d", uid, gid), "--network", "none", "--mount", "type=bind,src="+legacy+",dst=/legacy", "--entrypoint", "/bin/sh", o.Profile.CLI.Reference, "-c", "rm -rf /legacy/* /legacy/.[!.]* /legacy/..?*"); err != nil {
			return fmt.Errorf("clear legacy WordPress bind mount %s through the owner container: %w", legacy, err)
		}
		if err := os.Remove(legacy); err != nil {
			return fmt.Errorf("remove empty legacy WordPress data directory %s: %w", legacy, err)
		}
	}
	if err := rejectSymlinkAncestors(paths.Root); err != nil {
		return err
	}
	if err := verifyPathInside(o.DataRoot, paths.Root); err != nil {
		return err
	}
	if err := os.Remove(paths.Secrets); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove WordPress site credential file: %w", err)
	}
	if err := os.Remove(paths.Root); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove empty WordPress site metadata directory: %w", err)
	}
	return nil
}

// Status returns Docker and HTTP readiness for one site without exposing site
// configuration or database credentials.
func (o *Owner) Status(ctx context.Context, siteID string) (SiteStatus, error) {
	if err := validateSiteID(siteID); err != nil {
		return SiteStatus{}, err
	}
	_, databaseName, siteName := objectNames(siteID)
	db, dbExists, err := o.inspectContainer(ctx, databaseName)
	if err != nil {
		return SiteStatus{}, err
	}
	site, siteExists, err := o.inspectContainer(ctx, siteName)
	if err != nil {
		return SiteStatus{}, err
	}
	status := SiteStatus{SiteID: siteID, Status: "stopped", Database: "stopped", WordPress: "stopped"}
	if dbExists {
		if !ownsContainer(db, siteID, "database") {
			return SiteStatus{}, ErrForeignObject
		}
		status.Database = db.State.Status
	}
	if siteExists {
		if !ownsContainer(site, siteID, "site") {
			return SiteStatus{}, ErrForeignObject
		}
		status.WordPress = site.State.Status
	}
	if !dbExists || !siteExists || db.State.Status != "running" || site.State.Status != "running" {
		return status, nil
	}
	url, err := o.siteURL(ctx, siteName, site)
	if err != nil {
		return SiteStatus{}, err
	}
	if err := o.checkHTTP(ctx, url+"/wp-login.php"); err != nil {
		status.Status = "starting"
		status.URL = url
		return status, nil
	}
	status.Status, status.URL = "running", url
	return status, nil
}

// Snapshot captures a bounded MariaDB-consistent SQL export and WordPress
// plugin/content tree in one atomically published gzip tar archive.
func (o *Owner) Snapshot(ctx context.Context, siteID string) (receipt SnapshotReceipt, returnErr error) {
	if err := validateSiteID(siteID); err != nil {
		return SnapshotReceipt{}, err
	}
	status, err := o.Status(ctx, siteID)
	if err != nil {
		return SnapshotReceipt{}, err
	}
	if status.Status != "running" {
		return SnapshotReceipt{}, fmt.Errorf("WordPress site %q must be running and ready to snapshot", siteID)
	}
	_, _, siteName := objectNames(siteID)
	paths, err := o.instancePaths(siteID, false)
	if err != nil {
		return SnapshotReceipt{}, err
	}
	secrets, err := o.ensureSecrets(paths)
	if err != nil {
		return SnapshotReceipt{}, err
	}
	if _, err := o.runner().Run(ctx, "pause", siteName); err != nil {
		return SnapshotReceipt{}, fmt.Errorf("pause owned WordPress site for a consistent snapshot: %w", err)
	}
	defer func() {
		unpauseCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if _, err := o.runner().Run(unpauseCtx, "unpause", siteName); err != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("resume owned WordPress site after snapshot: %w", err))
		}
	}()
	snapshotDir := filepath.Join(o.DataRoot, "snapshots")
	if err := ensureSafeDir(o.DataRoot, snapshotDir); err != nil {
		return SnapshotReceipt{}, err
	}
	stage, err := os.MkdirTemp(snapshotDir, ".wordpress-snapshot-*")
	if err != nil {
		return SnapshotReceipt{}, fmt.Errorf("create WordPress snapshot staging directory: %w", err)
	}
	defer os.RemoveAll(stage)
	dumpPath := filepath.Join(stage, "database.sql")
	dump, err := os.OpenFile(dumpPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return SnapshotReceipt{}, err
	}
	boundedDump := &boundedWriter{Writer: dump, remaining: SnapshotMaxBytes}
	command := append(o.wpCLIArgs(siteName, secrets.DatabasePassword, true), "wp", "db", "export", "-")
	if writer, ok := o.runner().(outputRunner); ok {
		err = writer.RunOutput(ctx, boundedDump, command...)
	} else {
		var output []byte
		output, err = o.runner().Run(ctx, command...)
		if err == nil {
			_, err = boundedDump.Write(output)
		}
	}
	closeErr := dump.Close()
	if err != nil {
		return SnapshotReceipt{}, fmt.Errorf("export WordPress database through WP-CLI: %w", err)
	}
	if closeErr != nil {
		return SnapshotReceipt{}, closeErr
	}
	if boundedDump.exceeded {
		return SnapshotReceipt{}, fmt.Errorf("WordPress SQL dump exceeds %d-byte snapshot limit", SnapshotMaxBytes)
	}
	contentPath := filepath.Join(stage, "wp-content.tar")
	content, err := os.OpenFile(contentPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return SnapshotReceipt{}, err
	}
	boundedContent := &boundedWriter{Writer: content, remaining: SnapshotMaxBytes}
	contentCommand := []string{"run", "--rm", "--volumes-from", siteName + ":ro", o.Profile.CLI.Reference, "sh", "-c", "tar -C /var/www/html/wp-content -cf - ."}
	if writer, ok := o.runner().(outputRunner); ok {
		err = writer.RunOutput(ctx, boundedContent, contentCommand...)
	} else {
		var output []byte
		output, err = o.runner().Run(ctx, contentCommand...)
		if err == nil {
			_, err = boundedContent.Write(output)
		}
	}
	closeErr = content.Close()
	if err != nil {
		return SnapshotReceipt{}, fmt.Errorf("snapshot WordPress content through the owner container: %w", err)
	}
	if closeErr != nil {
		return SnapshotReceipt{}, closeErr
	}
	if boundedContent.exceeded {
		return SnapshotReceipt{}, fmt.Errorf("WordPress content exceeds %d-byte snapshot limit", SnapshotMaxBytes)
	}
	archivePath := filepath.Join(stage, "snapshot.tar.gz")
	archive, err := os.OpenFile(archivePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return SnapshotReceipt{}, err
	}
	hash := sha256.New()
	limitWriter := &boundedWriter{Writer: io.MultiWriter(archive, hash), remaining: SnapshotMaxBytes}
	gz := gzip.NewWriter(limitWriter)
	tarWriter := tar.NewWriter(gz)
	meta, _ := json.Marshal(map[string]string{"siteId": siteID, "createdAt": time.Now().UTC().Format(time.RFC3339Nano), "siteImage": o.Profile.Site.Reference, "databaseImage": o.Profile.Database.Reference})
	writeErr := writeTarBytes(tarWriter, "meta.json", meta)
	if writeErr == nil {
		writeErr = writeTarFile(tarWriter, "database.sql", dumpPath)
	}
	if writeErr == nil {
		writeErr = writeTarFile(tarWriter, "wp-content.tar", contentPath)
	}
	if err := tarWriter.Close(); writeErr == nil {
		writeErr = err
	}
	if err := gz.Close(); writeErr == nil {
		writeErr = err
	}
	if err := archive.Close(); writeErr == nil {
		writeErr = err
	}
	if writeErr != nil {
		return SnapshotReceipt{}, fmt.Errorf("write WordPress snapshot: %w", writeErr)
	}
	if limitWriter.exceeded {
		return SnapshotReceipt{}, fmt.Errorf("WordPress snapshot exceeds %d-byte limit", SnapshotMaxBytes)
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	name := siteID + "-" + stamp + "-" + hex.EncodeToString(hash.Sum(nil)[:6]) + ".tar.gz"
	finalPath := filepath.Join(snapshotDir, name)
	if err := os.Rename(archivePath, finalPath); err != nil {
		return SnapshotReceipt{}, fmt.Errorf("publish WordPress snapshot: %w", err)
	}
	info, err := os.Stat(finalPath)
	if err != nil {
		return SnapshotReceipt{}, err
	}
	return SnapshotReceipt{SiteID: siteID, Path: finalPath, SHA256: hex.EncodeToString(hash.Sum(nil)), Bytes: info.Size()}, nil
}

type SnapshotReceipt struct {
	SiteID string `json:"siteId"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

// InstallDeliveryDestination installs the first-party, local-only MU plugin
// from the caller's pinned repository artifact. Third-party recipe packages
// remain subject to SDA; this method never fetches or installs them.
func (o *Owner) InstallDeliveryDestination(ctx context.Context, siteID, sourcePath, scriptPath string) (DestinationReceipt, error) {
	if err := validateSiteID(siteID); err != nil {
		return DestinationReceipt{}, err
	}
	info, exists, err := o.inspectContainer(ctx, siteContainerPrefix+siteID)
	if err != nil {
		return DestinationReceipt{}, err
	}
	if !exists {
		return DestinationReceipt{}, errors.New("WordPress destination install requires the matching owned site to be running")
	}
	if !ownsContainer(info, siteID, "site") {
		return DestinationReceipt{}, ErrForeignObject
	}
	if info.State.Status != "running" {
		return DestinationReceipt{}, errors.New("WordPress destination install requires the matching owned site to be running")
	}
	fileInfo, err := os.Lstat(sourcePath)
	if err != nil {
		return DestinationReceipt{}, fmt.Errorf("inspect first-party WordPress destination source: %w", err)
	}
	if !fileInfo.Mode().IsRegular() || fileInfo.Size() < 256 || fileInfo.Size() > 128<<10 {
		return DestinationReceipt{}, errors.New("WordPress destination source must be a bounded regular file")
	}
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return DestinationReceipt{}, err
	}
	if !bytes.Contains(content, []byte("Plugin Name: Vrooli WordPress Delivery Destination")) || !bytes.Contains(content, []byte("vrooli/v1")) {
		return DestinationReceipt{}, errors.New("WordPress destination source does not match the Vrooli local delivery plugin contract")
	}
	scriptInfo, err := os.Lstat(scriptPath)
	if err != nil {
		return DestinationReceipt{}, fmt.Errorf("inspect WordPress form event ID script: %w", err)
	}
	if !scriptInfo.Mode().IsRegular() || scriptInfo.Size() < 64 || scriptInfo.Size() > 32<<10 {
		return DestinationReceipt{}, errors.New("WordPress form event ID source must be a bounded regular file")
	}
	script, err := os.ReadFile(scriptPath)
	if err != nil {
		return DestinationReceipt{}, err
	}
	if !bytes.Contains(script, []byte("vrooli-event-id")) || !bytes.Contains(script, []byte("randomUUID")) {
		return DestinationReceipt{}, errors.New("WordPress form event ID source does not match the first-party script contract")
	}
	container := siteContainerPrefix + siteID
	paths, err := o.instancePaths(siteID, false)
	if err != nil {
		return DestinationReceipt{}, err
	}
	secrets, err := o.ensureSecrets(paths)
	if err != nil {
		return DestinationReceipt{}, err
	}
	if _, err := o.runner().Run(ctx, "exec", container, "mkdir", "-p", "/var/www/html/wp-content/mu-plugins"); err != nil {
		return DestinationReceipt{}, fmt.Errorf("prepare WordPress MU plugin directory: %w", err)
	}
	if _, err := o.runner().Run(ctx, "cp", sourcePath, container+":/var/www/html/wp-content/mu-plugins/vrooli-delivery-destination.php"); err != nil {
		return DestinationReceipt{}, fmt.Errorf("install first-party WordPress delivery destination: %w", err)
	}
	if _, err := o.runner().Run(ctx, "cp", scriptPath, container+":/var/www/html/wp-content/mu-plugins/vrooli-form-event-id.js"); err != nil {
		return DestinationReceipt{}, fmt.Errorf("install first-party WordPress form event ID script: %w", err)
	}
	if _, err := o.runner().Run(ctx, "exec", container, "php", "-l", "/var/www/html/wp-content/mu-plugins/vrooli-delivery-destination.php"); err != nil {
		return DestinationReceipt{}, fmt.Errorf("validate installed WordPress delivery destination: %w", err)
	}
	if err := o.updateSiteOption(ctx, container, secrets.DatabasePassword, "vrooli_resource_site_id", siteID); err != nil {
		return DestinationReceipt{}, fmt.Errorf("bind first-party delivery destination to the owned WordPress site ID: %w", err)
	}
	url, err := o.siteURL(ctx, container, info)
	if err != nil {
		return DestinationReceipt{}, err
	}
	return DestinationReceipt{SiteID: siteID, Contract: "vrooli.wordpress.delivery.v1", FormEndpoint: url + "/wp-json/vrooli/v1/form-intake", OrderEndpoint: url + "/wp-json/vrooli/v1/shop-order-sync", Status: "installed"}, nil
}

type DestinationReceipt struct {
	SiteID        string `json:"siteId"`
	Contract      string `json:"contract"`
	FormEndpoint  string `json:"formEndpoint"`
	OrderEndpoint string `json:"orderEndpoint"`
	Status        string `json:"status"`
}

// VerifyDeliveryDestination exercises the installed callbacks against the
// live WordPress PHP runtime and MySQL table. It creates only synthetic
// .invalid records and does not create credentials or send network requests.
func (o *Owner) VerifyDeliveryDestination(ctx context.Context, siteID string) (DestinationVerification, error) {
	if err := validateSiteID(siteID); err != nil {
		return DestinationVerification{}, err
	}
	container := siteContainerPrefix + siteID
	info, exists, err := o.inspectContainer(ctx, container)
	if err != nil {
		return DestinationVerification{}, err
	}
	if !exists || !ownsContainer(info, siteID, "site") || info.State.Status != "running" {
		return DestinationVerification{}, errors.New("WordPress destination verification requires the matching owned site to be running")
	}
	paths, err := o.instancePaths(siteID, false)
	if err != nil {
		return DestinationVerification{}, err
	}
	secrets, err := o.ensureSecrets(paths)
	if err != nil {
		return DestinationVerification{}, err
	}
	formKey, err := randomSecret()
	if err != nil {
		return DestinationVerification{}, err
	}
	orderKey, err := randomSecret()
	if err != nil {
		return DestinationVerification{}, err
	}
	formKey = base64.RawURLEncoding.EncodeToString([]byte(formKey))
	orderKey = base64.RawURLEncoding.EncodeToString([]byte(orderKey))
	code := fmt.Sprintf("$dispatch=static function($method,$payload){$request=new WP_REST_Request('POST','/vrooli/v1/'.$method);$request->set_header('content-type','application/json');$request->set_body(wp_json_encode($payload));$result=$method==='form-intake'?Vrooli_Delivery_Destination::receive_form($request):Vrooli_Delivery_Destination::receive_order($request);if($result instanceof WP_REST_Response){return $result->get_status();}if(is_wp_error($result)){$data=$result->get_error_data();return (int)($data['status']??500);}return 500;};\n$form=['provider'=>'contact-form-7','formId'=>'ground-form','eventId'=>'%s','name'=>'Synthetic Ground','email'=>'ground@example.invalid','message'=>'Synthetic local idempotency check'];\n$order=['siteId'=>'%s','orderId'=>'ground-%s','revision'=>'2026-11-14T12:00:00','status'=>'processing','customerEmail'=>'ground-buyer@example.invalid','items'=>[['productId'=>11,'sku'=>'K3-BOOK-001','quantity'=>1,'total'=>'18.00']]];\n$form1=$dispatch('form-intake',$form);$form2=$dispatch('form-intake',$form);$form['message']='changed payload under same event identity';$formConflict=$dispatch('form-intake',$form);$form['eventId']='bad-email-%s';$form['email']='invalid';$formInvalidEmail=$dispatch('form-intake',$form);$form['eventId']='bad-provider-%s';$form['provider']='elementor-shaped-synthetic';$form['email']='ground@example.invalid';$formInvalidProvider=$dispatch('form-intake',$form);\n$order1=$dispatch('shop-order-sync',$order);$order2=$dispatch('shop-order-sync',$order);$order['revision']='2026-11-14T12:01:00';$order['status']='completed';$orderRevision=$dispatch('shop-order-sync',$order);$order['siteId']='other-site';$orderWrongSite=$dispatch('shop-order-sync',$order);$order['siteId']='%s';$order['orderId']='ground-unknown-%s';$order['revision']='2026-11-14T12:02:00';$order['items'][0]['sku']='K3-UNKNOWN-999';$orderUnknownSku=$dispatch('shop-order-sync',$order);$order['orderId']='ground-missing-revision-%s';unset($order['revision']);$orderMissingRevision=$dispatch('shop-order-sync',$order);\necho wp_json_encode(['form'=>[$form1,$form2,$formConflict],'formRejected'=>[$formInvalidEmail,$formInvalidProvider],'order'=>[$order1,$order2,$orderRevision],'orderRejected'=>[$orderWrongSite,$orderUnknownSku,$orderMissingRevision]]);", formKey, siteID, orderKey, formKey, formKey, siteID, orderKey, orderKey)
	args := o.wpCLIArgs(container, secrets.DatabasePassword, true)
	args = append(args, "wp", "eval", code, "--allow-root")
	output, err := o.runner().Run(ctx, args...)
	if err != nil {
		return DestinationVerification{}, fmt.Errorf("exercise installed WordPress destination callbacks: %w", err)
	}
	var raw struct {
		Form          []int `json:"form"`
		FormRejected  []int `json:"formRejected"`
		Order         []int `json:"order"`
		OrderRejected []int `json:"orderRejected"`
	}
	if err := json.Unmarshal(output, &raw); err != nil || len(raw.Form) != 3 || len(raw.FormRejected) != 2 || len(raw.Order) != 3 || len(raw.OrderRejected) != 3 {
		return DestinationVerification{}, fmt.Errorf("WordPress destination verification returned an invalid receipt: %q", strings.TrimSpace(string(output)))
	}
	receipt := DestinationVerification{SiteID: siteID, Contract: "vrooli.wordpress.delivery.v1", FormStatuses: raw.Form, FormRejectedStatuses: raw.FormRejected, OrderStatuses: raw.Order, OrderRejectedStatuses: raw.OrderRejected, Status: "verified"}
	if raw.Form[0] != 201 || raw.Form[1] != 200 || raw.Form[2] != 409 || raw.FormRejected[0] != 422 || raw.FormRejected[1] != 422 || raw.Order[0] != 201 || raw.Order[1] != 200 || raw.Order[2] != 201 || raw.OrderRejected[0] != 409 || raw.OrderRejected[1] != 422 || raw.OrderRejected[2] != 422 {
		return receipt, errors.New("WordPress destination callback verification failed an idempotency known answer")
	}
	return receipt, nil
}

// VerifyCoreContentGround uses a temporary synthetic WordPress post to check
// custom-field create/read/update behavior in the real local database, then
// deletes that post before returning. It does not claim Pods plugin behavior.
func (o *Owner) VerifyCoreContentGround(ctx context.Context, siteID string) (ContentVerification, error) {
	if err := validateSiteID(siteID); err != nil {
		return ContentVerification{}, err
	}
	container := siteContainerPrefix + siteID
	info, exists, err := o.inspectContainer(ctx, container)
	if err != nil {
		return ContentVerification{}, err
	}
	if !exists || !ownsContainer(info, siteID, "site") || info.State.Status != "running" {
		return ContentVerification{}, errors.New("WordPress content verification requires the matching owned site to be running")
	}
	paths, err := o.instancePaths(siteID, false)
	if err != nil {
		return ContentVerification{}, err
	}
	secrets, err := o.ensureSecrets(paths)
	if err != nil {
		return ContentVerification{}, err
	}
	code := "$slug='k3-ground-'.strtolower(wp_generate_password(12,false,false));$post_id=0;try{$post_id=wp_insert_post(['post_type'=>'post','post_status'=>'publish','post_title'=>'Synthetic content field ground','post_name'=>$slug],true);if(is_wp_error($post_id)){throw new RuntimeException('post_create_failed');}update_post_meta($post_id,'client_type','synthetic');update_post_meta($post_id,'outcome_metric',24);update_post_meta($post_id,'verified_on','2026-11-14');$initial=['client_type'=>get_post_meta($post_id,'client_type',true),'outcome_metric'=>(int)get_post_meta($post_id,'outcome_metric',true),'verified_on'=>get_post_meta($post_id,'verified_on',true)];$first=update_post_meta($post_id,'outcome_metric',25);$second=update_post_meta($post_id,'outcome_metric',25);$final=(int)get_post_meta($post_id,'outcome_metric',true);$missing=metadata_exists('post',$post_id,'unconfigured_private_field');$keys=get_post_custom_keys($post_id)?:[];$groundKeys=['client_type','outcome_metric','verified_on'];$groundCount=count(array_intersect($keys,$groundKeys));echo wp_json_encode(['exists'=>get_post($post_id)!==null,'fieldCount'=>$groundCount,'initial'=>$initial,'created'=>1,'updated'=>is_int($first)||is_string($first)||$first===true?1:0,'duplicate'=>$second===false&&$final===25?1:0,'finalOutcomeMetric'=>$final,'missingFieldExists'=>$missing]);}finally{if($post_id&&!is_wp_error($post_id)){wp_delete_post($post_id,true);}}"
	args := o.wpCLIArgs(container, secrets.DatabasePassword, true)
	args = append(args, "wp", "eval", code, "--allow-root")
	output, err := o.runner().Run(ctx, args...)
	if err != nil {
		return ContentVerification{}, fmt.Errorf("exercise local WordPress custom-field ground: %w", err)
	}
	var raw struct {
		Exists             bool           `json:"exists"`
		FieldCount         int            `json:"fieldCount"`
		Initial            map[string]any `json:"initial"`
		Created            int            `json:"created"`
		Updated            int            `json:"updated"`
		Duplicate          int            `json:"duplicate"`
		FinalOutcomeMetric int            `json:"finalOutcomeMetric"`
		MissingFieldExists bool           `json:"missingFieldExists"`
	}
	if err := json.Unmarshal(output, &raw); err != nil {
		return ContentVerification{}, fmt.Errorf("decode local WordPress content receipt: %w", err)
	}
	receipt := ContentVerification{SiteID: siteID, FieldCount: raw.FieldCount, Created: raw.Created, Updated: raw.Updated, Duplicate: raw.Duplicate, FinalOutcomeMetric: raw.FinalOutcomeMetric, Status: "verified-and-cleaned"}
	metric, metricOK := raw.Initial["outcome_metric"].(float64)
	if !raw.Exists || !metricOK || metric != 24 || raw.Initial["client_type"] != "synthetic" || raw.Initial["verified_on"] != "2026-11-14" || raw.FieldCount != 3 || raw.Created != 1 || raw.Updated != 1 || raw.Duplicate != 1 || raw.FinalOutcomeMetric != 25 || raw.MissingFieldExists {
		return receipt, fmt.Errorf("local WordPress content ground failed a known answer: %s", strings.TrimSpace(string(output)))
	}
	return receipt, nil
}

type ContentVerification struct {
	SiteID             string `json:"siteId"`
	FieldCount         int    `json:"fieldCount"`
	Created            int    `json:"created"`
	Updated            int    `json:"updated"`
	Duplicate          int    `json:"duplicate"`
	FinalOutcomeMetric int    `json:"finalOutcomeMetric"`
	Status             string `json:"status"`
}

type DestinationVerification struct {
	SiteID                string `json:"siteId"`
	Contract              string `json:"contract"`
	FormStatuses          []int  `json:"formStatuses"`
	FormRejectedStatuses  []int  `json:"formRejectedStatuses"`
	OrderStatuses         []int  `json:"orderStatuses"`
	OrderRejectedStatuses []int  `json:"orderRejectedStatuses"`
	Status                string `json:"status"`
}

// ConfigureFormWebhook enables only the versioned F7 n8n webhook hostname;
// callers cannot direct form submissions to an arbitrary or public URL.
func (o *Owner) ConfigureFormWebhook(ctx context.Context, siteID string, enabled bool) (FormWebhookReceipt, error) {
	if err := validateSiteID(siteID); err != nil {
		return FormWebhookReceipt{}, err
	}
	container := siteContainerPrefix + siteID
	info, exists, err := o.inspectContainer(ctx, container)
	if err != nil {
		return FormWebhookReceipt{}, err
	}
	if !exists || !ownsContainer(info, siteID, "site") || info.State.Status != "running" {
		return FormWebhookReceipt{}, errors.New("form webhook configuration requires the matching owned site to be running")
	}
	paths, err := o.instancePaths(siteID, false)
	if err != nil {
		return FormWebhookReceipt{}, err
	}
	secrets, err := o.ensureSecrets(paths)
	if err != nil {
		return FormWebhookReceipt{}, err
	}
	target, status := "", "disabled"
	if enabled {
		target, status = "http://n8n:5678/webhook/wordpress-form-intake", "enabled"
	}
	if err := o.updateSiteOption(ctx, container, secrets.DatabasePassword, "vrooli_form_n8n_webhook", target); err != nil {
		return FormWebhookReceipt{}, fmt.Errorf("configure fixed local WordPress-to-n8n form route: %w", err)
	}
	return FormWebhookReceipt{SiteID: siteID, Enabled: enabled, Target: target, Status: status}, nil
}

func (o *Owner) updateSiteOption(ctx context.Context, container, dbPassword, key, value string) error {
	args := o.wpCLIArgs(container, dbPassword, true)
	args = append(args, "wp", "option", "update", key, value, "--autoload=no", "--allow-root")
	if _, err := o.runner().Run(ctx, args...); err == nil {
		return nil
	}
	readArgs := o.wpCLIArgs(container, dbPassword, true)
	readArgs = append(readArgs, "wp", "option", "get", key, "--allow-root")
	current, readErr := o.runner().Run(ctx, readArgs...)
	if readErr == nil && strings.TrimSpace(string(current)) == value {
		return nil
	}
	return fmt.Errorf("WordPress option %q could not be set or verified", key)
}

type FormWebhookReceipt struct {
	SiteID  string `json:"siteId"`
	Enabled bool   `json:"enabled"`
	Target  string `json:"target,omitempty"`
	Status  string `json:"status"`
}

// ApplyRecipe validates that a recipe's exact third-party package set has
// already been staged by SDA before any site change. Actual activation and
// content provisioning are extended below without a raw WP package manager.
func (o *Owner) ApplyRecipe(ctx context.Context, siteID, recipeID, recipePath string) (RecipeReceipt, error) {
	if err := validateSiteID(siteID); err != nil {
		return RecipeReceipt{}, err
	}
	recipe, err := LoadRecipe(recipePath, recipeID)
	if err != nil {
		return RecipeReceipt{}, err
	}
	for _, slug := range recipe.Plugins {
		return RecipeReceipt{}, fmt.Errorf("%w: %s@%s", ErrRecipeMissing, slug, recipe.PackageVersions[slug])
	}
	if recipe.Kind == "content" && recipe.Settings["fieldProvider"] == "wordpress-core-post-meta" {
		return o.applyCoreContentRecipe(ctx, siteID, recipe)
	}
	return RecipeReceipt{SiteID: siteID, RecipeID: recipe.ID, Version: recipe.Version, Status: "ready"}, nil
}

func (o *Owner) applyCoreContentRecipe(ctx context.Context, siteID string, recipe Recipe) (RecipeReceipt, error) {
	container := siteContainerPrefix + siteID
	info, exists, err := o.inspectContainer(ctx, container)
	if err != nil {
		return RecipeReceipt{}, err
	}
	if !exists || !ownsContainer(info, siteID, "site") || info.State.Status != "running" {
		return RecipeReceipt{}, errors.New("core content recipe requires the matching owned site to be running")
	}
	paths, err := o.instancePaths(siteID, false)
	if err != nil {
		return RecipeReceipt{}, err
	}
	secrets, err := o.ensureSecrets(paths)
	if err != nil {
		return RecipeReceipt{}, err
	}
	posts, ok := recipe.Content["posts"].([]any)
	if !ok || len(posts) == 0 {
		return RecipeReceipt{}, errors.New("core content recipe must include posts")
	}
	for index, raw := range posts {
		post, ok := raw.(map[string]any)
		if !ok {
			return RecipeReceipt{}, fmt.Errorf("core content recipe post %d is not an object", index)
		}
		if err := validateCoreContentPost(post); err != nil {
			return RecipeReceipt{}, fmt.Errorf("core content recipe post %d: %w", index, err)
		}
	}
	payload, err := json.Marshal(posts)
	if err != nil {
		return RecipeReceipt{}, err
	}
	encoded := base64.StdEncoding.EncodeToString(payload)
	code := fmt.Sprintf("$rows=json_decode(base64_decode('%s'),true,512,JSON_THROW_ON_ERROR);$recipe='%s';$created=0;$updated=0;$verified=[];foreach($rows as $row){$content=wp_kses_post($row['content']??'');$excerpt=sanitize_text_field($row['excerpt']??'');$post=get_page_by_path($row['slug'],OBJECT,'post');if($post){$id=(int)$post->ID;if(get_post_meta($id,'_vrooli_recipe_id',true)!==$recipe){throw new RuntimeException('existing_slug_not_owned');}$write=wp_update_post(['ID'=>$id,'post_title'=>$row['title'],'post_content'=>$content,'post_excerpt'=>$excerpt,'post_status'=>$row['status']],true);if(is_wp_error($write)||(int)$write!==$id){throw new RuntimeException('post_update_failed');}$updated++;}else{$id=wp_insert_post(['post_type'=>'post','post_status'=>$row['status'],'post_title'=>$row['title'],'post_content'=>$content,'post_excerpt'=>$excerpt,'post_name'=>$row['slug']],true);if(is_wp_error($id)){throw new RuntimeException('post_create_failed');}$created++;}foreach(['client_type','outcome_metric','verified_on'] as $field){update_post_meta($id,$field,$row[$field]);}$termIds=[];foreach(($row['categories']??[]) as $name){$term=term_exists($name,'category');if(!$term){$term=wp_insert_term($name,'category');}if(is_wp_error($term)){throw new RuntimeException('category_term_failed');}$termIds[]=(int)(is_array($term)?$term['term_id']:$term);} $set=wp_set_post_terms($id,$termIds,'category',false);if(is_wp_error($set)){throw new RuntimeException('category_assign_failed');}$tagIds=[];foreach(($row['tags']??[]) as $name){$term=term_exists($name,'post_tag');if(!$term){$term=wp_insert_term($name,'post_tag');}if(is_wp_error($term)){throw new RuntimeException('tag_term_failed');}$tagIds[]=(int)(is_array($term)?$term['term_id']:$term);} $set=wp_set_post_terms($id,$tagIds,'post_tag',false);if(is_wp_error($set)){throw new RuntimeException('tag_assign_failed');}$actualCats=wp_get_post_terms($id,'category',['fields'=>'names']);$actualTags=wp_get_post_terms($id,'post_tag',['fields'=>'names']);sort($actualCats);sort($actualTags);$wantCats=$row['categories']??[];$wantTags=$row['tags']??[];sort($wantCats);sort($wantTags);$actual=['slug'=>get_post_field('post_name',$id),'title'=>get_post_field('post_title',$id),'content'=>get_post_field('post_content',$id),'excerpt'=>get_post_field('post_excerpt',$id),'status'=>get_post_field('post_status',$id),'client_type'=>get_post_meta($id,'client_type',true),'outcome_metric'=>(int)get_post_meta($id,'outcome_metric',true),'verified_on'=>get_post_meta($id,'verified_on',true),'categories'=>$actualCats,'tags'=>$actualTags];if($actual['slug']!==$row['slug']||$actual['title']!==$row['title']||$actual['content']!==$content||$actual['excerpt']!==$excerpt||$actual['status']!==$row['status']||$actual['client_type']!==$row['client_type']||$actual['outcome_metric']!==(int)$row['outcome_metric']||$actual['verified_on']!==$row['verified_on']||$actualCats!==$wantCats||$actualTags!==$wantTags){throw new RuntimeException('post_field_verify_failed');}update_post_meta($id,'_vrooli_recipe_id',$recipe);$verified[]=$actual;}echo wp_json_encode(['created'=>$created,'updated'=>$updated,'posts'=>$verified]);", encoded, recipe.ID)
	args := o.wpCLIArgs(container, secrets.DatabasePassword, true)
	args = append(args, "wp", "eval", code, "--allow-root")
	output, err := o.runner().Run(ctx, args...)
	if err != nil {
		return RecipeReceipt{}, fmt.Errorf("apply WordPress core content recipe: %w", err)
	}
	var result struct {
		Created int              `json:"created"`
		Updated int              `json:"updated"`
		Posts   []map[string]any `json:"posts"`
	}
	if err := json.Unmarshal(output, &result); err != nil || len(result.Posts) != len(posts) || result.Created+result.Updated != len(posts) {
		return RecipeReceipt{}, fmt.Errorf("WordPress core content recipe returned an invalid receipt: %s", strings.TrimSpace(string(output)))
	}
	return RecipeReceipt{SiteID: siteID, RecipeID: recipe.ID, Version: recipe.Version, Status: "applied-and-verified", PostCount: len(result.Posts), Created: result.Created, Updated: result.Updated}, nil
}

var contentSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func validateCoreContentPost(post map[string]any) error {
	slug, _ := post["slug"].(string)
	title, _ := post["title"].(string)
	content, hasContent := post["content"].(string)
	excerpt, hasExcerpt := post["excerpt"].(string)
	status, _ := post["status"].(string)
	clientType, _ := post["client_type"].(string)
	date, _ := post["verified_on"].(string)
	categories, categoriesOK := contentTermNames(post["categories"])
	tags, tagsOK := contentTermNames(post["tags"])
	metric, ok := post["outcome_metric"].(float64)
	if !contentSlugPattern.MatchString(slug) || strings.TrimSpace(title) == "" || len(title) > 200 || (status != "publish" && status != "draft") || clientType != "synthetic" || !ok || metric < 0 || metric > 1_000_000_000 {
		return errors.New("expected safe slug/title/status and synthetic bounded custom fields")
	}
	if (post["content"] != nil && !hasContent) || (post["excerpt"] != nil && !hasExcerpt) || len(content) > 20_000 || len(excerpt) > 500 || strings.ContainsRune(content, '\x00') || strings.ContainsRune(excerpt, '\x00') {
		return errors.New("content and excerpt must be text within their bounded lengths")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return errors.New("verified_on must be a real YYYY-MM-DD date")
	}
	if !categoriesOK || !tagsOK || len(categories) > 10 || len(tags) > 10 {
		return errors.New("categories and tags must be lists of at most ten safe names")
	}
	return nil
}

var contentTermNamePattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9 &-]{0,62}[A-Za-z0-9])?$`)

func contentTermNames(value any) ([]string, bool) {
	if value == nil {
		return nil, true
	}
	var names []string
	switch items := value.(type) {
	case []string:
		names = items
	case []any:
		for _, item := range items {
			name, ok := item.(string)
			if !ok {
				return nil, false
			}
			names = append(names, name)
		}
	default:
		return nil, false
	}
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		key := strings.ToLower(strings.TrimSpace(name))
		if !contentTermNamePattern.MatchString(name) || key == "" || seen[key] {
			return nil, false
		}
		seen[key] = true
	}
	return append([]string(nil), names...), true
}

type RecipeReceipt struct {
	SiteID    string `json:"siteId"`
	RecipeID  string `json:"recipeId"`
	Version   int    `json:"version"`
	Status    string `json:"status"`
	PostCount int    `json:"postCount,omitempty"`
	Created   int    `json:"created,omitempty"`
	Updated   int    `json:"updated,omitempty"`
}

func (o *Owner) waitDatabase(ctx context.Context, name, rootPassword string) error {
	interval := o.PollEvery
	if interval <= 0 {
		interval = time.Second
	}
	deadline := time.NewTimer(readinessTimeout)
	defer deadline.Stop()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if _, err := o.runner().Run(ctx, "exec", name, "mariadb-admin", "ping", "--silent", "-uroot", "--password="+rootPassword); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("WordPress database did not become ready within %s", readinessTimeout)
		case <-ticker.C:
		}
	}
}

func (o *Owner) siteURL(_ context.Context, name string, info containerInfo) (string, error) {
	siteID := strings.TrimPrefix(name, siteContainerPrefix)
	network, exists := info.NetworkSettings.Networks[wordpressNetworkPrefix+siteID]
	if !exists {
		return "", errors.New("WordPress site is missing its private owner network")
	}
	ip := net.ParseIP(strings.TrimSpace(network.IPAddress))
	if ip == nil || !ip.IsPrivate() || ip.IsLoopback() {
		return "", fmt.Errorf("WordPress private network returned an invalid site address %q", network.IPAddress)
	}
	return "http://" + ip.String(), nil
}

func (o *Owner) waitHTTP(ctx context.Context, target string) error {
	interval := o.PollEvery
	if interval <= 0 {
		interval = time.Second
	}
	deadline := time.NewTimer(readinessTimeout)
	defer deadline.Stop()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := o.checkHTTP(ctx, target); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("WordPress did not serve wp-login.php within %s", readinessTimeout)
		case <-ticker.C:
		}
	}
}

func (o *Owner) checkHTTP(ctx context.Context, target string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	response, err := o.client().Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
	if response.StatusCode < 200 || response.StatusCode >= 400 {
		return fmt.Errorf("WordPress readiness returned HTTP %d", response.StatusCode)
	}
	return nil
}

type containerInfo struct {
	Config struct {
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
	State struct {
		Status string `json:"Status"`
	} `json:"State"`
	NetworkSettings struct {
		Networks map[string]struct {
			IPAddress string `json:"IPAddress"`
		} `json:"Networks"`
	} `json:"NetworkSettings"`
}

func (o *Owner) inspectContainer(ctx context.Context, name string) (containerInfo, bool, error) {
	output, err := o.runner().Run(ctx, "container", "inspect", name)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such container") || strings.Contains(strings.ToLower(err.Error()), "no such object") {
			return containerInfo{}, false, nil
		}
		return containerInfo{}, false, err
	}
	var values []containerInfo
	if json.Unmarshal(output, &values) != nil || len(values) != 1 {
		return containerInfo{}, false, errors.New("Docker returned invalid WordPress container inspection data")
	}
	return values[0], true, nil
}

func ownsContainer(info containerInfo, siteID, kind string) bool {
	return info.Config.Labels[labelResource] == resourceLabelValue && info.Config.Labels[labelObjectKind] == kind && info.Config.Labels[labelSiteID] == siteID
}

type sitePaths struct{ Root, Secrets string }

func (o *Owner) instancePaths(siteID string, create bool) (sitePaths, error) {
	if err := validateSiteID(siteID); err != nil {
		return sitePaths{}, err
	}
	root := filepath.Join(o.DataRoot, "instances", siteID)
	if err := ensureSafeDir(o.DataRoot, filepath.Dir(root)); err != nil {
		return sitePaths{}, err
	}
	if create {
		if err := ensureSafeDir(o.DataRoot, root); err != nil {
			return sitePaths{}, err
		}
	}
	if err := verifyPathInside(o.DataRoot, root); err != nil {
		return sitePaths{}, err
	}
	return sitePaths{Root: root, Secrets: filepath.Join(root, "secrets.json")}, nil
}

type instanceSecrets struct {
	RootPassword     string `json:"rootPassword"`
	DatabasePassword string `json:"databasePassword"`
	AdminPassword    string `json:"adminPassword"`
}

func (o *Owner) ensureSecrets(paths sitePaths) (instanceSecrets, error) {
	info, err := os.Lstat(paths.Secrets)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
			return instanceSecrets{}, errors.New("WordPress local credential file has unsafe ownership or permissions")
		}
		data, err := os.ReadFile(paths.Secrets)
		if err != nil {
			return instanceSecrets{}, err
		}
		var secrets instanceSecrets
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&secrets); err != nil || secrets.RootPassword == "" || secrets.DatabasePassword == "" || secrets.AdminPassword == "" {
			return instanceSecrets{}, errors.New("WordPress local credential file is invalid")
		}
		return secrets, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return instanceSecrets{}, err
	}
	secrets := instanceSecrets{}
	var genErr error
	secrets.RootPassword, genErr = randomSecret()
	if genErr != nil {
		return instanceSecrets{}, genErr
	}
	secrets.DatabasePassword, genErr = randomSecret()
	if genErr != nil {
		return instanceSecrets{}, genErr
	}
	secrets.AdminPassword, genErr = randomSecret()
	if genErr != nil {
		return instanceSecrets{}, genErr
	}
	data, _ := json.Marshal(secrets)
	file, err := os.OpenFile(paths.Secrets, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return instanceSecrets{}, fmt.Errorf("create private WordPress credential file: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return instanceSecrets{}, err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return instanceSecrets{}, err
	}
	if err := file.Close(); err != nil {
		return instanceSecrets{}, err
	}
	return secrets, nil
}

func randomSecret() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func objectNames(siteID string) (networkName, databaseName, siteName string) {
	return wordpressNetworkPrefix + siteID, databaseContainerPrefix + siteID, siteContainerPrefix + siteID
}

func validateSiteID(siteID string) error {
	if !siteIDPattern.MatchString(siteID) {
		return ErrInvalidSiteID
	}
	return nil
}

func ensureSafeDir(root, target string) error {
	if err := verifyPathInside(root, target); err != nil {
		return err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	current := string(filepath.Separator)
	for _, component := range strings.Split(strings.TrimPrefix(rootAbs, string(filepath.Separator)), string(filepath.Separator)) {
		if component == "." || component == "" {
			continue
		}
		current = filepath.Join(current, component)
		if err := os.Mkdir(current, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("WordPress resource path is not a safe directory: %s", current)
		}
	}
	relative, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return err
	}
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "." || component == "" {
			continue
		}
		current = filepath.Join(current, component)
		if err := os.Mkdir(current, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("WordPress resource path is not a safe directory: %s", current)
		}
	}
	return nil
}

func verifyPathInside(root, target string) error {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("WordPress resource path escapes its owner data root")
	}
	return nil
}

func removeOwnedDir(root, target string) error {
	if err := verifyPathInside(root, target); err != nil {
		return err
	}
	info, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := rejectSymlinkAncestors(root); err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("refusing to reset WordPress site data through a non-directory or symlink")
	}
	return os.RemoveAll(target)
}

func rejectSymlinkAncestors(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	current := string(filepath.Separator)
	for _, component := range strings.Split(strings.TrimPrefix(absolute, string(filepath.Separator)), string(filepath.Separator)) {
		if component == "" {
			continue
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing WordPress operation through symlink ancestor %s", current)
		}
	}
	return nil
}

type boundedWriter struct {
	Writer    io.Writer
	remaining int64
	exceeded  bool
}

func (w *boundedWriter) Write(data []byte) (int, error) {
	if int64(len(data)) > w.remaining {
		w.exceeded = true
		allowed := w.remaining
		if allowed > 0 {
			n, err := w.Writer.Write(data[:allowed])
			w.remaining -= int64(n)
			if err != nil {
				return n, err
			}
		}
		return len(data), nil
	}
	n, err := w.Writer.Write(data)
	w.remaining -= int64(n)
	return n, err
}

func writeTarBytes(writer *tar.Writer, name string, data []byte) error {
	header := &tar.Header{Name: name, Mode: 0o600, Size: int64(len(data)), Typeflag: tar.TypeReg, ModTime: time.Unix(0, 0).UTC()}
	if err := writer.WriteHeader(header); err != nil {
		return err
	}
	_, err := writer.Write(data)
	return err
}

func writeTarFile(writer *tar.Writer, name, source string) error {
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("snapshot source must be a regular file")
	}
	if info.Size() > SnapshotMaxBytes {
		return fmt.Errorf("snapshot file exceeds %d-byte limit", SnapshotMaxBytes)
	}
	header := &tar.Header{Name: name, Mode: 0o600, Size: info.Size(), Typeflag: tar.TypeReg, ModTime: time.Unix(0, 0).UTC()}
	if err := writer.WriteHeader(header); err != nil {
		return err
	}
	_, err = io.Copy(writer, io.LimitReader(file, SnapshotMaxBytes+1))
	return err
}

func writeTarTree(writer *tar.Writer, sourceRoot, archiveRoot string) error {
	info, err := os.Lstat(sourceRoot)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("WordPress content snapshot root is not a safe directory")
	}
	return filepath.WalkDir(sourceRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == sourceRoot {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to snapshot WordPress symlink %s", path)
		}
		rel, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(filepath.Join(archiveRoot, rel))
		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return writer.WriteHeader(&tar.Header{Name: name + "/", Mode: 0o700, Typeflag: tar.TypeDir, ModTime: time.Unix(0, 0).UTC()})
		}
		if !fileInfo.Mode().IsRegular() {
			return fmt.Errorf("refusing to snapshot non-regular WordPress content %s", path)
		}
		if fileInfo.Size() > SnapshotMaxBytes {
			return fmt.Errorf("WordPress content file exceeds snapshot limit: %s", path)
		}
		return addTarFile(writer, name, path, fileInfo.Size())
	})
}

func addTarFile(writer *tar.Writer, name, path string, size int64) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	if err := writer.WriteHeader(&tar.Header{Name: name, Mode: 0o600, Size: size, Typeflag: tar.TypeReg, ModTime: time.Unix(0, 0).UTC()}); err != nil {
		_ = file.Close()
		return err
	}
	_, copyErr := io.Copy(writer, io.LimitReader(file, SnapshotMaxBytes+1))
	closeErr := file.Close()
	return errors.Join(copyErr, closeErr)
}

// Fingerprint is a stable hash for a locally recorded plugin archive.
func Fingerprint(content []byte) string {
	digest := sha256.Sum256(content)
	return hex.EncodeToString(digest[:])
}
