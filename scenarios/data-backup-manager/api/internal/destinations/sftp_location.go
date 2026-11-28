package destinations

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type sftpLocation struct {
	Host       string
	Port       int
	Username   string
	RemotePath string
	KnownHosts string
}

// parseSFTPLocation accepts only non-secret connection metadata. Private-key
// content is provisioned separately under the destination's Kopia identity.
func parseSFTPLocation(raw string) (sftpLocation, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return sftpLocation{}, fmt.Errorf("parse SFTP location: %w", err)
	}
	if u.Scheme != "sftp" || u.Hostname() == "" || u.User == nil {
		return sftpLocation{}, fmt.Errorf("SFTP location must be sftp://<username>@<host>[:port]/<absolute-path>?known_hosts=<local-file>")
	}
	if _, hasPassword := u.User.Password(); hasPassword {
		return sftpLocation{}, fmt.Errorf("SFTP password is not supported; provision an SSH private key through the credential authority")
	}
	username := strings.TrimSpace(u.User.Username())
	if username == "" || u.Fragment != "" {
		return sftpLocation{}, fmt.Errorf("SFTP username is required and fragments are not allowed")
	}
	knownHosts := strings.TrimSpace(u.Query().Get("known_hosts"))
	for key := range u.Query() {
		if key != "known_hosts" {
			return sftpLocation{}, fmt.Errorf("unsupported SFTP location query parameter %q", key)
		}
	}
	if knownHosts == "" {
		return sftpLocation{}, fmt.Errorf("SFTP location requires known_hosts=<local-file>")
	}
	remotePath, err := url.PathUnescape(u.EscapedPath())
	if err != nil {
		return sftpLocation{}, fmt.Errorf("decode SFTP remote path: %w", err)
	}
	if !strings.HasPrefix(remotePath, "/") || remotePath == "/" {
		return sftpLocation{}, fmt.Errorf("SFTP repository path must be an absolute non-root path")
	}
	for _, part := range strings.Split(remotePath, "/") {
		if part == ".." {
			return sftpLocation{}, fmt.Errorf("SFTP repository path cannot contain parent traversal")
		}
	}
	port := 22
	if rawPort := u.Port(); rawPort != "" {
		port, err = strconv.Atoi(rawPort)
		if err != nil || port < 1 || port > 65535 {
			return sftpLocation{}, fmt.Errorf("SFTP port must be between 1 and 65535")
		}
	}
	return sftpLocation{Host: u.Hostname(), Port: port, Username: username, RemotePath: remotePath, KnownHosts: knownHosts}, nil
}
