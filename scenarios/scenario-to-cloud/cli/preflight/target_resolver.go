package preflight

import (
	"encoding/json"
	"flag"
	"fmt"
	"strconv"
	"strings"

	"scenario-to-cloud/cli/deployment"
)

type targetContext struct {
	Host         string
	Port         int
	User         string
	KeyPath      string
	Workdir      string
	ScenarioID   string
	DeploymentID string
}

type preflightTargetFlags struct {
	host       *string
	scenarioID *string
	domain     *string
	target     *string
	user       *string
	keyPath    *string
	sshPort    *int
	workdir    *string
}

func registerPreflightTargetFlags(fs *flag.FlagSet) preflightTargetFlags {
	return preflightTargetFlags{
		host:       fs.String("host", "", "VPS host selector"),
		scenarioID: fs.String("scenario", "", "Scenario ID selector"),
		domain:     fs.String("domain", "", "Domain selector"),
		target:     fs.String("target", "", "Convenience selector (domain or host)"),
		user:       fs.String("user", "", "Override SSH user"),
		keyPath:    fs.String("key-path", "", "Override SSH key path"),
		sshPort:    fs.Int("ssh-port", 0, "Override SSH port"),
		workdir:    fs.String("workdir", "", "Override VPS workdir (used by fix-processes)"),
	}
}

func (f preflightTargetFlags) resolve(client *Client) (targetContext, error) {
	selector, err := f.toSelector()
	if err != nil {
		return targetContext{}, err
	}

	deploymentClient := deployment.NewClient(client.APIClient())
	dep, err := deployment.ResolveLatestBySelector(deploymentClient, selector)
	if err != nil {
		return targetContext{}, fmt.Errorf("resolve deployment by selector: %w", err)
	}
	if dep == nil {
		return targetContext{}, fmt.Errorf(
			"no deployment found for selector host=%s scenario=%s domain=%s target=%s",
			displayOrNA(selector.Host),
			displayOrNA(selector.ScenarioID),
			displayOrNA(selector.Domain),
			displayOrNA(selector.Target),
		)
	}

	_, getResp, err := deploymentClient.Get(dep.ID)
	if err != nil {
		return targetContext{}, fmt.Errorf("get deployment %s: %w", dep.ID, err)
	}
	if getResp.Deployment == nil {
		return targetContext{}, fmt.Errorf("deployment %s not found", dep.ID)
	}

	manifest := make(map[string]interface{})
	if len(getResp.Deployment.Manifest) > 0 {
		if err := json.Unmarshal(getResp.Deployment.Manifest, &manifest); err != nil {
			return targetContext{}, fmt.Errorf("decode deployment manifest: %w", err)
		}
	}

	ctx := targetContext{
		Host:         strings.TrimSpace(getNestedString(manifest, "target", "vps", "host")),
		Port:         getNestedInt(manifest, "target", "vps", "port"),
		User:         strings.TrimSpace(getNestedString(manifest, "target", "vps", "user")),
		KeyPath:      strings.TrimSpace(getNestedString(manifest, "target", "vps", "key_path")),
		Workdir:      strings.TrimSpace(getNestedString(manifest, "target", "vps", "workdir")),
		ScenarioID:   strings.TrimSpace(getNestedString(manifest, "scenario", "id")),
		DeploymentID: dep.ID,
	}

	if ctx.Host == "" {
		ctx.Host = strings.TrimSpace(dep.Host)
	}
	if ctx.Port == 0 {
		ctx.Port = 22
	}
	if ctx.ScenarioID == "" {
		ctx.ScenarioID = strings.TrimSpace(dep.ScenarioID)
	}
	if ctx.Workdir == "" {
		ctx.Workdir = "/root/Vrooli"
	}

	// Explicit command flags take precedence over resolved deployment values.
	if v := strings.TrimSpace(*f.host); v != "" {
		ctx.Host = v
	}
	if v := strings.TrimSpace(*f.user); v != "" {
		ctx.User = v
	}
	if v := strings.TrimSpace(*f.keyPath); v != "" {
		ctx.KeyPath = v
	}
	if v := *f.sshPort; v > 0 {
		ctx.Port = v
	}
	if v := strings.TrimSpace(*f.workdir); v != "" {
		ctx.Workdir = v
	}
	if v := strings.TrimSpace(*f.scenarioID); v != "" {
		ctx.ScenarioID = v
	}

	if ctx.Host == "" {
		return targetContext{}, fmt.Errorf("resolved deployment %s is missing target.vps.host", dep.ID)
	}
	if ctx.KeyPath == "" {
		return targetContext{}, fmt.Errorf("resolved deployment %s is missing target.vps.key_path; provide --key-path", dep.ID)
	}

	return ctx, nil
}

func (f preflightTargetFlags) toSelector() (deployment.ManifestSelector, error) {
	host := strings.TrimSpace(*f.host)
	scenarioID := strings.TrimSpace(*f.scenarioID)
	domain := strings.TrimSpace(*f.domain)
	target := strings.TrimSpace(*f.target)

	if target != "" && (host != "" || domain != "") {
		return deployment.ManifestSelector{}, fmt.Errorf("--target cannot be combined with --host or --domain")
	}
	if host == "" && domain == "" && target == "" {
		return deployment.ManifestSelector{}, fmt.Errorf("at least one selector is required: --host, --domain, or --target")
	}

	return deployment.ManifestSelector{
		Host:       host,
		ScenarioID: scenarioID,
		Domain:     domain,
		Target:     target,
	}, nil
}

func getNestedInt(m map[string]interface{}, path ...string) int {
	value := getNestedValue(m, path...)
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case json.Number:
		n, _ := strconv.Atoi(v.String())
		return n
	default:
		return 0
	}
}

func getNestedString(m map[string]interface{}, path ...string) string {
	value := getNestedValue(m, path...)
	if value == nil {
		return ""
	}
	s, ok := value.(string)
	if !ok {
		return ""
	}
	return s
}

func getNestedValue(m map[string]interface{}, path ...string) interface{} {
	var current interface{} = m
	for i, key := range path {
		obj, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		value, ok := obj[key]
		if !ok {
			return nil
		}
		if i == len(path)-1 {
			return value
		}
		current = value
	}
	return nil
}

func displayOrNA(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "n/a"
	}
	return v
}
