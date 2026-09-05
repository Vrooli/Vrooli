import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { test, vi } from "vitest";
import { StatusDialog } from "../../src/components/dialogs/StatusDialog.js";
import { HealthStatus } from "../../src/types.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

const stringValue = (value: string) => ({ kind: { case: "stringValue", value } });
const intValue = (value: bigint) => ({ kind: { case: "intValue", value } });
const objectValue = (fields: Record<string, unknown>) => ({ kind: { case: "objectValue", value: { fields } } });

test("StatusDialog renders healthy dependency and metric details and delegates close", async () => {
  const user = userEvent.setup();
  const onOpenChange = vi.fn();
  renderWithProviders(createElement(StatusDialog, {
    open: true,
    onOpenChange,
    wsStatus: "connected",
    healthError: "intermittent probe failure",
    health: {
      status: HealthStatus.HEALTHY,
      service: "agent-manager",
      readiness: true,
      version: "1.2.3",
      timestamp: "2026-07-29T00:00:00Z",
      dependencies: { postgres: objectValue({ status: stringValue("ready"), latency_ms: intValue(4n), storage: stringValue("sqlite") }) },
      metrics: { active_runs: intValue(2n) },
    } as any,
  }));

  assert.ok(screen.getByText("Healthy"));
  assert.ok(screen.getByText("Live"));
  assert.ok(screen.getByText("postgres"));
  assert.ok(screen.getByText("4ms"));
  assert.ok(screen.getByText("active_runs"));
  assert.ok(screen.getByText("intermittent probe failure"));
  await user.keyboard("{Escape}");
  assert.deepEqual(onOpenChange.mock.calls, [[false]]);
});

test("StatusDialog communicates degraded offline state when no health response is available", () => {
  renderWithProviders(createElement(StatusDialog, {
    open: true,
    onOpenChange: vi.fn(),
    wsStatus: "error",
    health: null,
    healthError: null,
  }));
  assert.ok(screen.getAllByText("Unknown").length >= 1);
  assert.ok(screen.getByText("Error"));
  assert.ok(screen.getByText("Not ready"));
});

test("StatusDialog labels connecting and offline websocket states and renders optional dependency data", () => {
  const { rerender } = renderWithProviders(createElement(StatusDialog, {
    open: true, onOpenChange: vi.fn(), wsStatus: "connecting", healthError: null,
    health: {
      status: HealthStatus.DEGRADED, service: "agent-manager", readiness: false, version: "", timestamp: "",
      dependencies: { vault: objectValue({ status: stringValue("slow"), storage: stringValue("disk"), error: stringValue("locked") }) },
      metrics: {},
    } as any,
  }));
  assert.ok(screen.getByText("Degraded"));
  assert.ok(screen.getByText("Connecting"));
  assert.ok(screen.getByText("disk"));
  assert.ok(screen.getByText("Error: locked"));
  assert.ok(screen.getByText("n/a"));

  rerender(createElement(StatusDialog, { open: true, onOpenChange: vi.fn(), wsStatus: "disconnected", health: undefined, healthError: undefined }));
  assert.ok(screen.getByText("Offline"));
});
