import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi } from "vitest";
import { screen, fireEvent, within } from "@testing-library/react";

import MachinePicker from "../components/launcher/MachinePicker";
import type { TerminalTarget } from "../api/targets";

// [REQ:P0-014b] Compact Machine Selection
//
// These cover the behaviour the full-width TargetCard stack used to carry:
// selection, readiness detail, the grant chip, and keyboard navigation — now
// in one row instead of one card per machine.

const local: TerminalTarget = {
  id: "local",
  kind: "local",
  label: "This machine",
  available: true,
  state: "dispatchable",
  readiness: [{ key: "local", label: "Web Console process", passed: true, detail: "Available on this machine" }],
};

function remote(overrides: Partial<TerminalTarget> & { id: string; label: string }): TerminalTarget {
  return {
    kind: "bridge-node",
    available: true,
    state: "dispatchable",
    ...overrides,
  } as TerminalTarget;
}

describe("MachinePicker", () => {
  it("collapses machine selection into one trigger row", () => {
    render(<MachinePicker targets={[local]} selectedId="local" onSelect={vi.fn()} />);
    expect(screen.getByTestId("launcher-machine-picker")).toHaveTextContent("This machine");
    // The menu is closed until asked for: the row is the whole resting state.
    expect(screen.queryByTestId("launcher-machine-list")).not.toBeInTheDocument();
  });

  it("opens a listbox naming every machine", () => {
    const targets = [local, remote({ id: "node-1", label: "Build node", os: "linux", arch: "amd64" })];
    render(<MachinePicker targets={targets} selectedId="local" onSelect={vi.fn()} />);

    fireEvent.click(screen.getByTestId("launcher-machine-picker"));
    const list = screen.getByTestId("launcher-machine-list");
    expect(list).toHaveAttribute("role", "listbox");
    expect(within(list).getAllByRole("option")).toHaveLength(2);
  });

  it("selects a machine and closes", () => {
    const onSelect = vi.fn();
    const targets = [local, remote({ id: "node-1", label: "Build node" })];
    render(<MachinePicker targets={targets} selectedId="local" onSelect={onSelect} />);

    fireEvent.click(screen.getByTestId("launcher-machine-picker"));
    fireEvent.click(screen.getByTestId("launcher-machine-option-node-1"));
    expect(onSelect).toHaveBeenCalledWith("node-1");
    expect(screen.queryByTestId("launcher-machine-list")).not.toBeInTheDocument();
  });

  // The meta line is what replaced the standalone readiness grid: a failing
  // fact has to stay visible without a second surface.
  it("names the failing readiness fact on the row", () => {
    const targets = [
      local,
      remote({
        id: "node-1",
        label: "Sad node",
        available: false,
        state: "needs-update",
        os: "linux",
        arch: "amd64",
        readiness: [{ key: "agent_version", label: "Bridge agent out of date", passed: false, detail: "1.0 < 2.0" }],
      }),
    ];
    render(<MachinePicker targets={targets} selectedId="local" onSelect={vi.fn()} />);

    fireEvent.click(screen.getByTestId("launcher-machine-picker"));
    expect(screen.getByTestId("launcher-machine-option-node-1")).toHaveTextContent("Bridge agent out of date");
  });

  it("shows the concrete grant as a chip on an unselected row", () => {
    const targets = [
      local,
      remote({
        id: "node-1",
        label: "Mac mini",
        readiness: [{ key: "bridge_scope", label: "Bridge scope", passed: true, detail: "Read only; system-monitor:read" }],
      }),
    ];
    render(<MachinePicker targets={targets} selectedId="local" onSelect={vi.fn()} />);

    fireEvent.click(screen.getByTestId("launcher-machine-picker"));
    expect(screen.getByTestId("launcher-machine-option-node-1")).toHaveTextContent("Read only; system-monitor:read");
  });

  it("moves the active row with the arrow keys", () => {
    const targets = [local, remote({ id: "node-1", label: "Build node" })];
    render(<MachinePicker targets={targets} selectedId="local" onSelect={vi.fn()} />);

    fireEvent.click(screen.getByTestId("launcher-machine-picker"));
    fireEvent.keyDown(screen.getByTestId("launcher-machine-list"), { key: "ArrowDown" });
    expect(screen.getByTestId("launcher-machine-option-node-1")).toHaveAttribute("data-active", "true");
  });

  // Losing this leaves a keyboard reader with focus on nothing.
  it("returns focus to the trigger when Escape closes the menu", () => {
    render(<MachinePicker targets={[local]} selectedId="local" onSelect={vi.fn()} />);
    const trigger = screen.getByTestId("launcher-machine-picker");

    fireEvent.click(trigger);
    fireEvent.keyDown(document, { key: "Escape" });
    expect(screen.queryByTestId("launcher-machine-list")).not.toBeInTheDocument();
    expect(trigger).toHaveFocus();
  });

  // A button inside the listbox breaks the relationship assistive tech relies
  // on, and the list scrolls while this action must not.
  it("keeps the link action outside the listbox element", () => {
    const onOpenMachines = vi.fn();
    render(<MachinePicker targets={[local]} selectedId="local" onSelect={vi.fn()} onOpenMachines={onOpenMachines} />);

    fireEvent.click(screen.getByTestId("launcher-machine-picker"));
    const link = screen.getByTestId("launcher-machine-link");
    const list = screen.getByTestId("launcher-machine-list");
    expect(list.contains(link)).toBe(false);

    fireEvent.click(link);
    expect(onOpenMachines).toHaveBeenCalledOnce();
  });

  it("renders no link action when the caller cannot open the machines surface", () => {
    render(<MachinePicker targets={[local]} selectedId="local" onSelect={vi.fn()} />);
    fireEvent.click(screen.getByTestId("launcher-machine-picker"));
    expect(screen.queryByTestId("launcher-machine-link")).not.toBeInTheDocument();
  });
});
