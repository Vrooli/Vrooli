import "@testing-library/jest-dom";
import { expectNoA11yViolations } from "@vrooli/api-base/testing";
import { screen } from "@testing-library/react";
import { vi } from "vitest";

import App from "./App";
import { renderWithProviders } from "./test-utils/renderWithProviders";

describe("application accessibility", () => {
  beforeEach(() => {
    window.history.replaceState(null, "", "#dashboard");
    localStorage.clear();
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        if (String(input).includes("/health")) {
          return new Response(
            JSON.stringify({ status: "healthy", service: "Scenario To Cloud API" }),
            { status: 200, headers: { "Content-Type": "application/json" } },
          );
        }
        return new Response(JSON.stringify({ error: { message: "not found" } }), {
          status: 404,
          headers: { "Content-Type": "application/json" },
        });
      }),
    );
  });

  it("renders the dashboard without axe violations", async () => {
    renderWithProviders(<App />);

    expect(await screen.findByText("Deploy Scenarios to the Cloud")).toBeInTheDocument();
    await expectNoA11yViolations(document.body);
  });
});
