import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import type { FlowDetail } from "../../api/inventory";
import { OverviewTab } from "./OverviewTab";

const baseDetail: FlowDetail = {
  flowId: "demo.flow.api",
  kind: "temporal",
  domain: "demo",
  description: "A small example flow.",
  contractPath: "api/demo/flow/flow.json",
  language: "go",
  schemaVersion: 6,
  initialState: "a",
  states: [{ id: "a", quint: "A", initial: true }],
  events: [],
  transitions: [],
  traces: [{ name: "happy", initial: "a", steps: [] }],
  invariants: [{ id: "inv1", quint: "Inv1", description: "always true" }],
  model: {
    module: "Demo",
    seed: "2026-05-12",
    maxSteps: 4,
    traceCount: 1,
    verify: { invariants: ["Inv1"] },
  },
  runtime: { go: { package: "demo", statusType: "S", eventType: "E", constantPrefix: "Demo" } },
  report: "",
};

describe("OverviewTab", () => {
  afterEach(() => cleanup());

  it("renders description, metadata, model, invariants, and traces", () => {
    renderWithProviders(<OverviewTab detail={baseDetail} />);
    expect(screen.getByTestId("flow-overview")).toBeInTheDocument();
    expect(screen.getByTestId("flow-overview-description")).toHaveTextContent(
      "A small example flow.",
    );
    expect(screen.getByTestId("flow-overview-model")).toHaveTextContent("Demo");
    expect(screen.getByTestId("flow-overview-invariant-inv1")).toHaveTextContent("always true");
    expect(screen.getByTestId("flow-overview-trace-happy")).toBeInTheDocument();
    expect(screen.getByTestId("flow-overview-runtime")).toHaveTextContent(/demo/i);
  });

  it("falls back when description is empty", () => {
    renderWithProviders(<OverviewTab detail={{ ...baseDetail, description: undefined }} />);
    expect(screen.getByTestId("flow-overview-description")).toHaveTextContent(
      "flowOverview.noDescription",
    );
  });
});
