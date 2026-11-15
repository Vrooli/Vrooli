import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import axe from "axe-core";
import { describe, expect, it } from "vitest";

const helperSource = readFileSync(
  resolve(process.cwd(), "../api/handlers/preview/assets/axe-evidence.js"),
  "utf8",
);

function installAxeEvidence(target: typeof globalThis) {
  const install = new Function(
    "globalThis",
    `${helperSource}; return globalThis.__rclRunAxeEvidence;`,
  ) as (
    globalThis: typeof globalThis,
  ) => (root: Element, axeRuntime?: unknown) => Promise<Record<string, unknown>>;
  return install(target);
}

const axeForJSDOM = {
  version: axe.version,
  run: (root: Element) => axe.run(root, { rules: { "color-contrast": { enabled: false } } }),
};

describe("RCL browser axe evidence producer", () => {
  it("records a clean, versioned result from a real accessible story DOM", async () => {
    const run = installAxeEvidence(globalThis);
    const root = document.createElement("div");
    root.innerHTML = '<button type="button" aria-label="Save changes"></button>';
    document.body.append(root);

    const result = (await run(root, axeForJSDOM)) as {
      schema: string;
      status: string;
      engine: string;
      version: string;
      violations: unknown[];
    };

    expect(result).toMatchObject({
      schema: "rcl-axe-results/v1",
      status: "complete",
      engine: "axe-core",
      version: axe.version,
      violations: [],
    });
    root.remove();
  });

  it("records actionable violations without copying story HTML or text", async () => {
    const run = installAxeEvidence(globalThis);
    const root = document.createElement("div");
    root.innerHTML = "<p>PRIVATE FIXTURE LABEL</p><button></button>";
    document.body.append(root);

    const result = (await run(root, axeForJSDOM)) as {
      status: string;
      violations: Array<{ id: string; targets: string[] }>;
    };

    expect(result.status).toBe("complete");
    expect(result.violations.map((item) => item.id)).toContain("button-name");
    expect(JSON.stringify(result)).not.toContain("PRIVATE FIXTURE LABEL");
    expect(result.violations[0].targets.length).toBeGreaterThan(0);
    root.remove();
  });

  it("keeps unavailable axe or a missing story root explicitly unmeasured", async () => {
    const run = installAxeEvidence(globalThis);
    expect(await run(null as unknown as Element, axeForJSDOM)).toMatchObject({
      schema: "rcl-axe-results/v1",
      status: "unavailable",
      reason: "story-root-missing",
    });
    expect(await run(document.createElement("div"), null)).toMatchObject({
      schema: "rcl-axe-results/v1",
      status: "unavailable",
      reason: "axe-core-not-loaded",
    });
  });
});
