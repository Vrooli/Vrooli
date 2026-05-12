import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { InventoryTable } from "./InventoryTable";
import type { FlowSummary, RunRow } from "../../api/inventory";

const flow: FlowSummary = {
  flowId: "alpha.flow",
  contractPath: "a/flow.json",
  language: "ts",
  schemaVersion: 1,
};

const passedRun: RunRow = {
  id: "run-1",
  flowId: "alpha.flow",
  flowPath: "a/flow.json",
  root: ".",
  mode: "check",
  status: "passed",
  startedAt: "2026-05-12T00:00:00Z",
  finishedAt: "2026-05-12T00:00:01Z",
  durationMs: 1000,
};

describe("InventoryTable", () => {
  afterEach(() => cleanup());

  it("renders rows with status pill and verify button", () => {
    renderWithProviders(
      <InventoryTable
        flows={[flow]}
        latestByFlow={new Map([[flow.flowId, passedRun]])}
        onVerifyOne={() => {}}
        anyPending={false}
      />,
    );
    expect(screen.getByTestId("inventory-table")).toBeInTheDocument();
    expect(screen.getByTestId("inventory-row-alpha.flow")).toBeInTheDocument();
    expect(screen.getByTestId("inventory-link-alpha.flow")).toHaveAttribute(
      "href",
      "/flows/alpha.flow",
    );
    expect(screen.getByTestId("inventory-status-alpha.flow")).toHaveTextContent("passed");
  });

  it("invokes onVerifyOne when the row verify button is clicked", async () => {
    const onVerify = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <InventoryTable
        flows={[flow]}
        latestByFlow={new Map()}
        onVerifyOne={onVerify}
        anyPending={false}
      />,
    );
    await user.click(screen.getByTestId("inventory-verify-alpha.flow"));
    expect(onVerify).toHaveBeenCalledWith(expect.objectContaining({ flowId: "alpha.flow" }));
  });

  it("shows em-dash when no run has happened yet", () => {
    renderWithProviders(
      <InventoryTable
        flows={[flow]}
        latestByFlow={new Map()}
        onVerifyOne={() => {}}
        anyPending={false}
      />,
    );
    expect(screen.getByTestId("inventory-status-alpha.flow")).toHaveTextContent("—");
  });

  it("disables verify buttons when any verification is pending", () => {
    renderWithProviders(
      <InventoryTable
        flows={[flow]}
        latestByFlow={new Map()}
        onVerifyOne={() => {}}
        anyPending
      />,
    );
    expect(screen.getByTestId("inventory-verify-alpha.flow")).toBeDisabled();
  });
});
