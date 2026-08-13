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
  it("launches the explicit version closure and renders durable remediation", async () => {
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
    fireEvent.click(screen.getByRole("button", { name: "Run component tests" }));
    await waitFor(() =>
      expect(api.runComponentTest).toHaveBeenCalledWith({
        componentId: "button-id",
        version: "1.0.0",
        includeClosure: true,
      }),
    );
    expect(await screen.findByText("Recommended next step:")).toBeInTheDocument();
    expect(screen.getByText("fix contract")).toBeInTheDocument();
    expect(screen.getAllByText("Needs attention")).toHaveLength(2);
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
});
