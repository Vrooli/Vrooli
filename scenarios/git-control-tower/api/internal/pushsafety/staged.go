package pushsafety

import (
	"bytes"
	"context"
	"strconv"
	"strings"
)

// InspectStaged reads the index, never working-file sizes. Its advisory result
// is excluded from the immutable outgoing-history recovery fingerprint.
func InspectStaged(ctx context.Context, c Commands, repo string, r *Report) {
	r.StagedReason = "Staged file sizes could not be checked. Local commits remain available."
	args := []string{"diff", "--cached", "--raw", "--no-abbrev", "--no-renames", "--no-ext-diff", "-z"}
	raw, err := c.Run(ctx, repo, nil, args...)
	if err != nil {
		return
	}
	records := strings.Split(string(raw), "\x00")
	var files []File
	var candidates []File
	var input strings.Builder
	for i := 0; i < len(records)-1; i += 2 {
		if i+1 >= len(records)-1 {
			return
		}
		fields := strings.Fields(records[i])
		if len(fields) != 5 || !strings.HasPrefix(fields[0], ":") || !validOID(fields[3]) {
			return
		}
		if fields[4] == "U" {
			r.StagedReason = "Resolve merge conflicts before checking staged content."
			return
		}
		if fields[4] == "D" || fields[1] == "160000" {
			continue
		}
		candidates = append(candidates, File{OID: fields[3], Paths: []string{records[i+1]}})
		input.WriteString(fields[3] + "\n")
	}
	if len(candidates) > 0 {
		metadata, e := c.Run(ctx, repo, []byte(input.String()), "cat-file", "--batch-check=%(objectname) %(objecttype) %(objectsize)")
		if e != nil {
			return
		}
		lines := strings.Split(strings.TrimSpace(string(metadata)), "\n")
		if len(lines) != len(candidates) {
			return
		}
		for i, line := range lines {
			fields := strings.Fields(line)
			if len(fields) != 3 || fields[0] != candidates[i].OID || fields[1] != "blob" {
				return
			}
			size, e := strconv.ParseInt(fields[2], 10, 64)
			if e != nil || size < 0 {
				return
			}
			if size > WarningBytes {
				f := candidates[i]
				f.Bytes = size
				f.Blocked = r.Limit > 0 && size > r.Limit
				files = append(files, f)
			}
		}
	}
	after, err := c.Run(ctx, repo, nil, args...)
	if err != nil || !bytes.Equal(raw, after) {
		r.StagedReason = "Staged content changed during inspection. Check again before committing."
		return
	}
	r.StagedFiles = files
	r.StagedComplete = true
	r.StagedReason = "Checked staged Git objects against the destination file-size policy. Local commits are allowed; push checks run again before transfer."
	if r.Limit == 0 {
		r.StagedReason = "Staged sizes checked; the destination file-size policy is unknown."
	}
}
