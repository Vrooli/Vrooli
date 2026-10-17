import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { ResultCard } from "./ResultCard";
import type { ProvenanceTag, SearchHit, SurfaceKind } from "../../api/search";

function hit(overrides: Partial<SearchHit> = {}): SearchHit {
  return {
    scenario: "ui-health",
    slot: "Card",
    kind: "component",
    displayName: "Card",
    description: "card description",
    filePath: "ui/src/Card.tsx",
    score: 0.87,
    provenance: "custom",
    library: "",
    componentName: "",
    ...overrides,
  };
}

function renderCard(h: SearchHit) {
  return renderWithProviders(<ResultCard hit={h} index={0} query="card" />);
}

describe("ResultCard", () => {
  it.each<SurfaceKind>(["component", "page", "feature", "hook", "layout", "other", "unspecified"])(
    "renders kind %s",
    (kind) => {
      renderCard(hit({ kind }));
      expect(screen.getByRole("article")).toBeInTheDocument();
    },
  );

  it.each<ProvenanceTag>([
    "custom",
    "adopted-unmodified",
    "adopted-modified",
    "unknown",
    "unspecified",
  ])("renders provenance %s", (provenance) => {
    renderCard(hit({ provenance }));
    expect(screen.getByRole("article")).toBeInTheDocument();
  });

  it("renders without a description or filePath", () => {
    renderCard(hit({ description: "", filePath: "" }));
    expect(screen.getByRole("article")).toBeInTheDocument();
  });

  it("renders '—' for a non-finite score", () => {
    renderCard(hit({ score: Number.NaN }));
    expect(screen.getByRole("article").textContent).toContain("—");
  });

  it("highlights query matches inside the display name", () => {
    renderCard(hit({ displayName: "CardHolder" }));
    const marks = screen.getByRole("article").querySelectorAll("mark");
    expect(marks.length).toBeGreaterThan(0);
  });
});
