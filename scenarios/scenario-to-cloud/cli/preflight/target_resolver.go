package preflight

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strconv"
	"strings"

	"scenario-to-cloud/cli/internal/selector"
)

// targetContext is the target a preflight fix acts on, resolved from the
// deployment's identity (locator) and its manifest. The transport
// authenticates through the deployment's credential binding (or the
// operator's ambient identity); no key path travels on the wire.
type targetContext struct {
	Host         string
	Port         int
	User         string
	Workdir      string
	ScenarioID   string
	DeploymentID string
}

type preflightTargetFlags struct {
	sel     *selector.Flags
	user    *string
	sshPort *int
	workdir *string
}

func registerPreflightTargetFlags(fs *flag.FlagSet) preflightTargetFlags {
	return preflightTargetFlags{
		sel:     selector.Register(fs),
		user:    fs.String("user", "", "Override SSH user"),
		sshPort: fs.Int("ssh-port", 0, "Override SSH port"),
		workdir: fs.String("workdir", "", "Override VPS workdir (used by fix-processes)"),
	}
}

// resolve maps the selector to one deployment through the API and reads the
// SSH facts from its identity and manifest. Explicit overrides win.
func (f preflightTargetFlags) resolve(client *Client, positional []string) (targetContext, error) {
	chosen, err := f.sel.Selector(positional)
	if err != nil {
		return targetContext{}, err
	}
	ref, err := client.Deployments.Resolve(context.Background(), chosen)
	if err != nil {
		return targetContext{}, err
	}
	record, err := client.Deployments.Get(context.Background(), ref.GetId())
	if err != nil {
		return targetContext{}, fmt.Errorf("get deployment %s: %w", ref.GetId(), err)
	}
	manifest := record.GetDeployment().GetManifest().AsMap()
	ctx := targetContext{
		Host:         strings.TrimSpace(getNestedString(manifest, "target", "vps", "host")),
		Port:         getNestedInt(manifest, "target", "vps", "port"),
		User:         strings.TrimSpace(getNestedString(manifest, "target", "vps", "user")),
		Workdir:      strings.TrimSpace(getNestedString(manifest, "target", "vps", "workdir")),
		ScenarioID:   ref.GetScenarioId(),
		DeploymentID: ref.GetId(),
	}
	if loc := ref.GetTarget().GetLocator(); loc != nil {
		if ctx.Host == "" {
			ctx.Host = strings.TrimSpace(loc.GetHost())
		}
		if ctx.Port == 0 {
			ctx.Port = int(loc.GetPort())
		}
		if ctx.User == "" {
			ctx.User = strings.TrimSpace(loc.GetUser())
		}
		if ctx.Workdir == "" {
			ctx.Workdir = strings.TrimSpace(loc.GetWorkdir())
		}
	}
	if ctx.Port == 0 {
		ctx.Port = 22
	}
	if ctx.Workdir == "" {
		ctx.Workdir = "/root/Vrooli"
	}
	if v := strings.TrimSpace(*f.user); v != "" {
		ctx.User = v
	}
	if v := *f.sshPort; v > 0 {
		ctx.Port = v
	}
	if v := strings.TrimSpace(*f.workdir); v != "" {
		ctx.Workdir = v
	}
	if ctx.Host == "" {
		return targetContext{}, fmt.Errorf("resolved deployment %s has no target host", ref.GetId())
	}
	return ctx, nil
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
