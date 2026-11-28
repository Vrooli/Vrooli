import { beforeEach, vi } from "vitest";
import { screen } from "../../test-utils";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { StepCredentials } from "./StepCredentials";

const readinessApi = vi.hoisted(() => ({ fetchReadiness: vi.fn() }));
const credentialsApi = vi.hoisted(() => ({ fetchCredentials: vi.fn() }));
const capabilitiesApi = vi.hoisted(() => ({ fetchCapabilities: vi.fn() }));
const operatorInputsApi = vi.hoisted(() => ({ fetchOperatorInputs: vi.fn() }));
const authApi = vi.hoisted(() => ({ loginAuthenticator: vi.fn() }));
const applyApi = vi.hoisted(() => ({ fetchApplyPlan: vi.fn() }));

vi.mock("../../api/readiness", () => readinessApi);
vi.mock("../../api/credentials", () => credentialsApi);
vi.mock("../../api/capabilities", () => capabilitiesApi);
vi.mock("../../api/operatorinputs", () => operatorInputsApi);
vi.mock("../../api/auth", () => authApi);
vi.mock("../../api/apply", () => applyApi);

describe("credential scope summary", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    readinessApi.fetchReadiness.mockResolvedValue({
      status: "missing", scenarios: [], resources: [], credentials: [], hosts: [], integrations: [],
      checked_at: "2026-11-22T00:00:00Z", blockers: [], degraded: [],
    });
    credentialsApi.fetchCredentials.mockResolvedValue({
      count: 3,
      credentials: [
        { resource: "mail", logical_id: "mail", field: "api_key", label: "Mail key", required: true, status: "pending" },
        { resource: "cache", logical_id: "cache", field: "token", label: "Cache token", required: true, status: "configured" },
        { resource: "telemetry", logical_id: "telemetry", field: "endpoint", label: "Telemetry endpoint", required: false, status: "deferred" },
      ],
    });
  });

  it("explains closure scope and distinguishes required attention from optional deferred inputs", async () => {
    renderWithProviders(<StepCredentials />);

    expect(await screen.findByTestId("credential-scope-summary")).toHaveTextContent(
      "Only inputs in this setup's selected and system-required capability closure, plus project-level inputs, are listed: 3 total; 2 required (1 need attention); 1 optional (1 deferred until needed).",
    );
    expect(screen.getByTestId("credential-outstanding-count")).toHaveTextContent("1");
    expect(screen.getByText("Required needing attention")).toBeInTheDocument();
  });
});
