import { screen } from "../../test-utils";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { vi } from "vitest";
import { StepCredentials } from "./StepCredentials";

const readinessApi = vi.hoisted(() => ({ fetchReadiness: vi.fn() }));
const capabilitiesApi = vi.hoisted(() => ({ fetchCapabilities: vi.fn() }));
const operatorInputsApi = vi.hoisted(() => ({ fetchOperatorInputs: vi.fn() }));
vi.mock("../../api/readiness", () => readinessApi);
vi.mock("../../api/capabilities", () => capabilitiesApi);
vi.mock("../../api/operatorinputs", () => operatorInputsApi);

describe("StepCredentials", () => {
  it("renders the owner purpose and declared acquisition link", async () => {
    readinessApi.fetchReadiness.mockResolvedValue({
      status: "missing",
      scenarios: [],
      resources: [],
      credentials: [{ logical_id: "mail", field: "api_key", label: "Mail API key", description: "Sends sign-in mail", obtain_url: "https://provider.example/keys", required: true, status: "unconfigured" }],
      hosts: [],
      integrations: [],
      checked_at: "now",
      blockers: [],
      degraded: [],
    });
    capabilitiesApi.fetchCapabilities.mockResolvedValue({ capabilities: [], count: 0 });
    operatorInputsApi.fetchOperatorInputs.mockResolvedValue({ requests: [] });
    renderWithProviders(<StepCredentials />);
    expect(await screen.findByText("Mail API key")).toBeInTheDocument();
    expect(screen.getByText("Sends sign-in mail")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "How to obtain this credential" })).toHaveAttribute("href", "https://provider.example/keys");
  });
});
