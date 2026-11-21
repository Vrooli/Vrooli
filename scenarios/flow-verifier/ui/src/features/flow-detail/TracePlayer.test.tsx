/**
 * TracePlayer unit tests.
 *
 * StateGraph is mocked away — TracePlayer's job is step-management
 * over a trace, so we assert against the trace-player surface, not the
 * graph it composes with.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";

vi.mock("./StateGraph", () => ({
  StateGraph: ({ activeState }: { activeState?: string }) => (
    <div data-testid="state-graph-stub" data-active={activeState ?? ""} />
  ),
}));

import { TracePlayer } from "./TracePlayer";
import type { FlowTrace } from "../../api/inventory";

const states = [
  { id: "draft", quint: "Draft", initial: true },
  { id: "uploading", quint: "Uploading" },
  { id: "uploaded", quint: "Uploaded" },
];

const traces: FlowTrace[] = [
  {
    name: "happy-path",
    initial: "draft",
    steps: [
      { event: "begin", want: "uploading", wantError: false },
      { event: "complete", want: "uploaded", wantError: false },
    ],
  },
  {
    name: "cancel",
    initial: "draft",
    steps: [
      { event: "begin", want: "uploading", wantError: false },
      { event: "cancel", want: "draft", wantError: false },
    ],
  },
];

const graphProps = {
  states,
  events: [{ id: "begin" }, { id: "complete" }, { id: "cancel" }],
  transitions: [],
  initialState: "draft",
};

describe("TracePlayer", () => {
  afterEach(() => cleanup());

  it("renders an empty hint when no traces are available", () => {
    renderWithProviders(
      <TracePlayer traces={[]} graphProps={graphProps} />,
    );
    expect(screen.getByTestId("trace-player-empty")).toBeInTheDocument();
  });

  it("starts on the initial state of the first trace", () => {
    renderWithProviders(<TracePlayer traces={traces} graphProps={graphProps} />);
    expect(screen.getByTestId("trace-player-active-state")).toHaveTextContent("draft");
    expect(screen.getByTestId("state-graph-stub").getAttribute("data-active")).toBe("draft");
    expect(screen.getByTestId("trace-player-prev")).toBeDisabled();
  });

  it("Next walks through steps and disables at the end", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TracePlayer traces={traces} graphProps={graphProps} />);

    await user.click(screen.getByTestId("trace-player-next"));
    expect(screen.getByTestId("trace-player-active-state")).toHaveTextContent("uploading");
    expect(screen.getByTestId("trace-player-last-event")).toHaveTextContent("begin");

    await user.click(screen.getByTestId("trace-player-next"));
    expect(screen.getByTestId("trace-player-active-state")).toHaveTextContent("uploaded");
    expect(screen.getByTestId("trace-player-next")).toBeDisabled();
  });

  it("Reset returns to step 0", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TracePlayer traces={traces} graphProps={graphProps} />);
    await user.click(screen.getByTestId("trace-player-next"));
    await user.click(screen.getByTestId("trace-player-reset"));
    expect(screen.getByTestId("trace-player-active-state")).toHaveTextContent("draft");
    expect(screen.getByTestId("trace-player-prev")).toBeDisabled();
  });

  it("Switching trace resets the step counter and active state", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TracePlayer traces={traces} graphProps={graphProps} />);
    await user.click(screen.getByTestId("trace-player-next"));
    await user.click(screen.getByTestId("trace-player-next"));
    expect(screen.getByTestId("trace-player-active-state")).toHaveTextContent("uploaded");

    await user.selectOptions(screen.getByTestId("trace-player-select"), "1");
    expect(screen.getByTestId("trace-player-active-state")).toHaveTextContent("draft");
    expect(screen.getByTestId("trace-player-prev")).toBeDisabled();
  });
});
