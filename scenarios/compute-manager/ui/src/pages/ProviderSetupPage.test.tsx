import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ProviderSetupPage } from "./ProviderSetupPage";
import { strings } from "../consts/strings";

const setupMock = vi.hoisted(() => ({
  listProviders: vi.fn(),
  getReadiness: vi.fn(),
  beginSetupStep: vi.fn(),
  verifyProvider: vi.fn(),
}));

vi.mock("@connectrpc/connect", () => ({ createClient: () => setupMock }));

const akashDescriptor = {
  id: "akash",
  name: "Akash",
  consoleUrl: "",
  credentialLogicalId: "vrooli/compute-manager",
  credentialField: "akash-testnet-wallet",
  credentialLabel: "Akash sandbox-2 wallet signing credential",
  credentialScopes: ["sign transactions"],
  consoleSteps: [{ id: "wallet", title: "Enter the existing sandbox wallet securely", instructions: "fixture step", url: "/embedded/vrooli-onboarding/?target=local&step=credentials", decision: "fixture question" }],
  probes: [],
  offers: [],
  billingFacts: {
      network: "sandbox-2",
    effect_mode: "stub",
    credential_reference: "credential-authority://vrooli/compute-manager/akash-testnet-wallet",
    price_status: "no_valid_price",
    credential_authority_signer_status: "unverified",
    blocked_reason: "No verified signer or authoritative price catalogue",
    setup_actions_enabled: "false",
    read_only_probe_enabled: "false",
  },
};

describe("ProviderSetupPage Akash status", () => {
  beforeEach(() => vi.clearAllMocks());

  it("states the Akash descriptor is absent instead of inventing readiness", async () => {
    setupMock.listProviders.mockResolvedValue({ providers: [] });
    render(<ProviderSetupPage />);
    expect(await screen.findByTestId("akash-not-listed")).toHaveTextContent(strings.providerSetup.akash.descriptorMissing);
    expect(screen.getByTestId("akash-not-listed")).toHaveTextContent(strings.providerSetup.akash.noActions);
    expect(screen.queryByRole("button", { name: /verify/i })).not.toBeInTheDocument();
  });

  it("shows the secure wallet entry and phone prompt without probing or treating the URI as readiness", async () => {
    setupMock.listProviders.mockResolvedValue({ providers: [akashDescriptor] });
    setupMock.getReadiness.mockResolvedValue({ readiness: {
      providerId: "akash", current: "credential_needed", credentialStatus: "not_configured", purchaseEnabled: false,
      evidence: [], prices: [],
    } });
    render(<ProviderSetupPage />);
    const status = await screen.findByTestId("akash-readiness");
    expect(status).toHaveTextContent("sandbox-2");
    expect(status).toHaveTextContent("stub");
    expect(status).toHaveTextContent(strings.providerSetup.akash.signerUnverified);
    expect(status).toHaveTextContent("no_valid_price");
    expect(status).toHaveTextContent("No verified signer or authoritative price catalogue");
    expect(await screen.findByTestId("akash-no-price")).toHaveTextContent(strings.providerSetup.akash.noPriceFact);
    expect(screen.getByText(strings.providerSetup.akash.actionsDisabled)).toBeInTheDocument();
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
    expect(screen.getByTestId("akash-wallet-entry")).toHaveTextContent("not_configured");
    expect(screen.getByTestId("akash-wallet-entry")).toHaveTextContent("credential-authority://vrooli/compute-manager/akash-testnet-wallet");
    expect(screen.getByTestId("akash-wallet-entry")).toHaveTextContent(strings.providerSetup.akash.walletIdentifierCaveat);
    expect(screen.getAllByRole("link", { name: strings.providerSetup.akash.secureEntryAction }).length).toBeGreaterThan(0);
    expect(screen.getByRole("button", { name: /ask me on my phone/i })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /verify/i })).not.toBeInTheDocument();
    expect(setupMock.verifyProvider).not.toHaveBeenCalled();
    await userEvent.click(screen.getByRole("button", { name: /ask me on my phone/i }));
    expect(setupMock.beginSetupStep).toHaveBeenCalledWith(expect.objectContaining({ providerId: "akash", stepId: "wallet" }));
  });
});
