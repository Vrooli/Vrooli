import assert from "node:assert/strict";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { createElement, createRef } from "react";
import { test, vi } from "vitest";
import { InvestigationTab, type InvestigationTabHandle } from "../../src/components/dialogs/SettingsDialog/InvestigationTab.js";
import type { InvestigationSettings } from "../../src/types.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

const settings: InvestigationSettings = {
  promptTemplate: "inspect the run", applyPromptTemplate: "apply approved fixes", defaultDepth: "standard",
  defaultContext: { runSummaries: true, runEvents: true, runDiffs: true, fullLogs: false },
  investigationTagAllowlist: [{ pattern: "investigation", isRegex: false, caseSensitive: false }], updatedAt: "2026-01-01T00:00:00Z",
};

test("InvestigationTab loads settings, tracks meaningful edits, and saves the complete investigation contract", async () => {
  const ref = createRef<InvestigationTabHandle>();
  const onSave = vi.fn(async () => undefined);
  const dirty = vi.fn();
  renderWithProviders(createElement(InvestigationTab, { ref, settings, loading: false, error: null, onSave, onReset: vi.fn(), onDirtyChange: dirty }));
  assert.ok(screen.getByDisplayValue("inspect the run"));
  fireEvent.click(screen.getByRole("radio", { name: /Deep/ }));
  fireEvent.change(screen.getByDisplayValue("inspect the run"), { target: { value: "inspect tool failures and cost" } });
  fireEvent.click(screen.getByText("Full logs"));
  await waitFor(() => assert.equal(ref.current?.hasChanges, true));
  await act(async () => { await ref.current?.save(); });
  assert.deepEqual(onSave.mock.calls[0]?.[0], {
    promptTemplate: "inspect tool failures and cost", applyPromptTemplate: "apply approved fixes", defaultDepth: "deep",
    defaultContext: { runSummaries: true, runEvents: true, runDiffs: true, fullLogs: true },
    investigationTagAllowlist: settings.investigationTagAllowlist,
  });
  assert.ok(dirty.mock.calls.some(([value]) => value === true));
});

test("InvestigationTab manages apply tag rules and exposes operator-facing failures", async () => {
  const ref = createRef<InvestigationTabHandle>();
  const onReset = vi.fn(async () => { throw new Error("defaults unavailable"); });
  const { rerender } = renderWithProviders(createElement(InvestigationTab, { ref, settings, loading: false, error: null, onSave: vi.fn(async () => { throw new Error("save unavailable"); }), onReset }));
  fireEvent.click(screen.getByRole("tab", { name: "Apply Investigation" }));
  fireEvent.click(screen.getByText("Add tag rule"));
  const patterns = screen.getAllByPlaceholderText("Tag pattern (glob or regex)");
  fireEvent.change(patterns[1]!, { target: { value: "*-investigation" } });
  await act(async () => { await ref.current?.save(); });
  await waitFor(() => assert.ok(screen.getByText("save unavailable")));
  await act(async () => { await ref.current?.reset(); });
  await waitFor(() => assert.ok(screen.getByText("defaults unavailable")));
  assert.equal(onReset.mock.calls.length, 1);

  rerender(createElement(InvestigationTab, { ref, settings: null, loading: true, error: null, onSave: vi.fn(), onReset: vi.fn() }));
  assert.ok(screen.getByText("Loading investigation settings..."));
});
