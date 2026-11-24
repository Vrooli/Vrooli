package pipeline

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	repocontract "github.com/vrooli/repo-contract-go"
	conf "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-plugin/v1/conformance"
	"golang.org/x/text/unicode/norm"
)

type cliManifest struct {
	Name   string `json:"name"`
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
var downloadURL = regexp.MustCompile(`(?i)^https?://[^\s"']+`)

func (h *handler) conformance(p packageRecord) []*conf.Finding {
	findings := make([]*conf.Finding, 0)
	add := func(code, message, path string) {
		findings = append(findings, &conf.Finding{Code: code, Message: message, Path: path})
	}

	manifestRel, relErr := repocontract.ScenarioCLIManifestRel(h.root)
	if relErr != nil {
		manifestRel, _ = repocontract.ScenarioCLIManifestRel("")
	}
	manifestBytes, err := os.ReadFile(filepath.Join(p.ScenarioRoot, filepath.FromSlash(manifestRel)))
	var manifest cliManifest
	if err != nil || json.Unmarshal(manifestBytes, &manifest) != nil {
		add("PLG-CONF-DRIFT", "pinned cli-manifest.json is missing or invalid", manifestRel)
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
			if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
				continue
			}
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
		downloads, verifications := parseInstallIntegrity(s)
		for _, download := range downloads {
			u := download.url
			if u == "" || strings.Contains(u, "latest") || strings.Contains(u, "main") || strings.Contains(u, "master") || strings.Contains(u, "@") {
				add("PLG-CONF-INSTALL-PIN", "download target must use a statically pinned URL", "cli/install.sh")
			}
		}
		if reason := checksumMappingFailure(downloads, verifications); reason != "" {
			add("PLG-CONF-INSTALL-SUM", reason, "cli/install.sh")
		}
	}
	return findings
}

type installDownload struct {
	url    string
	output string
}

type installVerification struct {
	algorithm string
	manifest  string
}

// parseInstallIntegrity handles the install-script command forms supported by
// this gate. Unknown output forms fail closed because a verifier cannot be
// associated with the file written by the download.
func parseInstallIntegrity(script string) ([]installDownload, []installVerification) {
	var downloads []installDownload
	var verifications []installVerification
	for _, sourceLine := range installScriptLines(script) {
		line := strings.TrimSpace(strings.SplitN(sourceLine, "#", 2)[0])
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		for i := range fields {
			fields[i] = strings.Trim(fields[i], `"'`)
		}
		for i, token := range fields {
			command := filepath.Base(token)
			if command != "curl" && command != "wget" {
				continue
			}
			var output, rawURL string
			remoteName := false
			for j := i + 1; j < len(fields); j++ {
				arg := fields[j]
				if downloadURL.MatchString(arg) {
					rawURL = arg
				}
				if arg == "-O" && command == "curl" || arg == "--remote-name" && command == "curl" {
					remoteName = true
					continue
				}
				if (arg == "-o" || arg == "--output" || arg == "-O" && command == "wget" || arg == "--output-document") && j+1 < len(fields) {
					output = fields[j+1]
					j++
					continue
				}
				for _, prefix := range []string{"--output=", "--output-document="} {
					if strings.HasPrefix(arg, prefix) {
						output = strings.TrimPrefix(arg, prefix)
					}
				}
				if len(arg) > 2 && (strings.HasPrefix(arg, "-o") || strings.HasPrefix(arg, "-O") && command == "wget") {
					output = arg[2:]
				}
			}
			if output == "" && rawURL != "" && (command == "wget" || remoteName) {
				// curl -O and wget without an output override save the URL basename.
				output = path.Base(strings.SplitN(rawURL, "?", 2)[0])
			}
			if output != "" {
				output = filepath.Clean(output)
			}
			downloads = append(downloads, installDownload{url: rawURL, output: output})
			break
		}
		for i, token := range fields {
			algorithm := ""
			if token == "sha256sum" || token == "sha512sum" {
				algorithm = strings.TrimSuffix(token, "sum")
			} else if token == "shasum" && i+2 < len(fields) && fields[i+1] == "-a" {
				if fields[i+2] == "256" || fields[i+2] == "512" {
					algorithm = "sha" + fields[i+2]
				}
			}
			if algorithm == "" {
				continue
			}
			for j := i + 1; j < len(fields); j++ {
				if fields[j] == "--check" || fields[j] == "-c" {
					if j+1 < len(fields) {
						verifications = append(verifications, installVerification{algorithm: algorithm, manifest: filepath.Clean(fields[j+1])})
					}
					break
				}
			}
			break
		}
	}
	return downloads, verifications
}

func installScriptLines(script string) []string {
	var lines []string
	var continuation strings.Builder
	for _, line := range strings.Split(strings.ReplaceAll(script, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimRight(line, " \t")
		if strings.HasSuffix(trimmed, "\\") {
			continuation.WriteString(strings.TrimSuffix(trimmed, "\\"))
			continuation.WriteByte(' ')
			continue
		}
		continuation.WriteString(line)
		lines = append(lines, continuation.String())
		continuation.Reset()
	}
	if continuation.Len() > 0 {
		lines = append(lines, continuation.String())
	}
	return lines
}

func checksumMappingFailure(downloads []installDownload, verifications []installVerification) string {
	matched := make([]bool, len(verifications))
	for _, download := range downloads {
		if download.output == "" {
			return fmt.Sprintf("download from %q has no statically identifiable output file", download.url)
		}
		isSidecar := false
		for _, candidate := range downloads {
			if download.output == candidate.output+".sha256" || download.output == candidate.output+".sha512" {
				isSidecar = true
				break
			}
		}
		if isSidecar {
			continue // a checksum sidecar supplies the expected digest for its payload
		}
		match := -1
		for i, verification := range verifications {
			if matched[i] {
				continue
			}
			if verification.manifest == download.output+".sha256" && verification.algorithm == "sha256" || verification.manifest == download.output+".sha512" && verification.algorithm == "sha512" {
				if match >= 0 {
					return "a download has more than one checksum verifier; each payload must map to exactly one matching verifier"
				}
				match = i
			}
		}
		if match < 0 {
			return fmt.Sprintf("download %q requires a matching sha256 or sha512 --check file named %q.sha256 or %q.sha512", download.output, download.output, download.output)
		}
		matched[match] = true
	}
	for i := range verifications {
		if !matched[i] {
			return "checksum verifier does not map one-to-one to a downloaded payload"
		}
	}
	return ""
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
