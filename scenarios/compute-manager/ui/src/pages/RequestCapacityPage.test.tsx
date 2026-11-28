import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { fetchSpendReceipt, requestInstance } from "../api/compute";
import { renderWithProviders } from "../test-utils";
import { RequestCapacityPage } from "./RequestCapacityPage";
import { fetchProviderCatalogue } from "../api/provider-catalogue";

vi.mock("../api/compute", () => ({
  requestInstance: vi.fn(),
  fetchSpendReceipt: vi.fn(),
}));
vi.mock("../api/provider-catalogue", () => ({ fetchProviderCatalogue: vi.fn() }));

const future = 4_000_000_000n;
const catalogueFixture = (options: { providerId?: string; size?: string; price?: boolean; stale?: boolean; serverMinor?: bigint; purchaseEnabled?: boolean; prices?: unknown[] } = {}) => {
  const providerId = options.providerId ?? "hetzner";
  const size = options.size ?? "cx23";
  const providers = [{ id: providerId, name: providerId === "digitalocean" ? "DigitalOcean" : providerId === "akash" ? "Akash Testnet" : "Hetzner Cloud", offers: [{ id: size, label: size.toUpperCase(), regions: [providerId === "digitalocean" ? "nyc1" : providerId === "akash" ? "sandbox-2" : "nbg1"] }], billingFacts: providerId === "akash" ? { network: "testnet", effect_mode: "stub", blocked_reason: "shared effect gate is closed", price_status: "no_fact" } : {} }];
  const price = { provider: providerId, offerId: `${size}-fixture`, region: providers[0]?.offers[0]?.regions[0] ?? "nbg1", size, currency: "USD", serverMinorPerMonth: options.serverMinor ?? 649n, ipv4MinorPerMonth: 60n, taxBasis: "pre-tax", sourceUrl: "https://fixture.invalid/prices", sourceVersion: "fixture-only", observedAt: { seconds: 1_750_000_000n, nanos: 0 }, expiresAt: { seconds: options.stale ? 1_700_000_000n : future, nanos: 0 } };
  return { providers, readiness: { [providerId]: { current: "payment_valid", credentialStatus: "configured", purchaseEnabled: options.purchaseEnabled ?? true, prices: options.prices ?? (options.price === false ? [] : [price]) } } } as never;
};

describe("RequestCapacityPage", () => {
  afterEach(() => {
    cleanup();
    vi.resetAllMocks();
    sessionStorage.clear();
  });

  beforeEach(() => {
    vi.mocked(fetchProviderCatalogue).mockResolvedValue(catalogueFixture());
  });

  it("refuses an invalid lifetime before calling the API", async () => {
    const user = userEvent.setup();
    renderWithProviders(<RequestCapacityPage />);
    const lifetime = screen.getByLabelText(/lifetime/i);
    await user.clear(lifetime);
    fireEvent.submit(screen.getByTestId("page-request-form"));

    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(requestInstance).not.toHaveBeenCalled();
  });

  it("renders success after the request resolves", async () => {
    vi.mocked(requestInstance).mockResolvedValueOnce({} as never);
    vi.mocked(fetchSpendReceipt).mockRejectedValueOnce(new Error("not found yet"));
    const user = userEvent.setup();
    renderWithProviders(<RequestCapacityPage />);
    await user.click(screen.getByRole("button"));

    await waitFor(() => expect(screen.getByText("pages.request.success")).toBeInTheDocument());
    expect(requestInstance).toHaveBeenCalledTimes(1);
  });

  it("renders an error when the request is refused", async () => {
    vi.mocked(requestInstance).mockRejectedValueOnce(new Error("refused"));
    vi.mocked(fetchSpendReceipt).mockRejectedValueOnce(new Error("receipt lookup failed"));
    const user = userEvent.setup();
    renderWithProviders(<RequestCapacityPage />);
    await user.click(screen.getByRole("button"));

    await waitFor(() => expect(screen.getByRole("alert")).toBeInTheDocument());
  });

  it("shows stored line items and the refused cap reason", async () => {
    vi.mocked(fetchProviderCatalogue).mockResolvedValue(catalogueFixture({ size: "cx33" }));
    vi.mocked(requestInstance).mockRejectedValueOnce(new Error("refused"));
    vi.mocked(fetchSpendReceipt).mockResolvedValueOnce({ receipt: {
      idempotencyKey: "request-cx33", provider: "hetzner", offerId: "cx33-nbg1", region: "nbg1", size: "cx33", ipv4: true, term: "monthly", currency: "USD",
      serverMinorPerMonth: 899n, ipv4MinorPerMonth: 60n, totalMinorPerMonth: 959n, taxBasis: "pre-tax", sourceUrl: "https://api.hetzner.cloud/v1/pricing", sourceVersion: "etag-2",
      observedAt: { seconds: 1795435200n, nanos: 0 }, expiresAt: { seconds: 1795435500n, nanos: 0 }, quoteState: "refused_over_cap", quoteReason: "monthly quote USD 959 exceeds cap 709",
      capCurrency: "USD", capMonthlyMinor: 709n, capTaxBasis: "pre-tax", admissionDecision: "refused_over_cap", admissionReason: "monthly quote USD 959 exceeds cap 709", createdAt: { seconds: 1795435201n, nanos: 0 }, requestState: "refused",
    } } as never);
    renderWithProviders(<RequestCapacityPage />);
    await userEvent.setup().click(screen.getByRole("button", { name: "pages.request.submit" }));
    const receipt = await screen.findByTestId("spend-receipt");
    expect(receipt).toHaveTextContent("USD 8.99");
    expect(receipt).toHaveTextContent("USD 0.60");
    expect(receipt).toHaveTextContent("USD 9.59");
    expect(receipt).toHaveTextContent("USD 7.09");
    expect(receipt).toHaveTextContent("refused_over_cap");
    expect(receipt).toHaveTextContent("etag-2");
  });

  it("keeps missing billing facts unknown instead of rendering zero as a price", async () => {
    vi.mocked(requestInstance).mockRejectedValueOnce(new Error("no price fact"));
    vi.mocked(fetchSpendReceipt).mockResolvedValueOnce({ receipt: {
      idempotencyKey: "request-no-price", provider: "hetzner", offerId: "cx23-nbg1", region: "nbg1", size: "cx23", ipv4: true, term: "monthly", currency: "",
      serverMinorPerMonth: 0n, ipv4MinorPerMonth: 0n, totalMinorPerMonth: 0n, taxBasis: "", sourceUrl: "", sourceVersion: "",
      quoteState: "refused_missing_price", quoteReason: "no fresh catalogue fact", capCurrency: "USD", capMonthlyMinor: 709n, capTaxBasis: "pre-tax",
      admissionDecision: "refused_missing_price", admissionReason: "no fresh catalogue fact", requestState: "refused",
    } } as never);
    vi.mocked(fetchProviderCatalogue).mockResolvedValue(catalogueFixture({ price: false, purchaseEnabled: false }));
    renderWithProviders(<RequestCapacityPage />);
    expect(await screen.findByText(/Price unknown/)).toBeInTheDocument();
    expect(screen.getByTestId("catalogue-admission")).toHaveTextContent("not zero");
    expect(screen.getByTestId("page-request-submit")).toBeDisabled();
    expect(requestInstance).not.toHaveBeenCalled();
    expect(screen.queryByText("USD 0.00/month")).not.toBeInTheDocument();
  });

  it("does not show an expired catalogue observation as a live price", async () => {
    vi.mocked(fetchProviderCatalogue).mockResolvedValue(catalogueFixture({ stale: true }));
    renderWithProviders(<RequestCapacityPage />);
    expect(await screen.findByText(/Price unknown/)).toBeInTheDocument();
    expect(screen.getByTestId("catalogue-admission")).toHaveTextContent("not zero");
    expect(screen.getByTestId("page-request-submit")).toBeDisabled();
    expect(screen.getByTestId("catalogue-admission")).not.toHaveTextContent("USD 7.09/month");
    expect(requestInstance).not.toHaveBeenCalled();
  });

  it("selects DigitalOcean from the generated catalogue and submits its exact persisted offer", async () => {
    vi.mocked(fetchProviderCatalogue).mockResolvedValue(catalogueFixture({ providerId: "digitalocean", size: "s-1vcpu-1gb" }));
    vi.mocked(requestInstance).mockResolvedValueOnce({} as never);
    vi.mocked(fetchSpendReceipt).mockRejectedValueOnce(new Error("not found yet"));
    renderWithProviders(<RequestCapacityPage />);
    const submit = await screen.findByRole("button", { name: "pages.request.submit" });
    expect(screen.getByTestId("catalogue-admission")).toHaveTextContent("USD 7.09/month");
    await userEvent.setup().click(submit);
    await waitFor(() => expect(requestInstance).toHaveBeenCalledWith(expect.objectContaining({ provider: "digitalocean", size: "s-1vcpu-1gb", region: "nyc1" })));
  });

  it("reattaches a reloaded request to the provider and selection in its stored receipt", async () => {
    sessionStorage.setItem("compute-manager.pending-spend-request", "stored-do-request");
    vi.mocked(fetchProviderCatalogue).mockResolvedValue(catalogueFixture({ providerId: "digitalocean", size: "s-1vcpu-1gb" }));
    vi.mocked(fetchSpendReceipt).mockResolvedValue({ receipt: {
      idempotencyKey: "stored-do-request", provider: "digitalocean", offerId: "fixture-do-offer", region: "nyc1", size: "s-1vcpu-1gb", requestState: "refused", currency: "", sourceUrl: "", sourceVersion: "", capMonthlyMinor: 709n, capCurrency: "USD", capTaxBasis: "pre-tax",
    } } as never);
    renderWithProviders(<RequestCapacityPage />);
    await waitFor(() => expect(screen.getByLabelText("Compute source")).toHaveValue("digitalocean"));
    expect(screen.getByTestId("spend-receipt")).toHaveTextContent("digitalocean");
    expect(screen.getByTestId("spend-receipt")).toHaveTextContent("s-1vcpu-1gb · nyc1");
    expect(requestInstance).not.toHaveBeenCalled();
  });

  it("blocks an over-cap catalogue quote before calling the request API", async () => {
    vi.mocked(fetchProviderCatalogue).mockResolvedValue(catalogueFixture({ size: "cx33", serverMinor: 899n }));
    renderWithProviders(<RequestCapacityPage />);
    expect(await screen.findByText(/over the approved A2 monthly cap/i)).toBeInTheDocument();
    expect(screen.getByTestId("page-request-submit")).toBeDisabled();
    expect(requestInstance).not.toHaveBeenCalled();
  });

  it("keeps a decentralized source unavailable when the catalogue reports a closed gate and no price", async () => {
    vi.mocked(fetchProviderCatalogue).mockResolvedValue(catalogueFixture({ providerId: "akash", size: "lease", price: false, purchaseEnabled: false }));
    renderWithProviders(<RequestCapacityPage />);
    expect(await screen.findByText(/Price unknown/)).toBeInTheDocument();
    expect(screen.getByTestId("catalogue-admission")).toHaveTextContent("purchase for this source");
    expect(screen.getByTestId("catalogue-admission")).toHaveTextContent("effect gate");
    expect(screen.getByTestId("page-request-submit")).toBeDisabled();
    expect(requestInstance).not.toHaveBeenCalled();
  });

  it("routes existing VPS owners to the typed declared-cost admission flow", async () => {
    renderWithProviders(<RequestCapacityPage />);
    await userEvent.setup().selectOptions(await screen.findByLabelText("Compute source"), "adopted-vps");
    expect(screen.getByTestId("adopted-source")).toHaveTextContent("explicit zero");
    expect(screen.getByRole("link", { name: /stored admission receipt/i })).toHaveAttribute("href", "/adopt");
    expect(requestInstance).not.toHaveBeenCalled();
  });
});
