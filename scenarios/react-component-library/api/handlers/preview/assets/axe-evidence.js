// The isolated component-story harness calls this after story interactions and
// assertions. Keep only rule identity and target selectors in the persisted
// result; axe node HTML can contain fixture text and is not needed to fix a rule.
(function installRCLAxeEvidence(target) {
  target.__rclRunAxeEvidence = async function runRCLAxeEvidence(root, axeRuntime = target.axe) {
    if (!root || root.nodeType !== 1) {
      return { schema: "rcl-axe-results/v1", status: "unavailable", reason: "story-root-missing" };
    }
    if (!axeRuntime || typeof axeRuntime.run !== "function" || typeof axeRuntime.version !== "string") {
      return { schema: "rcl-axe-results/v1", status: "unavailable", reason: "axe-core-not-loaded" };
    }
    try {
      const report = await axeRuntime.run(root);
      return {
        schema: "rcl-axe-results/v1",
        status: "complete",
        engine: "axe-core",
        version: axeRuntime.version,
        violations: (report.violations || []).map((violation) => ({
          id: String(violation.id || "unnamed-violation"),
          impact: String(violation.impact || "unspecified impact"),
          description: String(violation.description || violation.help || "Axe rule failed."),
          targets: (violation.nodes || []).flatMap((node) => Array.isArray(node.target) ? node.target : []).map(String),
        })),
      };
    } catch (_error) {
      return { schema: "rcl-axe-results/v1", status: "unavailable", reason: "axe-core-run-failed" };
    }
  };
})(globalThis);
