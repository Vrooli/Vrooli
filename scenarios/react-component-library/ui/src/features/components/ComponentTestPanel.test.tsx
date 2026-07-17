import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { ComponentTestPanel } from "./ComponentTestPanel";
import { renderWithProviders } from "../../test-utils/renderWithProviders";

const api = vi.hoisted(() => ({
	getComponentTestReport: vi.fn(),
  listComponentTestReports: vi.fn(),
  runComponentTest: vi.fn(),
}));
vi.mock("../../api/componentTests", () => api);

describe("ComponentTestPanel", () => {
  it("launches the explicit version closure and renders durable remediation", async () => {
    api.listComponentTestReports.mockResolvedValue([]);
    api.runComponentTest.mockResolvedValue({ id: "ctr_123", verdict: "failed", results: [{ stage: "contract_validation", assetLibraryId: "rcl:Button", version: "1.0.0", verdict: "failed", message: "invalid", remediation: "fix contract" }] });
    renderWithProviders(<ComponentTestPanel componentId="button-id" version="1.0.0" />);
    await screen.findByText("No component test history");
    fireEvent.click(screen.getByRole("button", { name: "Run tests" }));
    await waitFor(() => expect(api.runComponentTest).toHaveBeenCalledWith({ componentId: "button-id", version: "1.0.0", includeClosure: true }));
    expect(await screen.findByText(/Next: fix contract/)).toBeInTheDocument();
    expect(screen.getAllByText("failed")).toHaveLength(2);
  });

  it("provides a deep link for historical durable evidence", async () => {
    api.listComponentTestReports.mockResolvedValue([
      { id: "ctr_latest", verdict: "passed", results: [] },
      { id: "ctr_history", verdict: "failed", results: [] },
    ]);
    renderWithProviders(<ComponentTestPanel componentId="button-id" version="1.0.0" />);
    const summary = await screen.findByText("1 earlier run(s)");
    fireEvent.click(summary);
    expect(screen.getByRole("link", { name: "Open component test report ctr_history" })).toHaveAttribute("href", "?tab=tests&testReport=ctr_history");
  });
});
