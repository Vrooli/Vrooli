package main

import (
	"fmt"
	"net"
	"path"
	"strconv"
	"strings"
)

func isIPLiteral(host string) bool {
	return net.ParseIP(strings.TrimSpace(host)) != nil
}

func shellQuoteSingle(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func localSSHCommand(cfg SSHConfig, cmd string) string {
	args := []string{
		"ssh",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-p", strconv.Itoa(cfg.Port),
	}
	if strings.TrimSpace(cfg.KeyPath) != "" {
		args = append(args, "-i", cfg.KeyPath)
	}
	args = append(args, fmt.Sprintf("%s@%s", cfg.User, cfg.Host), "--", "bash", "-lc", shellQuoteSingle(cmd))
	return strings.Join(args, " ")
}

func localSCPCommand(cfg SSHConfig, localPath, remotePath string) string {
	args := []string{
		"scp",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-P", strconv.Itoa(cfg.Port),
	}
	if strings.TrimSpace(cfg.KeyPath) != "" {
		args = append(args, "-i", cfg.KeyPath)
	}
	args = append(args, localPath, fmt.Sprintf("%s@%s:%s", cfg.User, cfg.Host, remotePath))
	return strings.Join(args, " ")
}

func safeRemoteJoin(elem ...string) string {
	cleaned := make([]string, 0, len(elem))
	for _, e := range elem {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		cleaned = append(cleaned, e)
	}
	if len(cleaned) == 0 {
		return ""
	}
	return path.Clean(path.Join(cleaned...))
}
