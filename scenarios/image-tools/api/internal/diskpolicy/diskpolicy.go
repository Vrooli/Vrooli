// Package diskpolicy is the one place Image Tools decides how much free disk a
// write must leave behind, and how much of the disk a job may ask the shared
// host ledger to hold.
//
// The shared capacity ledger enforces a host-wide minimum floor on every
// ordinary disk job (120 GiB, which it refuses to lower outside tests). Image
// Tools therefore asks for exactly the host minimum by default and never a
// private, larger number: a job is refused only when the host's own policy
// refuses it. An operator who wants more headroom raises it with
// IMAGE_TOOLS_DISK_FLOOR. Writes that are genuinely small (at most the ledger's
// small-write bound) use the ledger's small-write class and its far lower
// floor, so a 5 MB import is never refused for a promise made to a 40 GB
// download.
package diskpolicy

import (
	"fmt"
	"strings"

	"github.com/vrooli/api-core/retention"
	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
)

const (
	// EnvFloor raises the floor above the host minimum (for example "200GiB").
	// A value below the host minimum is ignored and reported, never applied.
	EnvFloor = "IMAGE_TOOLS_DISK_FLOOR"

	// HostMinimumFloorBytes is the ordinary-job floor the shared capacity ledger
	// enforces for every owner (internal/capacity, "cannot be lowered below 120
	// GiB outside tests"). The ledger raises any lower request to this value, so
	// requesting less changes nothing; it is mirrored here so the request, the
	// storage planner and the diagnostics agree with the ledger.
	HostMinimumFloorBytes int64 = 120 << 30

	// SmallWriteMaxBytes is the most a small-write disk job may declare in total.
	SmallWriteMaxBytes = sharedcapacity.DiskJobMetadataMaxPeakBytes
)

// Floor is the resolved post-job free-space requirement and why.
type Floor struct {
	Bytes   int64    `json:"bytes"`
	Reasons []string `json:"reasons"`
}

// Resolve reads the operator override and returns the floor.
func Resolve(getenv func(string) string) Floor {
	f := Floor{Bytes: HostMinimumFloorBytes, Reasons: []string{
		fmt.Sprintf("host minimum %s: the shared capacity ledger refuses to hold an ordinary disk job to less, so Image Tools asks for exactly that and no private number", human(HostMinimumFloorBytes)),
	}}
	raw := ""
	if getenv != nil {
		raw = strings.TrimSpace(getenv(EnvFloor))
	}
	if raw == "" {
		return f
	}
	v, err := retention.ParseBytes(raw)
	switch {
	case err != nil || v < 0:
		f.Reasons = append(f.Reasons, fmt.Sprintf("%s=%q is not a size such as 200GiB: ignored", EnvFloor, raw))
	case v <= HostMinimumFloorBytes:
		f.Reasons = append(f.Reasons, fmt.Sprintf("%s=%s is not above the host minimum: ignored (the ledger would raise it anyway)", EnvFloor, human(v)))
	default:
		f.Bytes = v
		f.Reasons = append(f.Reasons, fmt.Sprintf("operator raised the floor to %s with %s", human(v), EnvFloor))
	}
	return f
}

// Request fills a ledger request for a write of the declared paths: small
// writes use the ledger's small-write class, everything else is held to floor.
func (f Floor) Request(owner, job string, paths []sharedcapacity.DiskJobPath) sharedcapacity.DiskJobRequest {
	req := sharedcapacity.DiskJobRequest{OwnerID: owner, JobID: job, Paths: paths, FloorBytes: f.Bytes}
	if SmallWrite(paths) {
		req.Class = sharedcapacity.DiskJobClassMetadata
		req.FloorBytes = 0
	}
	return req
}

// SmallWrite reports whether the declared peaks fit the ledger's small-write
// class.
func SmallWrite(paths []sharedcapacity.DiskJobPath) bool {
	var total int64
	for _, p := range paths {
		if p.PeakBytes <= 0 || total > SmallWriteMaxBytes-p.PeakBytes {
			return false
		}
		total += p.PeakBytes
	}
	return total > 0
}

// Headroom is the free space left above the floor (negative when below it).
func (f Floor) Headroom(freeBytes int64) int64 { return freeBytes - f.Bytes }

func human(n int64) string { return fmt.Sprintf("%.1f GiB", float64(n)/float64(1<<30)) }

// Status is the diagnostic view of the policy on one filesystem.
type Status struct {
	Path                string   `json:"path"`
	FloorBytes          int64    `json:"floor_bytes"`
	FloorReasons        []string `json:"floor_reasons"`
	DiskTotalBytes      int64    `json:"disk_total_bytes"`
	FreeBytes           int64    `json:"free_bytes"`
	HeadroomBytes       int64    `json:"headroom_bytes"`
	FloorPercentOfDisk  float64  `json:"floor_percent_of_disk"`
	SmallWriteMaxBytes  int64    `json:"small_write_max_bytes"`
	SmallWriteFloor     string   `json:"small_write_floor"`
	InstallPeakFormula  string   `json:"install_peak_formula"`
	JobAdmissionSummary string   `json:"job_admission_summary"`
}

// Describe reports the floor and the room above it at path.
func Describe(path string, getenv func(string) string) (Status, error) {
	f := Resolve(getenv)
	total, free, err := observe(path)
	if err != nil {
		return Status{}, err
	}
	st := Status{
		Path: path, FloorBytes: f.Bytes, FloorReasons: f.Reasons,
		DiskTotalBytes: total, FreeBytes: free, HeadroomBytes: f.Headroom(free),
		SmallWriteMaxBytes:  SmallWriteMaxBytes,
		SmallWriteFloor:     "the shared ledger's small-write floor (8 GiB) for writes of at most 64 MiB in total, such as a link import",
		InstallPeakFormula:  "model install = artifact + max(10%, 64 MiB) at the model root + tool scratch max(128 MiB, min(artifact/16, 2 GiB)) + Hub cache allowance min(artifact, 512 MiB); growth extends the lease to observed bytes + 25% (at least 1 GiB) before each further write",
		JobAdmissionSummary: "a job is admitted when free space >= (every other active reservation) + this job's peak + the floor; when only other owners' in-flight reservations cause a denial the job waits and is admitted again instead of failing",
	}
	if total > 0 {
		st.FloorPercentOfDisk = float64(f.Bytes) / float64(total) * 100
	}
	return st, nil
}
