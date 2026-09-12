import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import { createElement } from "react";
import { afterEach, test, vi } from "vitest";
import { ModelFailureAlertBanner } from "../../src/features/stats/components/operational/ModelFailureAlertBanner.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

const state = vi.hoisted(() => ({ hook: vi.fn() }));
vi.mock("../../src/features/stats/hooks/useOperationalStats.js", () => ({ useHealthSummary: state.hook }));
afterEach(() => vi.resetAllMocks());

test("ModelFailureAlertBanner stays absent while no recent model failures are reported", () => {
  state.hook.mockReturnValue({ data: undefined });
  const empty = renderWithProviders(createElement(ModelFailureAlertBanner));
  assert.equal(empty.container.innerHTML, "");
  empty.unmount();

  state.hook.mockReturnValue({ data: { failing_last_hour: [] } });
  const healthy = renderWithProviders(createElement(ModelFailureAlertBanner));
  assert.equal(healthy.container.innerHTML, "");
});

test("ModelFailureAlertBanner names recent failures, optional reasons, and the health destination", () => {
  state.hook.mockReturnValue({ data: { failing_last_hour: [
    { runner: "codex", model: "gpt-5", reason: "rate limited" },
    { runner: "claude", model: "opus" },
    { runner: "runner-3", model: "model-3" },
    { runner: "runner-4", model: "model-4" },
    { runner: "runner-5", model: "model-5" },
    { runner: "runner-6", model: "model-6" },
  ] } });

  renderWithProviders(createElement(ModelFailureAlertBanner));

  assert.equal(screen.getByRole("alert").getAttribute("data-testid"), "model-failure-banner");
  assert.ok(screen.getByText("6 models failed in the last hour"));
  assert.ok(screen.getByText("codex / gpt-5 — rate limited"));
  assert.ok(screen.getByText("claude / opus"));
  assert.equal(screen.queryByText("runner-6 / model-6"), null);
  assert.equal(screen.getByRole("link", { name: /open the health page/i }).getAttribute("href"), "/observability");
});

test("ModelFailureAlertBanner uses singular grammar for one failed model", () => {
  state.hook.mockReturnValue({ data: { failing_last_hour: [{ runner: "codex", model: "gpt-5" }] } });
  renderWithProviders(createElement(ModelFailureAlertBanner));
  assert.ok(screen.getByText("1 model failed in the last hour"));
});
