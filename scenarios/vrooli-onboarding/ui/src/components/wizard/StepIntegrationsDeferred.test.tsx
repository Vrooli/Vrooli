import { fireEvent, renderWithProviders, screen, waitFor } from "../../test-utils";
import { vi } from "vitest";
import { StepIntegrationsDeferred } from "./StepIntegrationsDeferred";

const readinessApi = vi.hoisted(() => ({ fetchReadiness: vi.fn() }));
vi.mock("../../api/readiness", () => readinessApi);

const ready = {
  status: "ready",
  scenarios: [],
  resources: [],
  credentials: [],
  hosts: [],
  integrations: [{ name: "alpha/github-oauth", category: "integration", required: true, detail: "Read project issues" }],
  checked_at: "now",
  blockers: [],
  degraded: [],
};

describe("StepIntegrationsDeferred readiness recovery", () => {
  beforeEach(() => readinessApi.fetchReadiness.mockReset());

  it("shows loading without claiming there are no declared integrations", async () => {
    readinessApi.fetchReadiness.mockImplementationOnce(() => new Promise(() => undefined));
    renderWithProviders(<StepIntegrationsDeferred />);

    expect(screen.getByTestId("integrations-loading")).toHaveTextContent("Checking readiness");
    expect(screen.getByTestId("integrations-deferred-notice")).toHaveTextContent("Integration setup is deferred");
    expect(screen.queryByTestId("declared-integrations")).not.toBeInTheDocument();
    expect(screen.queryByText("No selected capability declares an external integration requirement.")).not.toBeInTheDocument();
  });

  it("keeps the failure visible and retries the same readiness request", async () => {
    readinessApi.fetchReadiness.mockRejectedValueOnce(new Error("control plane unavailable"))
      .mockResolvedValueOnce(ready);
    renderWithProviders(<StepIntegrationsDeferred target="node-1" />);

    expect(await screen.findByTestId("integrations-load-error")).toHaveTextContent("Readiness could not be checked");
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));

    expect(await screen.findByTestId("declared-integrations")).toHaveTextContent("alpha/github-oauth");
    await waitFor(() => expect(readinessApi.fetchReadiness).toHaveBeenCalledTimes(2));
    expect(readinessApi.fetchReadiness).toHaveBeenNthCalledWith(1, "node-1");
    expect(readinessApi.fetchReadiness).toHaveBeenNthCalledWith(2, "node-1");
  });
});
