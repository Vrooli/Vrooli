import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";

import { CounterexampleDiff } from "./CounterexampleDiff";
import type { FlowTransition } from "../../api/inventory";

const expected: FlowTransition[] = [
  { from: "draft", event: "begin", to: "uploading", wantError: false },
  { from: "uploading", event: "complete", to: "uploaded", wantError: false },
];

describe("CounterexampleDiff", () => {
  afterEach(() => cleanup());

  it("shows an empty hint when no counterexample is present", () => {
    renderWithProviders(
      <CounterexampleDiff counterexampleJson="" expectedTransitions={expected} />,
    );
    expect(screen.getByTestId("ce-diff-empty")).toBeInTheDocument();
  });

  it("surfaces a parse error rather than swallowing bad JSON", () => {
    renderWithProviders(
      <CounterexampleDiff
        counterexampleJson="{not json}"
        expectedTransitions={expected}
      />,
    );
    expect(screen.getByTestId("ce-diff-parse-error")).toBeInTheDocument();
  });

  it("renders one row per transition with expected vs actual", () => {
    const ce = JSON.stringify({
      states: [
        { state: "draft" },
        { state: "uploading", event: "begin" },
        { state: "uploaded", event: "complete" },
      ],
    });
    renderWithProviders(
      <CounterexampleDiff counterexampleJson={ce} expectedTransitions={expected} />,
    );
    expect(screen.getByTestId("ce-diff-row-1")).toBeInTheDocument();
    expect(screen.getByTestId("ce-diff-row-2")).toBeInTheDocument();
    expect(screen.getByTestId("ce-diff-actual-1")).toHaveTextContent("uploading");
    expect(screen.getByTestId("ce-diff-actual-2")).toHaveTextContent("uploaded");
  });

  it("flags rows where the actual state diverges from the model", () => {
    const ce = JSON.stringify({
      states: [
        { state: "draft" },
        // model expects "uploading" for begin from "draft"
        { state: "uploaded", event: "begin" },
      ],
    });
    renderWithProviders(
      <CounterexampleDiff counterexampleJson={ce} expectedTransitions={expected} />,
    );
    expect(screen.getByTestId("ce-diff-actual-1")).toHaveClass("text-app-danger");
  });
});
