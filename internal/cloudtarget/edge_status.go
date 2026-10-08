package cloudtarget

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// EdgeHostStatus is one routed host as the target observes it.
type EdgeHostStatus struct {
	Host         string `json:"host"`
	UpstreamPort int    `json:"upstream_port"`
	// Certificate fields come from Caddy's storage; absent when no
	// certificate has been obtained for the host yet.
	CertificatePresent bool   `json:"certificate_present"`
	Issuer             string `json:"issuer,omitempty"`
	NotAfter           string `json:"not_after,omitempty"`
	DaysLeft           int    `json:"days_left,omitempty"`
	CertificateError   string `json:"certificate_error,omitempty"`
}

// EdgeStatus is what `edge route status` prints. It carries no secrets: no
// private keys, no DNS tokens, only public certificate metadata.
type EdgeStatus struct {
	DeploymentID       string           `json:"deployment_id"`
	SnippetPath        string           `json:"snippet_path"`
	SnippetPresent     bool             `json:"snippet_present"`
	SnippetDigest      string           `json:"snippet_digest,omitempty"`
	ImportLinePresent  bool             `json:"import_line_present"`
	Hosts              []EdgeHostStatus `json:"hosts"`
	PreviousRetained   bool             `json:"previous_retained"`
	UnrelatedHash      string           `json:"unrelated_config_hash"`
	OtherSnippetCount  int              `json:"other_snippet_count"`
	ObservedAt         string           `json:"observed_at"`
	ActiveSpecDigest   string           `json:"active_spec_digest,omitempty"`
	ACMEEnvironment    string           `json:"acme_environment,omitempty"`
	CertificateStorage string           `json:"certificate_storage"`
}

var upstreamPattern = regexp.MustCompile(`reverse_proxy\s+127\.0\.0\.1:(\d+)`)

// EdgeRouteStatus reads the deployment snippet, the retained state and
// Caddy's certificate storage. It never mutates anything.
func (s *Store) EdgeRouteStatus(deploymentID string, paths CaddyPaths) (EdgeStatus, error) {
	dir, err := s.DeploymentDir(deploymentID)
	if err != nil {
		return EdgeStatus{}, err
	}
	paths = paths.withDefaults()
	snippetPath := paths.SnippetPath(deploymentID)
	status := EdgeStatus{DeploymentID: deploymentID, SnippetPath: snippetPath, Hosts: []EdgeHostStatus{}, ObservedAt: s.now().Format(time.RFC3339), CertificateStorage: paths.DataDir}
	snippet, present, err := readOptional(snippetPath)
	if err != nil {
		return EdgeStatus{}, fail(CodeStoreIO, "read snippet: %v", err)
	}
	status.SnippetPresent = present
	if present {
		status.SnippetDigest = sha256Hex(snippet)
		for _, block := range parseSiteBlocks(snippet) {
			host := EdgeHostStatus{Host: block.host, UpstreamPort: block.port}
			fillCertificate(&host, paths.DataDir, block.host, s.now())
			status.Hosts = append(status.Hosts, host)
		}
	}
	main, _, err := readOptional(paths.MainConfig)
	if err != nil {
		return EdgeStatus{}, fail(CodeStoreIO, "read %s: %v", paths.MainConfig, err)
	}
	for _, line := range strings.Split(main, "\n") {
		if strings.TrimSpace(line) == caddyImportLine {
			status.ImportLinePresent = true
		}
	}
	hash, err := unrelatedConfigHash(paths, snippetPath)
	if err != nil {
		return EdgeStatus{}, err
	}
	status.UnrelatedHash = hash
	if entries, err := os.ReadDir(paths.ConfDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), snippetSuffix) && filepath.Join(paths.ConfDir, entry.Name()) != snippetPath {
				status.OtherSnippetCount++
			}
		}
	}
	if _, retained, _ := readOptional(filepath.Join(dir, edgeDirName, edgePreviousFile)); retained {
		status.PreviousRetained = true
	}
	var spec EdgeRouteSpec
	if err := readJSON(filepath.Join(dir, edgeDirName, edgeSpecFile), &spec); err == nil {
		if digest, err := spec.SpecDigest(); err == nil {
			status.ActiveSpecDigest = digest
		}
		status.ACMEEnvironment = spec.ACMEEnvironment
	}
	return status, nil
}

type siteBlock struct {
	host string
	port int
}

// parseSiteBlocks reads the site headers and loopback upstreams out of a
// snippet this owner rendered. It is intentionally narrow: anything else in
// the file was refused by checkSnippetScope before it was written.
func parseSiteBlocks(snippet string) []siteBlock {
	var blocks []siteBlock
	var open []string
	depth := 0
	for _, line := range strings.Split(snippet, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if depth == 0 {
			if match := siteHeaderPattern.FindStringSubmatch(line); match != nil {
				open = open[:0]
				for _, host := range strings.Split(match[1], ",") {
					open = append(open, strings.ToLower(strings.TrimSpace(host)))
				}
				depth++
			}
			continue
		}
		if match := upstreamPattern.FindStringSubmatch(line); match != nil {
			port, _ := strconv.Atoi(match[1])
			for _, host := range open {
				blocks = append(blocks, siteBlock{host: host, port: port})
			}
		}
		if strings.HasSuffix(trimmed, "{") {
			depth++
		}
		if trimmed == "}" {
			depth--
		}
	}
	sort.Slice(blocks, func(i, j int) bool { return blocks[i].host < blocks[j].host })
	return blocks
}

// fillCertificate reads the leaf certificate Caddy stored for host beneath
// <data>/certificates/<issuer-dir>/<host>/<host>.crt. Only public metadata
// leaves this function.
func fillCertificate(host *EdgeHostStatus, dataDir, name string, now time.Time) {
	matches, _ := filepath.Glob(filepath.Join(dataDir, "certificates", "*", name, name+".crt"))
	if len(matches) == 0 {
		return
	}
	sort.Strings(matches)
	var newest *x509.Certificate
	for _, path := range matches {
		raw, err := os.ReadFile(path) //nolint:gosec // fixed layout beneath the proxy data dir
		if err != nil {
			host.CertificateError = err.Error()
			continue
		}
		block, _ := pem.Decode(raw)
		if block == nil {
			host.CertificateError = "certificate file is not PEM"
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			host.CertificateError = err.Error()
			continue
		}
		if newest == nil || cert.NotAfter.After(newest.NotAfter) {
			newest = cert
		}
	}
	if newest == nil {
		return
	}
	host.CertificatePresent = true
	host.CertificateError = ""
	host.Issuer = newest.Issuer.CommonName
	if host.Issuer == "" {
		host.Issuer = newest.Issuer.String()
	}
	host.NotAfter = newest.NotAfter.UTC().Format(time.RFC3339)
	host.DaysLeft = int(newest.NotAfter.Sub(now).Hours() / 24)
}
