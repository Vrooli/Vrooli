import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { installMatchMediaMock } from "../../test-utils/browser";
import { OperationProvenancePopover } from "./operation-provenance-popover";
import type { OperationProvenanceData } from "../../lib/agent-ops-utils";

installMatchMediaMock();

const fullData: OperationProvenanceData = {
  source: "canonical",
  operation: "workshop-round",
  operationVersion: "1.2.0",
  executionId: "exec-7",
  runId: "run-42",
  mode: "workshop-loop",
  modeRevision: "sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
  bindingLayer: "initiative-override",
  bindingOwnerKind: "initiative",
  bindingOwnerId: "init-a",
  recordedAt: "2026-07-14T12:00:00Z",
  attempt: 3,
  priorExecutionId: "exec-6",
  state: "completed",
  outcome: "success",
  snapshotFound: true,
  provenanceDigest: "sha256:1111222233334444aaaabbbbccccdddd1111222233334444aaaabbbbccccdddd",
  compiledModeDigest: "sha256:5555666677778888aaaabbbbccccdddd5555666677778888aaaabbbbccccdddd",
  promptCatalogDigest: "sha256:9999aaaabbbbcccc9999aaaabbbbcccc9999aaaabbbbcccc9999aaaabbbbcccc",
  callerInputDigest: "sha256:eeeeffff0000111122223333444455556666777788889999aaaabbbbccccdddd",
  reproducible: true,
};

describe("OperationProvenancePopover", () => {
  it("renders a canonical source badge as the trigger", () => {
    render(<OperationProvenancePopover data={fullData} />);
    const trigger = screen.getByTestId("operation-provenance-trigger");
    expect(trigger).toHaveTextContent("canonical");
    expect(screen.queryByTestId("operation-provenance-popover")).not.toBeInTheDocument();
  });

  it("shows every provenance field when opened", async () => {
    const user = userEvent.setup();
    render(<OperationProvenancePopover data={fullData} />);
    await user.click(screen.getByTestId("operation-provenance-trigger"));

    const popover = screen.getByTestId("operation-provenance-popover");
    // Contract identity (operation id @ pinned version).
    expect(popover).toHaveTextContent("workshop-round@1.2.0");
    // Mode id + exact revision (digest prefix).
    expect(popover).toHaveTextContent("workshop-loop @ sha256:abcdef012345…");
    // Binding source: layer + owner.
    expect(popover).toHaveTextContent("Initiative override (initiative init-a)");
    // Run / execution ids.
    expect(popover).toHaveTextContent("exec-7");
    expect(popover).toHaveTextContent("run-42");
    // Attempt / prior-execution retry linkage.
    expect(popover).toHaveTextContent("3 (retry of exec-6)");
    // State / outcome.
    expect(popover).toHaveTextContent("completed");
    expect(popover).toHaveTextContent("success");
    // Digests (shortened).
    expect(popover).toHaveTextContent("sha256:111122223333…");
    expect(popover).toHaveTextContent("sha256:555566667777…");
    expect(popover).toHaveTextContent("sha256:9999aaaabbbb…");
    expect(popover).toHaveTextContent("sha256:eeeeffff0000…");
    // Verified evidence flag.
    expect(screen.getByTestId("operation-provenance-reproducible")).toHaveTextContent(
      "Verified evidence",
    );
  });

  it("flags digest drift when reproducible=false", async () => {
    const user = userEvent.setup();
    render(<OperationProvenancePopover data={{ ...fullData, reproducible: false }} />);
    await user.click(screen.getByTestId("operation-provenance-trigger"));
    expect(screen.getByTestId("operation-provenance-reproducible")).toHaveTextContent(
      "Digest drift",
    );
  });

  it("warns when the execution snapshot is missing", async () => {
    const user = userEvent.setup();
    render(
      <OperationProvenancePopover
        data={{ ...fullData, snapshotFound: false, reproducible: undefined }}
      />,
    );
    await user.click(screen.getByTestId("operation-provenance-trigger"));
    expect(screen.getByTestId("operation-provenance-popover")).toHaveTextContent(
      "Execution snapshot missing on disk",
    );
    expect(screen.queryByTestId("operation-provenance-reproducible")).not.toBeInTheDocument();
  });

  it("is reachable by role and accessible name, opens with the keyboard, closes on Escape", async () => {
    const user = userEvent.setup();
    render(<OperationProvenancePopover data={fullData} />);

    const trigger = screen.getByRole("button", { name: "Operation provenance" });
    trigger.focus();
    expect(trigger).toHaveFocus();

    await user.keyboard("{Enter}");
    expect(screen.getByTestId("operation-provenance-popover")).toBeInTheDocument();

    await user.keyboard("{Escape}");
    expect(screen.queryByTestId("operation-provenance-popover")).not.toBeInTheDocument();
  });

  it("labels the binding as system default without an owner suffix", async () => {
    const user = userEvent.setup();
    render(
      <OperationProvenancePopover
        data={{
          ...fullData,
          bindingLayer: "system-default",
          bindingOwnerKind: "system",
          bindingOwnerId: "",
        }}
      />,
    );
    await user.click(screen.getByTestId("operation-provenance-trigger"));
    expect(screen.getByTestId("operation-provenance-popover")).toHaveTextContent("System default");
  });
});
