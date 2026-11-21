import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { renderWithProviders } from "../test-utils";
import { strings } from "../consts/strings";
import { i18n } from "../i18n";

const api = vi.hoisted(() => ({ fetchServiceOfferPreview: vi.fn(), serviceOfferSampleUrl: vi.fn(() => "/api/v1/proposal-preview/service-offer/sample") }));
vi.mock("../api/serviceoffer", () => api);

import { ServiceOfferPreviewPage } from "./ServiceOfferPreviewPage";

describe("ServiceOfferPreviewPage", () => {
  beforeEach(() => {
    api.fetchServiceOfferPreview.mockResolvedValue({
      offer: {
        id: "company-market-scan", name: "Company Market Scan", summary: "Cited public company snapshot.", audience: "Small teams",
        proposed_price: { amount_minor: 4900, currency: "USD" }, delivery: { target_minutes: 60, format: "markdown" },
        verification_promise: "Every claim links to a public source.", refund_rule: "Release if citations fail.",
        legal: { reviewed_at: "2026-11-13", status: "open", notes: ["Public sources only."], evidence: ["https://www.sec.gov/"] },
        operator_approval: "pending",
      },
      surfaces: {
        checkout: { mode: "test", payment_mode: "authorize-then-capture", capture_when: "after-delivery-verification" },
        mcp: { name: "company-market-scan", description: "Cited public company snapshot.", payment_path: "checkout:company-market-scan" },
        mpp: { path: "/api/v1/services/company-market-scan", mode: "test", capture_when: "after-delivery-verification" },
        plugin: { disposition: "free-discovery-only", landing_path: "/services/company-market-scan" },
        page: { free_sample_path: "/samples/company-market-scan.md", publication_state: "staged" },
      },
      sample: "sample",
    });
  });

  afterEach(() => { cleanup(); vi.clearAllMocks(); });

  it("shows the phone-friendly proposal, free sample, and disabled purchase state", async () => {
    const t = i18n.t;
    renderWithProviders(<ServiceOfferPreviewPage />);

    expect(await screen.findByRole("heading", { level: 1 })).toBeInTheDocument();
    expect(screen.getByText("$49.00")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: t(strings.pages.serviceOfferPreview.sampleLink) })).toHaveAttribute("href", "/api/v1/proposal-preview/service-offer/sample");
    expect(screen.getByRole("button", { name: t(strings.pages.serviceOfferPreview.purchaseDisabled) })).toBeDisabled();
    expect(screen.getByText(t(strings.pages.serviceOfferPreview.mcpTool))).toBeInTheDocument();
  });
});
