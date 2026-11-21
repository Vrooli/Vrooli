import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { test, vi } from "vitest";
import { StatusDialog } from "./StatusDialog.js";
import { HealthStatus } from "../../types.js";
import { renderWithProviders } from "../../test-utils/index.js";

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
    healthError: null,
    health: {
      status: HealthStatus.HEALTHY,
      service: "agent-manager",
      readiness: true,
      version: "1.2.3",
      timestamp: "2026-07-29T00:00:00Z",
      dependencies: { postgres: objectValue({ status: stringValue("ready"), latency_ms: intValue(4n), storage: stringValue("sqlite") }) },
      metrics: { active_runs: intValue(2n) },
    } as never,
  }));

  assert.ok(screen.getByText("Healthy"));
  assert.ok(screen.getByText("Live"));
  assert.ok(screen.getByText("postgres"));
  assert.ok(screen.getByText("4 ms"));
  assert.ok(screen.getByText("active_runs"));
  assert.equal(screen.queryByText(/Agent Manager is (degraded|unreachable)/), null);
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
    } as never,
  }));
  assert.ok(screen.getByText("Degraded"));
  assert.ok(screen.getByText("Connecting"));
  assert.ok(screen.getByText("disk"));
  assert.ok(screen.getByText("Error: locked"));
  assert.ok(screen.getAllByText("—").length > 0);

  rerender(createElement(StatusDialog, { open: true, onOpenChange: vi.fn(), wsStatus: "disconnected", health: undefined, healthError: undefined }));
  assert.ok(screen.getByText("Offline"));
});

test("StatusDialog tells a degraded API (answers, with its reason) from an unreachable one", () => {
  const { rerender } = renderWithProviders(createElement(StatusDialog, {
    open: true, onOpenChange: vi.fn(), wsStatus: "connected", healthError: null,
    healthReason: "startup degraded: in_flight_runs: context deadline exceeded",
    health: { status: HealthStatus.DEGRADED, service: "agent-manager", readiness: false, version: "", timestamp: "", dependencies: {}, metrics: {} } as never,
  }));
  assert.ok(screen.getByText("Degraded"));
  assert.ok(screen.getByText("Agent Manager is degraded"));
  assert.ok(screen.getByText("startup degraded: in_flight_runs: context deadline exceeded"));
  assert.equal(screen.queryByText(/unreachable/i), null);

  rerender(createElement(StatusDialog, { open: true, onOpenChange: vi.fn(), wsStatus: "error", health: null, healthError: "Failed to fetch" }));
  assert.ok(screen.getByText("Unreachable"));
  assert.ok(screen.getByText("Agent Manager is unreachable"));
});
