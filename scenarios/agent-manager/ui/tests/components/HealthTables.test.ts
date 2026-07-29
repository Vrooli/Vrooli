import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { test, vi } from "vitest";
import { ModelHealthTable } from "../../src/features/health/components/ModelHealthTable.js";
import { RunnerHealthTable } from "../../src/features/health/components/RunnerHealthTable.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

test("health tables render empty states, severity-sorted observations, fallback reasons, and audit actions", async () => {
  const user = userEvent.setup();
  const onRunnerAudit = vi.fn();
  const onModelAudit = vi.fn();
  const { rerender } = renderWithProviders(createElement(RunnerHealthTable, { rows: [], onShowAudit: onRunnerAudit }));
  assert.ok(screen.getByTestId("runner-health-empty"));
  rerender(createElement(RunnerHealthTable, { onShowAudit: onRunnerAudit, rows: [
    { runner: "zeta", status: "ok", last_checked: "2026-07-29T00:00:00Z" },
    { runner: "alpha", status: "failed", reason: "offline", last_checked: "2026-07-29T00:00:00Z" },
  ] } as any));
  assert.equal(screen.getAllByTestId(/runner-health-row-/)[0]!.getAttribute("data-testid"), "runner-health-row-alpha");
  await user.click(screen.getByRole("button", { name: "Show audit history for runner alpha" }));
  assert.deepEqual(onRunnerAudit.mock.calls, [["alpha"]]);

  rerender(createElement(ModelHealthTable, { rows: [], onShowAudit: onModelAudit }));
  assert.ok(screen.getByTestId("model-health-empty"));
  rerender(createElement(ModelHealthTable, { onShowAudit: onModelAudit, rows: [
    { runner: "codex", model: "gpt-5", status: "ok", last_checked: "2026-07-29T00:00:00Z" },
    { runner: "claude", model: "sonnet", status: "unknown", last_checked: "2026-07-29T00:00:00Z" },
    { runner: "codex", model: "mini", status: "failed", reason: "quota", last_checked: "2026-07-29T00:00:00Z" },
  ] } as any));
  assert.equal(screen.getAllByTestId(/model-health-row-/)[0]!.getAttribute("data-testid"), "model-health-row-codex-mini");
  await user.click(screen.getByRole("button", { name: "Show audit history for codex / mini" }));
  assert.deepEqual(onModelAudit.mock.calls, [["codex", "mini"]]);
});

test("health tables rank unknown between failed and healthy, then alphabetize equal severities", () => {
  const onRunnerAudit = vi.fn();
  const onModelAudit = vi.fn();
  const { rerender } = renderWithProviders(createElement(RunnerHealthTable, { onShowAudit: onRunnerAudit, rows: [
    { runner: "zeta", status: "unknown", last_checked: "2026-07-29T00:00:00Z" },
    { runner: "alpha", status: "unknown", last_checked: "2026-07-29T00:00:00Z" },
    { runner: "ok", status: "ok", last_checked: "2026-07-29T00:00:00Z" },
  ] } as any));
  assert.deepEqual(screen.getAllByTestId(/runner-health-row-/).map((row) => row.dataset.testid), ["runner-health-row-alpha", "runner-health-row-zeta", "runner-health-row-ok"]);
  rerender(createElement(ModelHealthTable, { onShowAudit: onModelAudit, rows: [
    { runner: "codex", model: "z", status: "unknown", last_checked: "2026-07-29T00:00:00Z" },
    { runner: "codex", model: "a", status: "unknown", last_checked: "2026-07-29T00:00:00Z" },
    { runner: "alpha", model: "z", status: "unknown", last_checked: "2026-07-29T00:00:00Z" },
  ] } as any));
  assert.deepEqual(screen.getAllByTestId(/model-health-row-/).map((row) => row.dataset.testid), ["model-health-row-alpha-z", "model-health-row-codex-a", "model-health-row-codex-z"]);
});
