import { afterEach, beforeEach, describe, it, vi } from "vitest";
import { cleanup } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";

vi.mock("@xyflow/react/dist/style.css", () => ({}));
vi.mock("@xyflow/react", () => ({
  Background: () => null,
  Position: { Left: "left", Right: "right", Top: "top", Bottom: "bottom" },
  ReactFlow: () => <div role="img" aria-label="state-graph-stub" />,
}));

import { StateGraph } from "./StateGraph";

describe("StateGraph accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = renderWithProviders(
      <StateGraph
        states={[
          { id: "a", quint: "A", initial: true },
          { id: "b", quint: "B" },
        ]}
        events={[{ id: "go" }]}
        transitions={[{ from: "a", event: "go", to: "b", wantError: false }]}
        initialState="a"
      />,
    );
    await expectNoA11yViolations(container);
  });
});
