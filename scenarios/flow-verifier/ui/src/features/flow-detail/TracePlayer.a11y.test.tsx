import { afterEach, beforeEach, describe, it, vi } from "vitest";
import { cleanup } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";

vi.mock("./StateGraph", () => ({
  StateGraph: () => <div role="img" aria-label="state-graph-stub" />,
}));

import { TracePlayer } from "./TracePlayer";

describe("TracePlayer accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = renderWithProviders(
      <TracePlayer
        traces={[
          {
            name: "happy",
            initial: "a",
            steps: [{ event: "go", want: "b", wantError: false }],
          },
        ]}
        graphProps={{
          states: [
            { id: "a", quint: "A", initial: true },
            { id: "b", quint: "B" },
          ],
          events: [{ id: "go" }],
          transitions: [{ from: "a", event: "go", to: "b", wantError: false }],
          initialState: "a",
        }}
      />,
    );
    await expectNoA11yViolations(container);
  });
});
