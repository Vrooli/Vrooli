import { fireEvent, screen, waitFor } from "../../test-utils";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { Code, ConnectError } from "@connectrpc/connect";
import { vi } from "vitest";
import { StepCredentials } from "./StepCredentials";

const readinessApi = vi.hoisted(() => ({ fetchReadiness: vi.fn() }));
const credentialsApi = vi.hoisted(() => ({ fetchCredentials: vi.fn(), provisionCredential: vi.fn() }));
const capabilitiesApi = vi.hoisted(() => ({ fetchCapabilities: vi.fn() }));
const operatorInputsApi = vi.hoisted(() => ({ fetchOperatorInputs: vi.fn() }));
const authApi = vi.hoisted(() => ({ loginAuthenticator: vi.fn() }));
vi.mock("../../api/readiness", () => readinessApi);
vi.mock("../../api/credentials", () => credentialsApi);
vi.mock("../../api/capabilities", () => capabilitiesApi);
vi.mock("../../api/operatorinputs", () => operatorInputsApi);
vi.mock("../../api/auth", () => authApi);

describe("StepCredentials", () => {
  it("renders the owner purpose and declared acquisition link", async () => {
    readinessApi.fetchReadiness.mockResolvedValue({
      status: "ready",
      scenarios: [],
      resources: [],
      credentials: [{ logical_id: "mail", field: "api_key", label: "Mail API key", description: "Sends sign-in mail", obtain_url: "https://provider.example/keys", required: true, status: "configured" }],
      hosts: [],
      integrations: [],
      checked_at: "now",
      blockers: [],
      degraded: [],
    });
    capabilitiesApi.fetchCapabilities.mockResolvedValue({ capabilities: [], count: 0 });
    operatorInputsApi.fetchOperatorInputs.mockResolvedValue({ requests: [] });
    credentialsApi.fetchCredentials.mockResolvedValue({ credentials: [{ resource: "mail", logical_id: "mail", field: "api_key", label: "Mail API key", description: "Sends sign-in mail", obtain_url: "https://provider.example/keys", required: true, status: "pending" }], count: 1 });
    renderWithProviders(<StepCredentials />);
    expect(await screen.findByText("Mail API key")).toBeInTheDocument();
    expect(screen.getByText("Sends sign-in mail")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "How to obtain this credential" })).toHaveAttribute("href", "https://provider.example/keys");
    expect(screen.getByTestId("credential-status")).toHaveTextContent("Stored securely");
    expect(screen.getByRole("searchbox", { name: "Search credential inputs" })).toBeInTheDocument();
    expect(screen.getByTestId("credential-filter")).toBeInTheDocument();
    expect(screen.getByTestId("credential-sort")).toBeInTheDocument();
  });

  it("renders the credential rows from readiness while the richer inventory is still loading", async () => {
    readinessApi.fetchReadiness.mockResolvedValue({
      status: "degraded",
      scenarios: [],
      resources: [],
      credentials: [{ logical_id: "mail", field: "api_key", label: "Mail API key", description: "Sends sign-in mail", obtain_url: "https://provider.example/keys", required: true, status: "pending", evidence_status: "pending" }],
      hosts: [],
      integrations: [],
      checked_at: "now",
      blockers: [],
      degraded: [],
    });
    credentialsApi.fetchCredentials.mockReturnValue(new Promise(() => undefined));
    capabilitiesApi.fetchCapabilities.mockResolvedValue({ capabilities: [] });
    operatorInputsApi.fetchOperatorInputs.mockResolvedValue({ requests: [] });

    renderWithProviders(<StepCredentials />);

    expect(await screen.findByTestId("credential-card")).toHaveTextContent("Mail API key");
    expect(screen.getByTestId("credential-entry-group")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save securely" })).toBeInTheDocument();
    expect(screen.queryByRole("status", { name: "Checking credential requirements…" })).not.toBeInTheDocument();
  });

  it("explains operator authentication failures and offers recovery actions", async () => {
    readinessApi.fetchReadiness.mockResolvedValue({
      status: "degraded",
      scenarios: [],
      resources: [],
      credentials: [{ logical_id: "mail", field: "api_key", label: "Mail API key", description: "Sends sign-in mail", required: true, status: "pending" }],
      hosts: [],
      integrations: [],
      checked_at: "now",
      blockers: [],
      degraded: [],
    });
    credentialsApi.fetchCredentials.mockResolvedValue({ credentials: [{ resource: "mail", logical_id: "mail", field: "api_key", label: "Mail API key", description: "Sends sign-in mail", required: true, status: "pending" }], count: 1 });
    const authError = new ConnectError("verified onboarding operator required", Code.Unauthenticated);
    authError.metadata.set("Vrooli-Auth-Recovery-Url", "https://auth.example.test/auth/login?return_to=%2Fsetup%2Fcredentials");
    // A hybrid deployment may report Cloudflare as the first failed provider
    // while Scenario Authenticator is also configured. The recovery surface
    // must choose the working embedded authenticator flow in that case.
    authError.metadata.set("Vrooli-Auth-Source", "cloudflare_access");
    authError.metadata.set("Vrooli-Auth-Providers", "cloudflare_access,scenario_authenticator");
    credentialsApi.provisionCredential.mockRejectedValueOnce(authError);
    authApi.loginAuthenticator.mockResolvedValueOnce({});
    capabilitiesApi.fetchCapabilities.mockResolvedValue({ capabilities: [] });
    operatorInputsApi.fetchOperatorInputs.mockResolvedValue({ requests: [] });

    renderWithProviders(<StepCredentials />);

    fireEvent.change(await screen.findByTestId("credential-input"), { target: { value: "secret" } });
    fireEvent.click(screen.getByTestId("credential-save"));

    expect(await screen.findByText("Sign in to save this credential")).toBeInTheDocument();
    expect(screen.getByTestId("credential-recovery-dialog")).toBeInTheDocument();
    expect(screen.getByText("Sign in through Scenario Authenticator below. Your credential stays in this form and is retried only after the sign-in succeeds.")).toBeInTheDocument();
    expect(screen.getByLabelText("Email")).toBeInTheDocument();
    expect(screen.getByLabelText("Password")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Sign in and retry" })).toBeInTheDocument();
    expect(screen.getByTestId("credential-recovery-retry")).toHaveTextContent("Try again");
    expect(screen.getByTestId("credential-save").querySelector("svg")).toBeTruthy();

    fireEvent.change(screen.getByLabelText("Email"), { target: { value: "operator@example.test" } });
    fireEvent.change(screen.getByLabelText("Password"), { target: { value: "password" } });
    fireEvent.click(screen.getByRole("button", { name: "Sign in and retry" }));
    await waitFor(() => expect(authApi.loginAuthenticator).toHaveBeenCalledWith("operator@example.test", "password"));
    await waitFor(() => expect(credentialsApi.provisionCredential).toHaveBeenCalledTimes(2));
    expect(screen.queryByTestId("credential-recovery-dialog")).not.toBeInTheDocument();
  });

  it("keeps configuration work behind the review action", async () => {
    readinessApi.fetchReadiness.mockResolvedValue({
      status: "ready",
      scenarios: [],
      resources: [],
      credentials: [],
      hosts: [],
      integrations: [],
      checked_at: "now",
      blockers: [],
      degraded: [],
    });
    credentialsApi.fetchCredentials.mockResolvedValue({ credentials: [], count: 0 });
    capabilitiesApi.fetchCapabilities.mockResolvedValue({ capabilities: [] });
    operatorInputsApi.fetchOperatorInputs.mockResolvedValue({ requests: [] });

    renderWithProviders(<StepCredentials />);

    expect(capabilitiesApi.fetchCapabilities).not.toHaveBeenCalled();
    fireEvent.click(await screen.findByRole("button", { name: "Review credential setup" }));
    expect(await screen.findByRole("dialog", { name: "Credential setup details" })).toBeInTheDocument();
    expect(capabilitiesApi.fetchCapabilities).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("credential-config-tabs")).toBeInTheDocument();
  });
});
