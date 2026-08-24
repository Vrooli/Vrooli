import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { ComponentTestPanel } from "./ComponentTestPanel";
import { renderWithProviders } from "@vrooli/api-base/testing";

const api = vi.hoisted(() => ({
  getComponentTestReport: vi.fn(),
  listComponentTestReports: vi.fn(),
  runComponentTest: vi.fn(),
}));
vi.mock("../../api/componentTests", () => api);

describe("ComponentTestPanel", () => {
  it("defaults to the selected version and renders durable remediation", async () => {
    api.listComponentTestReports.mockResolvedValue([]);
    api.runComponentTest.mockResolvedValue({
      id: "ctr_123",
      verdict: "failed",
      results: [
        {
          stage: "contract_validation",
          assetLibraryId: "rcl:Button",
          version: "1.0.0",
          verdict: "failed",
          message: "invalid",
          remediation: "fix contract",
        },
      ],
    });
    renderWithProviders(<ComponentTestPanel componentId="button-id" version="1.0.0" />);
    await screen.findByText("No component test evidence yet");
    expect(screen.getByRole("checkbox", { name: "Include dependency closure" })).not.toBeChecked();
    fireEvent.click(screen.getByRole("button", { name: "Run component tests" }));
    await waitFor(() =>
      expect(api.runComponentTest).toHaveBeenCalledWith({
        componentId: "button-id",
        version: "1.0.0",
        includeClosure: false,
      }),
    );
    expect(await screen.findAllByText("Recommended next step:")).toHaveLength(2);
    expect(screen.getAllByText("fix contract")).toHaveLength(2);
    expect(screen.getAllByText("Needs attention")).toHaveLength(3);
  });

  it("provides a deep link for historical durable evidence", async () => {
    api.listComponentTestReports.mockResolvedValue([
      { id: "ctr_latest", verdict: "passed", results: [] },
      { id: "ctr_history", verdict: "failed", results: [] },
    ]);
    renderWithProviders(<ComponentTestPanel componentId="button-id" version="1.0.0" />, {
      routerEntries: ["/assets/button-id?tab=tests"],
    });
    expect(
      await screen.findByRole("link", { name: "Open component test report ctr_history" }),
    ).toHaveAttribute("href", "/assets/button-id?tab=tests&testReport=ctr_history");
  });

  it("shows a structural loading skeleton while history is being retrieved", () => {
    api.listComponentTestReports.mockReturnValue(new Promise(() => undefined));
    renderWithProviders(<ComponentTestPanel componentId="button-id" version="1.0.0" />);
    expect(screen.getByTestId("component-test-history-skeleton")).toBeInTheDocument();
  });

  it("explains when history cannot be retrieved instead of presenting an empty history", async () => {
    api.listComponentTestReports.mockRejectedValue(new Error("legacy data could not be decoded"));
    renderWithProviders(<ComponentTestPanel componentId="button-id" version="1.0.0" />);
    expect(await screen.findByRole("alert")).toHaveTextContent("Test history could not be loaded");
    expect(screen.queryByText("No component test evidence yet")).not.toBeInTheDocument();
  });

  it("surfaces BAS artifacts as readable evidence states", async () => {
    api.listComponentTestReports.mockResolvedValue([
      {
        id: "ctr_capture",
        verdict: "passed",
        results: [],
        artifacts: [
          {
            kind: "bas-screenshot",
            label: "story:screenshot",
            assetLibraryId: "rcl:Button",
            version: "1.0.0",
            reference: "http://127.0.0.1:17116/api/v1/artifacts/capture.png",
          },
        ],
      },
    ]);
    renderWithProviders(<ComponentTestPanel componentId="button-id" version="1.0.0" />);
    expect(await screen.findByRole("tab", { name: "Screenshot: Captured" })).toBeInTheDocument();
    expect(
      screen.getByRole("tab", { name: "Accessibility Tree: Not captured" }),
    ).toBeInTheDocument();
    expect(screen.queryByText("screenshotmissing")).not.toBeInTheDocument();
    expect(
      screen.getByRole("img", { name: "Captured component screenshot" }).getAttribute("src"),
    ).toContain("/embedded/browser-automation-studio/api/v1/artifacts/capture.png");
    expect(screen.queryByRole("link", { name: /open.*capture/i })).not.toBeInTheDocument();
  });
});
