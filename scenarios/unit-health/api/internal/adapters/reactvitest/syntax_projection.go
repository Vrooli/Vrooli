package reactvitest

import (
	"fmt"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/quality-health/v1/audit"
	"unit-health/internal/adapters"
)

// This digest binds the reviewed advisory rule selection and alwaysAwait=false
// option, not merely the upstream plugin version. It is produced from the
// actual Quality Health configuration and calibrated against good/bad cases.
const syntaxConfigDigest = "831500ad6837f6467c495f22adac68f188cdf86d40d6480a90147c3f3694b23e"

func syntaxProjection(observation *auditv1.TestSyntaxObservation) adapters.ProjectionCheck {
	const expected = "vitest-syntax-1.6.9 / plugin 1.6.9 / eslint 9.39.4 / parser 8.59.2"
	pass := observation.GetSchemaVersion() == "vitest-lint/v1" && observation.GetProfile() == "vitest-syntax-1.6.9" && observation.GetPluginVersion() == "1.6.9" && observation.GetEslintVersion() == "9.39.4" && observation.GetParserVersion() == "8.59.2" && observation.GetConfigDigest() == syntaxConfigDigest
	return adapters.ProjectionCheck{Key: "test-syntax-native-configuration", Owner: "quality-health", File: observation.GetFile(), PolicyValue: expected + " / " + syntaxConfigDigest, NativeValue: fmt.Sprintf("%s / plugin %s / eslint %s / parser %s / %s", observation.GetProfile(), observation.GetPluginVersion(), observation.GetEslintVersion(), observation.GetParserVersion(), observation.GetConfigDigest()), Pass: pass, Remediation: "Use the reviewed owner profile or calibrate and version the changed native configuration. Do not install a duplicate target lint pass."}
}
