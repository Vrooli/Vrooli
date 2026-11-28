import assert from "node:assert/strict";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { afterEach, test, vi } from "vitest";
import { AutoSaveRules } from "./AutoSaveRules.js";
import { renderWithProviders } from "../../test-utils/index.js";

const rule = { runner: "codex", since: "2026-11-15T10:00:00Z", by: "operator@example.com", starts: 2, last_start: "2026-11-15T12:00:00Z" };

function serve(rules: unknown[]) {
  const calls: { url: string; method: string; body: string }[] = [];
  vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    calls.push({ url: String(input), method: init?.method ?? "GET", body: String(init?.body ?? "") });
    if (init?.method === "POST") {
      rules = [];
      return new Response(JSON.stringify({ rules }), { status: 200 });
    }
    return new Response(JSON.stringify({ rules }), { status: 200 });
  }));
  return calls;
}

afterEach(() => vi.unstubAllGlobals());

test("a standing rule stays visible with what it does and how often it ran, and one tap turns it off", async () => {
  const user = userEvent.setup();
  const calls = serve([rule]);
  renderWithProviders(createElement(AutoSaveRules));
  const panel = await screen.findByTestId("auto-save-rule-codex");
  assert.match(panel.textContent ?? "", /Save usage starts by itself for Codex/);
  assert.match(panel.textContent ?? "", /notice instead of a question/);
  assert.match(panel.textContent ?? "", /Started 2 times/);
  await user.click(within(panel).getByTestId("auto-save-off-codex"));
  await waitFor(() => assert.equal(screen.queryByTestId("auto-save-rules"), null));
  const write = calls.find((call) => call.method === "POST");
  assert.ok(write?.url.endsWith("/usage/modes/auto-save"));
  assert.deepEqual(JSON.parse(write?.body ?? "{}"), { runner: "codex", enabled: false });
});

test("nothing is shown while no rule is on", async () => {
  const calls = serve([]);
  renderWithProviders(createElement(AutoSaveRules));
  await waitFor(() => assert.equal(calls.length > 0, true));
  assert.equal(screen.queryByTestId("auto-save-rules"), null);
});
