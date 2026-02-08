package deployment

import (
	"flag"
	"fmt"
	"strings"
)

type selectorFlags struct {
	host       *string
	scenarioID *string
	domain     *string
	target     *string
}

func registerSelectorFlags(fs *flag.FlagSet) selectorFlags {
	return selectorFlags{
		host:       fs.String("host", "", "VPS host selector"),
		scenarioID: fs.String("scenario", "", "Scenario ID selector"),
		domain:     fs.String("domain", "", "Domain selector"),
		target:     fs.String("target", "", "Convenience selector (domain or host)"),
	}
}

func (s selectorFlags) anySet() bool {
	return strings.TrimSpace(*s.host) != "" ||
		strings.TrimSpace(*s.scenarioID) != "" ||
		strings.TrimSpace(*s.domain) != "" ||
		strings.TrimSpace(*s.target) != ""
}

func (s selectorFlags) toSelector() (ManifestSelector, error) {
	host := strings.TrimSpace(*s.host)
	scenarioID := strings.TrimSpace(*s.scenarioID)
	domain := strings.TrimSpace(*s.domain)
	target := strings.TrimSpace(*s.target)

	if target != "" && (host != "" || domain != "") {
		return ManifestSelector{}, fmt.Errorf("--target cannot be combined with --host or --domain")
	}
	if host == "" && domain == "" && target == "" {
		return ManifestSelector{}, fmt.Errorf("at least one selector is required: --host, --domain, or --target")
	}

	return ManifestSelector{
		Host:       host,
		ScenarioID: scenarioID,
		Domain:     domain,
		Target:     target,
	}, nil
}
