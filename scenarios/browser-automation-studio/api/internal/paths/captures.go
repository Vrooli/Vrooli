package paths

import "github.com/sirupsen/logrus"

// ResolveCapturesRoot returns an absolute base directory for capture output
// bundles. Relative --out values (and the default when --out is omitted)
// resolve under this root so capture artifacts land in scenario-owned storage
// instead of whatever working directory the API process happened to start in.
func ResolveCapturesRoot(log *logrus.Logger) string {
	return resolveScenarioDataRoot(log, "BAS_CAPTURES_ROOT", "captures", "captures")
}
