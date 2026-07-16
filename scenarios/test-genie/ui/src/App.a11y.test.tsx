/**
 * App-shell accessibility regression coverage.
 *
 * The dashboard is the initial Test Genie route and includes the application
 * navigation and main landmark. Mocking its data hooks keeps this test focused
 * on the shell's rendered accessibility semantics rather than network state.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render } from "@testing-library/react";

vi.mock("./hooks/useExecutions", () => ({
  useExecutions: () => ({ executions: [], lastFailedExecution: null }),
}));

vi.mock("./hooks/useScenarios", () => ({
  useScenarios: () => ({
    scenarioDirectoryEntries: [],
    catalogStats: { tracked: 0, failing: 0 },
  }),
}));

import App from "./App";
import { expectNoA11yViolations } from "./test-utils/a11y";
import { useUIStore } from "./stores/uiStore";

describe("Test Genie app shell accessibility", () => {
  afterEach(() => {
    cleanup();
    useUIStore.setState({ activeTab: "dashboard" });
  });

  it("renders the initial application shell without axe violations", async () => {
    useUIStore.setState({ activeTab: "dashboard" });
    const { container } = render(<App />);

    await expectNoA11yViolations(container);
    expect(container.querySelector("main")).toBeInTheDocument();
  });
});
