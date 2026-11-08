import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { strings } from "../consts/strings";
import type { SharedPhasePresentation } from "../api/experience";
import { ProviderPresentationCard } from "./ProviderPresentationCard";

type Capability = SharedPhasePresentation["capabilities"][number];

function capability(overrides: Partial<Capability>): Capability {
  return { id: "", label: "", currentLevel: "", currentLevelLabel: "", nextLevel: "", nextUnlock: "", ...overrides };
}

/** A complete presentation, as the proto decoder always produces one. */
function presentation(overrides: Partial<SharedPhasePresentation>): SharedPhasePresentation {
  return {
    contractVersion: "v1", provider: "", phase: "", currentLevel: "", currentLevelLabel: "",
    nextAction: "", northStar: "", documentationTopics: [], capabilities: [], ...overrides,
  };
}

describe("ProviderPresentationCard", () => {
  it("renders supplied capability order without regrouping", () => {
    renderWithProviders(
      <ProviderPresentationCard
        loading={false}
        unavailable={false}
        presentation={presentation({
          capabilities: [
            capability({ id: "second", label: "Second", currentLevel: "L1" }),
            capability({ id: "first", label: "First", currentLevel: "L0" }),
          ],
        })}
      />,
    );
    expect(screen.getByText(strings.experience.findings.providerMaturity)).toBeInTheDocument();
    const rows = screen.getAllByRole("listitem");
    expect(rows[0]).toHaveTextContent("Second");
    expect(rows[1]).toHaveTextContent("First");
  });

  it("labels a historical response instead of reconstructing a phase story", () => {
    renderWithProviders(
      <ProviderPresentationCard loading={false} unavailable={false} presentation={presentation({ contractVersion: "legacy" })} />,
    );
    expect(screen.getByText(strings.experience.findings.providerHistorical)).toBeInTheDocument();
  });
});
