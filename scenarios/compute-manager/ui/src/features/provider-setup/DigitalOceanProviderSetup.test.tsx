import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ProviderSetupPage } from "../../pages/ProviderSetupPage";

const setupMock = vi.hoisted(() => ({
  listProviders: vi.fn(),
  getReadiness: vi.fn(),
  beginSetupStep: vi.fn(),
  verifyProvider: vi.fn(),
}));

vi.mock("@connectrpc/connect", () => ({ createClient: () => setupMock }));

const digitalOceanDescriptor = {
  id: "digitalocean",
  name: "DigitalOcean",
  consoleUrl: "https://cloud.digitalocean.com/",
  credentialLogicalId: "vrooli/compute-manager",
  credentialField: "digitalocean-api-token",
  credentialLabel: "DigitalOcean API token for authorized lifecycle operations",
  credentialScopes: ["droplets:read", "droplets:write", "sizes:read", "regions:read"],
  consoleSteps: [
    { id: "account", title: "Confirm provider account", instructions: "Sign in on the provider website.", url: "https://cloud.digitalocean.com/", decision: "Can you open your account?" },
    { id: "payment", title: "Confirm payment method", instructions: "Review billing on the provider website.", url: "https://cloud.digitalocean.com/account/billing", decision: "Is a payment method present and valid?" },
    { id: "token", title: "Create the Compute Manager API token", instructions: "Enter it only through secure Vrooli Onboarding.", url: "https://cloud.digitalocean.com/account/api/tokens", decision: "Have you created the API token?" },
  ],
  probes: [
    { id: "droplets", method: "GET", path: "/v2/droplets", permission: "read droplets", readOnly: true },
    { id: "sizes", method: "GET", path: "/v2/sizes", permission: "read sizes", readOnly: true },
    { id: "regions", method: "GET", path: "/v2/regions", permission: "read regions", readOnly: true },
  ],
  offers: [{ id: "s-1vcpu-1gb", label: "Basic Droplet", cpu: "1 vCPU", memory: "1 GB", regions: [] }],
  billingFacts: { rounding: "second", minimum: "one minute", stopped: "continues billing", inbound: "not billed", source: "https://docs.digitalocean.com/products/droplets/details/pricing/" },
};

const readiness = (current: string, credentialStatus: string, prices: unknown[] = [], evidence: unknown[] = []) => ({
  readiness: { providerId: "digitalocean", current, credentialStatus, purchaseEnabled: false, evidence, prices },
});

const paragraphsContaining = (container: HTMLElement, fragment: string) =>
  Array.from(container.querySelectorAll("p")).filter((paragraph) => paragraph.textContent.includes(fragment));

describe("DigitalOcean catalogue setup UI", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupMock.listProviders.mockResolvedValue({ providers: [digitalOceanDescriptor] });
    setupMock.getReadiness.mockResolvedValue(readiness("credential_needed", "missing"));
    setupMock.beginSetupStep.mockResolvedValue({ askId: "fixture-ask", state: "awaiting_decision" });
  });

  it("renders the server catalogue entry, exact phone steps and secure credential handoff", async () => {
    render(<ProviderSetupPage />);
    expect(await screen.findByRole("heading", { name: "DigitalOcean" })).toBeInTheDocument();
    digitalOceanDescriptor.consoleSteps.forEach((step) => expect(screen.getByText(step.title)).toBeInTheDocument());
    expect(screen.getByText((_, element) => element?.tagName === "STRONG" && element.textContent === "Required scopes:")).toBeInTheDocument();
    expect(screen.getByText(digitalOceanDescriptor.credentialScopes.join(", "))).toBeInTheDocument();
    const handoff = document.querySelector<HTMLAnchorElement>('a[href="/embedded/vrooli-onboarding/?target=local&step=credentials"]');
    expect(handoff).not.toBeNull();
    expect(handoff?.getAttribute("href")).toBe("/embedded/vrooli-onboarding/?target=local&step=credentials");
    expect(screen.queryByRole("textbox", { name: /token/i })).not.toBeInTheDocument();
    expect(paragraphsContaining(document.body, "Purchase remains disabled")).toHaveLength(1);
  });

  it("routes a DigitalOcean phone decision by provider and step, then shows authenticated/account-ready read-back", async () => {
    setupMock.getReadiness
      .mockResolvedValueOnce(readiness("credential_needed", "missing"))
      .mockResolvedValueOnce(readiness("account_usable", "configured", [], [
        { id: "authenticated", state: "authenticated", verified: true, detail: "fixture credential authority read-back" },
        { id: "console:account", state: "account_usable", verified: true, detail: "fixture read-only probe" },
      ]));
    render(<ProviderSetupPage />);
    await waitFor(() => expect(paragraphsContaining(document.body, "Credential status: missing")).toHaveLength(1));
    const accountStep = digitalOceanDescriptor.consoleSteps.find((step) => step.id === "account");
    if (!accountStep) throw new Error("DigitalOcean account step descriptor missing");
    const accountItem = screen.getByRole("heading", { name: accountStep.title }).closest("li");
    if (!accountItem) throw new Error("DigitalOcean account step list item missing");
    fireEvent.click(within(accountItem).getByRole("button"));
    await waitFor(() => expect(setupMock.beginSetupStep).toHaveBeenCalledWith(expect.objectContaining({ providerId: "digitalocean", stepId: "account" })));
    const accountReadyLabel = "account usable";
    expect(await screen.findByText(accountReadyLabel)).toBeInTheDocument();
    await waitFor(() => expect(paragraphsContaining(document.body, "Credential status: configured")).toHaveLength(1));
    expect(Array.from(accountItem.querySelectorAll("p")).some((paragraph) => paragraph.textContent === "Confirmed")).toBe(true);
    expect(setupMock.verifyProvider).not.toHaveBeenCalled();
  });

  it("marks stale price evidence expired and keeps any purchase action absent", async () => {
    setupMock.getReadiness.mockResolvedValue(readiness("offer_available", "configured", [{
      provider: "digitalocean", offerId: "s-1vcpu-1gb", region: "nyc3", size: "s-1vcpu-1gb", currency: "USD",
      serverMinorPerMonth: 600n, ipv4MinorPerMonth: 0n, taxBasis: "pre-tax",
      sourceUrl: "https://docs.digitalocean.com/reference/api/reference/sizes/", sourceVersion: "fixture-stale",
      observedAt: { seconds: BigInt(Math.floor(Date.now() / 1000) - 7200), nanos: 0 }, expiresAt: { seconds: BigInt(Math.floor(Date.now() / 1000) - 3600), nanos: 0 },
    }]));
    render(<ProviderSetupPage />);
    expect(await screen.findByText(/Expired observation:/)).toBeInTheDocument();
    expect(screen.getByText(/fixture-stale/)).toBeInTheDocument();
    expect(paragraphsContaining(document.body, "Purchase remains disabled")).toHaveLength(1);
    expect(screen.queryByRole("button", { name: /purchase|create droplet|buy/i })).not.toBeInTheDocument();
  });

  it("keeps unknown pricing non-actionable and does not run a verification probe by rendering", async () => {
    render(<ProviderSetupPage />);
    await waitFor(() => expect(paragraphsContaining(document.body, "Credential status: missing")).toHaveLength(1));
    expect(paragraphsContaining(document.body, "Purchase remains disabled")).toHaveLength(1);
    expect(setupMock.verifyProvider).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: /Verify .* in selected region/i })).toBeDisabled();
  });

  it("keeps readiness blocked when the backend refuses a probe without an authorized token", async () => {
    setupMock.verifyProvider.mockResolvedValue(readiness("credential_needed", "missing"));
    render(<ProviderSetupPage />);
    const region = await screen.findByRole("textbox", { name: /Region/i });
    fireEvent.change(region, { target: { value: "nyc3" } });
    const verify = screen.getByRole("button", { name: /Verify .* in selected region/i });
    fireEvent.click(verify);
    await waitFor(() => expect(setupMock.verifyProvider).toHaveBeenCalledWith(expect.objectContaining({
      providerId: "digitalocean", region: "nyc3", size: "s-1vcpu-1gb",
    })));
    await waitFor(() => expect(paragraphsContaining(document.body, "Credential status: missing")).toHaveLength(1));
    const missingCredentialState = "credential needed";
    expect(screen.getByText(missingCredentialState)).toBeInTheDocument();
    expect(paragraphsContaining(document.body, "Purchase remains disabled")).toHaveLength(1);
  });
});
