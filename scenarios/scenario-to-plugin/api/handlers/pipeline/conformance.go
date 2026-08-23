package pipeline

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	conf "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/conformance"
	"golang.org/x/text/unicode/norm"
)

type cliManifest struct {
	Groups []struct {
		Name     string `json:"name"`
		Commands []struct {
			Name  string `json:"name"`
			Flags []struct {
				Name string `json:"name"`
			} `json:"flags"`
		} `json:"commands"`
	} `json:"groups"`
}

var commandLine = regexp.MustCompile(`(?m)^[ \t]*([A-Za-z0-9._-]+)[ \t]+([A-Za-z0-9._-]+)([^\n]*)$`)
var downloadLine = regexp.MustCompile(`(?i)(curl|wget)[^\n]*(https?://[^\s"']+)`)

func (h *handler) conformance(p packageRecord) []*conf.Finding {
	findings := make([]*conf.Finding, 0)
	add := func(code, message, path string) {
		findings = append(findings, &conf.Finding{Code: code, Message: message, Path: path})
	}

	manifestBytes, err := os.ReadFile(filepath.Join(p.ScenarioRoot, "cli", "manifest.json"))
	var manifest cliManifest
	if err != nil || json.Unmarshal(manifestBytes, &manifest) != nil {
		add("PLG-CONF-DRIFT", "pinned cli-manifest.json is missing or invalid", "cli/manifest.json")
	}
	groups := map[string]bool{}
	flags := map[string]map[string]bool{}
	for _, g := range manifest.Groups {
		groups[g.Name] = true
		flags[g.Name] = map[string]bool{}
		for _, c := range g.Commands {
			for _, f := range c.Flags {
				flags[g.Name][f.Name] = true
			}
		}
	}

	for _, entry := range p.PackageSkills() {
		body, err := os.ReadFile(filepath.Join(p.Root, entry))
		if err != nil {
			add("PLG-CONF-SPEC", "declared skill is missing", entry)
			continue
		}
		text := string(body)
		if !utf8.Valid(body) {
			add("PLG-CONF-UNICODE", "skill is not valid UTF-8", entry)
		}
		for i, r := range text {
			if r == '\u200b' || r == '\u200c' || r == '\u200d' || (r >= '\u202a' && r <= '\u202e') || (r >= '\u2066' && r <= '\u2069') {
				add("PLG-CONF-UNICODE", fmt.Sprintf("hidden Unicode control at byte offset %d", i), entry)
			}
		}
		if !norm.NFC.IsNormalString(text) {
			normalized := norm.NFC.String(text)
			offset := firstByteDifference(text, normalized)
			add("PLG-CONF-UNICODE", fmt.Sprintf("skill is not NFC normalized at byte offset %d", offset), entry)
		}
		front := text
		frontmatter := map[string]string{}
		if strings.HasPrefix(front, "---\n") {
			if end := strings.Index(front[4:], "\n---"); end >= 0 {
				front = front[4 : 4+end]
				for _, line := range strings.Split(front, "\n") {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						frontmatter[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
					}
				}
			} else {
				add("PLG-CONF-FRONTMATTER", "frontmatter is not closed", entry)
			}
		} else {
			add("PLG-CONF-FRONTMATTER", "skill requires YAML frontmatter", entry)
		}
		folder := filepath.Base(filepath.Dir(entry))
		name := frontmatter["name"]
		description := frontmatter["description"]
		if name == "" || description == "" {
			add("PLG-CONF-FRONTMATTER", "name and description are required", entry)
		}
		if name != folder || len(name) > 64 || !validSkillName(name) {
			add("PLG-CONF-FRONTMATTER", "skill name must match its folder and Agent Skills naming rules", entry)
		}
		if len(description) > 1024 {
			add("PLG-CONF-FRONTMATTER", "skill description exceeds 1024 characters", entry)
		}
		for _, line := range strings.Split(front, "\n") {
			if strings.Contains(line, ":") {
				key := strings.TrimSpace(strings.SplitN(line, ":", 2)[0])
				if key != "name" && key != "description" && key != "allowed-tools" && key != "license" && key != "compatibility" && key != "metadata" {
					add("PLG-CONF-FRONTMATTER", "unsupported frontmatter key", entry)
				}
			}
		}
		if strings.Contains(front, "<") || strings.Contains(front, ">") {
			add("PLG-CONF-ANGLE", "skill frontmatter contains angle brackets", entry)
		}
		allowed := strings.ToLower(frontmatter["allowed-tools"])
		if strings.Contains(allowed, "bash(*)") || strings.Contains(allowed, "shell(*)") || strings.Contains(allowed, "network(*)") || strings.Contains(allowed, "filesystem(*)") || strings.Contains(allowed, "fs(*)") {
			add("PLG-CONF-TOOLS", "allowed-tools grants unrestricted shell, network, or filesystem access", entry)
		}
		for _, match := range commandLine.FindAllStringSubmatch(text, -1) {
			if strings.HasPrefix(strings.TrimSpace(match[0]), "#") || strings.HasPrefix(strings.TrimSpace(match[0]), "---") {
				continue
			}
			binary := match[1]
			group := match[2]
			if !groups[group] {
				continue
			} // prose and headings are not commands
			if binary == "" {
				add("PLG-CONF-DRIFT", "documented command has no executable", entry)
				continue
			}
			fields := strings.Fields(match[3])
			command := ""
			if len(fields) > 0 {
				command = strings.TrimSpace(fields[0])
			}
			if command != "" && !strings.HasPrefix(command, "-") && command != "\\" {
				found := false
				for _, g := range manifest.Groups {
					if g.Name == group {
						for _, c := range g.Commands {
							if c.Name == command {
								found = true
							}
						}
					}
				}
				if !found {
					add("PLG-CONF-DRIFT", "documented command is absent from pinned cli-manifest", entry)
				}
			}
			if strings.Contains(match[3], "--") {
				for _, flag := range strings.Fields(match[3]) {
					if strings.HasPrefix(flag, "--") && !flags[group][strings.TrimPrefix(flag, "--")] && !strings.Contains(flag, "=") {
						add("PLG-CONF-DRIFT", "documented flag is absent from pinned cli-manifest", entry)
					}
				}
			}
		}
	}
	install, err := os.ReadFile(filepath.Join(p.Root, "cli/install.sh"))
	if err != nil {
		add("PLG-CONF-INSTALL-PRIV", "install script is missing", "cli/install.sh")
	} else {
		s := string(install)
		if strings.Contains(s, "sudo") || strings.Contains(s, "doas") || strings.Contains(s, "/usr/local") {
			add("PLG-CONF-INSTALL-PRIV", "install script requests privilege or a system prefix", "cli/install.sh")
		}
		for _, line := range strings.Split(s, "\n") {
			if (strings.Contains(line, ">") || strings.Contains(line, " cp ") || strings.Contains(line, " mv ") || strings.Contains(line, " install ")) && containsOutsideUserPrefix(line) {
				add("PLG-CONF-INSTALL-PRIV", "install script writes outside its user-scoped prefix", "cli/install.sh")
			}
		}
		for _, match := range downloadLine.FindAllStringSubmatch(s, -1) {
			u := match[2]
			if strings.Contains(u, "latest") || strings.Contains(u, "main") || strings.Contains(u, "master") || strings.Contains(u, "@") {
				add("PLG-CONF-INSTALL-PIN", "download target is mutable", "cli/install.sh")
			}
			if !strings.Contains(s, "sha256") && !strings.Contains(s, "sha512") && !strings.Contains(s, "sha1sum") {
				add("PLG-CONF-INSTALL-SUM", "download has no checksum verification", "cli/install.sh")
			}
		}
	}
	return findings
}

func firstByteDifference(left, right string) int {
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}
	for i := 0; i < limit; i++ {
		if left[i] != right[i] {
			return i
		}
	}
	return limit
}

func containsOutsideUserPrefix(line string) bool {
	for _, token := range strings.FieldsFunc(line, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '"' || r == '\'' || r == ';' || r == '(' || r == ')'
	}) {
		if !strings.HasPrefix(token, "/") {
			continue
		}
		if strings.HasPrefix(token, "/tmp/") || strings.HasPrefix(token, "/etc/") || strings.HasPrefix(token, "/var/") || strings.HasPrefix(token, "/opt/") || strings.HasPrefix(token, "/usr/") {
			return true
		}
	}
	return false
}

func (p packageRecord) PackageSkills() []string {
	out := make([]string, 0)
	entries, _ := os.ReadDir(filepath.Join(p.Root, "skills"))
	for _, entry := range entries {
		if entry.IsDir() {
			path := filepath.Join("skills", entry.Name(), "SKILL.md")
			if _, err := os.Stat(filepath.Join(p.Root, path)); err == nil {
				out = append(out, path)
			}
		}
	}
	return out
}

func validSkillName(name string) bool {
	if name == "" || strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return false
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '-' {
			return false
		}
	}
	return true
}

func cliManifestRevision(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return "missing"
	}
	sum := sha256.Sum256(b)
	return "sha256:" + fmt.Sprintf("%x", sum[:])
}
