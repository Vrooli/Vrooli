import { fireEvent, screen, waitFor } from "../../test-utils";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { vi } from "vitest";
import { StepReady } from "./StepReady";

const readinessApi = vi.hoisted(() => ({ fetchReadiness: vi.fn(), acknowledgeDegraded: vi.fn() }));
vi.mock("../../api/readiness", () => readinessApi);
vi.mock("../../api/capabilities", () => ({ fetchCapabilities: vi.fn().mockResolvedValue({ capabilities: [], count: 0 }) }));
vi.mock("../../api/operatorinputs", () => ({ fetchOperatorInputs: vi.fn().mockResolvedValue({ requests: [] }) }));

const response = (status: string) => ({ status, scenarios: [], resources: [], credentials: [], hosts: [], integrations: [], checked_at: "now", blockers: [], degraded: [] });

describe("StepValidation", () => {
  it("rechecks the composed readiness projection and renders the new status", async () => {
    readinessApi.fetchReadiness.mockResolvedValueOnce(response("missing")).mockResolvedValueOnce(response("ready"));
    renderWithProviders(<StepReady />);
    expect(await screen.findByTestId("readiness-summary")).toHaveTextContent("missing");
    fireEvent.click(screen.getByTestId("recheck"));
    await waitFor(() => expect(screen.getByTestId("readiness-summary")).toHaveTextContent("Ready"));
    expect(readinessApi.fetchReadiness).toHaveBeenCalledTimes(2);
  });
});
