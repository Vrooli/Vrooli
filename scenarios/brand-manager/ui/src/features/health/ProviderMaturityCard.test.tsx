import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";

const fetchBrandingValidation = vi.hoisted(() => vi.fn());
vi.mock("../../api/validation", async () => {
  const actual = await vi.importActual<typeof import("../../api/validation")>("../../api/validation");
  return { ...actual, fetchBrandingValidation };
});

import { ProviderMaturityCard } from "./ProviderMaturityCard";

describe("ProviderMaturityCard", () => {
  it("renders the provider capability sequence without regrouping", async () => {
    fetchBrandingValidation.mockResolvedValueOnce({
      assessment: {
        presentation: {
          contractVersion: "v1",
          capabilities: [
            { id: "second", label: "Second", currentLevel: "L1" },
            { id: "first", label: "First", currentLevel: "L0" },
          ],
        },
      },
    });
    renderWithProviders(<ProviderMaturityCard />);
    expect(await screen.findByText(strings.health.providerMaturity)).toBeInTheDocument();
    await screen.findByText("Second");
    const rows = screen.getAllByRole("listitem");
    expect(rows[0]).toHaveTextContent("Second");
    expect(rows[1]).toHaveTextContent("First");
  });

  it("labels an unsupported presentation instead of synthesizing one", async () => {
    fetchBrandingValidation.mockResolvedValueOnce({ assessment: { presentation: { contractVersion: "legacy" } } });
    renderWithProviders(<ProviderMaturityCard />);
    expect(await screen.findByText(strings.health.providerHistorical)).toBeInTheDocument();
  });
});
