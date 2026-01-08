package vps

import (
	"strconv"
	"strings"

	"scenario-to-cloud/domain"
)

// ParseLsOutput parses the output of `ls -la` into FileEntry structs.
func ParseLsOutput(output string) []domain.FileEntry {
	var entries []domain.FileEntry
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip total line
		if strings.HasPrefix(line, "total ") {
			continue
		}

		// Skip . and ..
		if strings.HasSuffix(line, " .") || strings.HasSuffix(line, " ..") {
			continue
		}

		// Parse ls -la output
		// Format: -rw-r--r-- 1 user group size month day time/year name
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		permissions := fields[0]
		sizeStr := fields[4]
		month := fields[5]
		day := fields[6]
		timeOrYear := fields[7]
		name := strings.Join(fields[8:], " ")

		// Determine type from permissions
		fileType := "file"
		if permissions[0] == 'd' {
			fileType = "directory"
		} else if permissions[0] == 'l' {
			fileType = "symlink"
		}

		size, _ := strconv.ParseInt(sizeStr, 10, 64)

		entries = append(entries, domain.FileEntry{
			Name:        name,
			Type:        fileType,
			SizeBytes:   size,
			Modified:    month + " " + day + " " + timeOrYear,
			Permissions: permissions,
		})
	}

	return entries
}

// IsPathWithinWorkdir checks if a path is within the workdir to prevent directory traversal.
func IsPathWithinWorkdir(path, workdir string) bool {
	// Normalize paths
	path = normalizePath(path)
	workdir = normalizePath(workdir)

	// Check if path starts with workdir
	return len(path) >= len(workdir) && path[:len(workdir)] == workdir
}

// normalizePath normalizes a path by removing trailing slashes.
func normalizePath(p string) string {
	// Remove trailing slash
	for len(p) > 1 && p[len(p)-1] == '/' {
		p = p[:len(p)-1]
	}
	return p
}
