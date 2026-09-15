import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { strings } from "../consts/strings";
import { ProviderPresentationCard } from "./ProviderPresentationCard";

describe("ProviderPresentationCard", () => {
  it("renders supplied capability order without regrouping", () => {
    renderWithProviders(
      <ProviderPresentationCard
        loading={false}
        unavailable={false}
        presentation={{
          contractVersion: "v1",
          capabilities: [
            { id: "second", label: "Second", currentLevel: "L1" },
            { id: "first", label: "First", currentLevel: "L0" },
          ],
        } as never}
      />,
    );
    expect(screen.getByText(strings.experience.findings.providerMaturity)).toBeInTheDocument();
    const rows = screen.getAllByRole("listitem");
    expect(rows[0]).toHaveTextContent("Second");
    expect(rows[1]).toHaveTextContent("First");
  });

  it("labels a historical response instead of reconstructing a phase story", () => {
    renderWithProviders(
      <ProviderPresentationCard loading={false} unavailable={false} presentation={{ contractVersion: "legacy" } as never} />,
    );
    expect(screen.getByText(strings.experience.findings.providerHistorical)).toBeInTheDocument();
  });
});
