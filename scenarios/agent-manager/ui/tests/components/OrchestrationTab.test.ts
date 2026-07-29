import assert from "node:assert/strict";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { createElement, createRef } from "react";
import { test, vi } from "vitest";
import { OrchestrationTab, type OrchestrationTabHandle } from "../../src/components/dialogs/SettingsDialog/OrchestrationTab.js";
import type { OrchestrationSettings } from "../../src/hooks/useOrchestrationSettings.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

const settings: OrchestrationSettings = {
  runExecution: { runTimeoutMinutes: 20, maxConcurrentRuns: 2, maxTurns: 8 },
  safetyIsolation: { requireSandbox: true, requireApproval: true, networkAccess: "localhost" },
  healthDetection: { heartbeatIntervalSeconds: 10, staleThresholdSeconds: 60, maxRecoveryAgeSeconds: 120, reconcilerIntervalSeconds: 30 },
  processTermination: { gracePeriodSeconds: 5, killProcessGroup: true, killOrphans: true, orphanGracePeriodSeconds: 10, terminationMaxRetries: 2 },
};

test("OrchestrationTab reports loading and unavailable states", () => {
  const { rerender } = renderWithProviders(createElement(OrchestrationTab, { settings: null, loading: true, error: null, onSave: vi.fn(), onReset: vi.fn() }));
  assert.ok(screen.getByText("Loading orchestration settings..."));
  rerender(createElement(OrchestrationTab, { settings: null, loading: false, error: null, onSave: vi.fn(), onReset: vi.fn() }));
  assert.ok(screen.getByText("No orchestration settings available."));
});

test("OrchestrationTab validates unsafe recovery windows and saves only a valid dirty draft", async () => {
  const ref = createRef<OrchestrationTabHandle>();
  const onSave = vi.fn(async () => undefined);
  const dirty = vi.fn();
  renderWithProviders(createElement(OrchestrationTab, { ref, settings, loading: false, error: null, onSave, onReset: vi.fn(), onDirtyChange: dirty }));
  await waitFor(() => assert.equal(ref.current?.hasChanges, false));
  fireEvent.change(screen.getByLabelText("Heartbeat Interval"), { target: { value: "60" } });
  await waitFor(() => assert.equal(ref.current?.hasChanges, true));
  await act(async () => { await ref.current?.save(); });
  assert.ok(screen.getByText("Heartbeat interval must be less than stale threshold."));
  assert.equal(onSave.mock.calls.length, 0);

  fireEvent.change(screen.getByLabelText("Heartbeat Interval"), { target: { value: "10" } });
  fireEvent.change(screen.getByLabelText("Max Turns"), { target: { value: "12" } });
  await act(async () => { await ref.current?.save(); });
  await waitFor(() => assert.equal(onSave.mock.calls.length, 1));
  assert.equal(onSave.mock.calls[0]?.[0].runExecution.maxTurns, 12);
  assert.ok(dirty.mock.calls.some(([value]) => value === true));
});

test("OrchestrationTab surfaces save/reset errors and delegates reset", async () => {
  const ref = createRef<OrchestrationTabHandle>();
  const onReset = vi.fn(async () => { throw new Error("reset unavailable"); });
  renderWithProviders(createElement(OrchestrationTab, { ref, settings, loading: false, error: "server warning", onSave: vi.fn(async () => { throw new Error("save unavailable"); }), onReset }));
  fireEvent.change(screen.getByLabelText("Max Turns"), { target: { value: "9" } });
  await act(async () => { await ref.current?.save(); });
  await waitFor(() => assert.ok(screen.getByText("save unavailable")));
  await act(async () => { await ref.current?.reset(); });
  await waitFor(() => assert.ok(screen.getByText("reset unavailable")));
  assert.equal(onReset.mock.calls.length, 1);
});
