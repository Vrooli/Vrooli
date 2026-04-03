import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { EvidenceContextSummary } from "./evidence-context-summary";
import { selectors } from "../../consts/selectors";
import type { ReviewRound, EvidenceItem } from "../../services/review-service";

function makeEvidence(overrides?: Partial<EvidenceItem>): EvidenceItem {
  return {
    id: "ev-1",
    type: "screenshot",
    title: "Homepage screenshot",
    description: "Screenshot of the homepage",
    capture_path: "/captures/homepage.png",
    verified: false,
    ...overrides,
  } as EvidenceItem;
}

function makeRound(overrides?: Partial<ReviewRound>): ReviewRound {
  return {
    round: 1,
    generated_at: new Date().toISOString(),
    execution_id: "exec-1",
    status: "complete",
    classification: "needs_work",
    agent_assessment: "Tests failed in deployment-manager",
    evidence: [makeEvidence()],
    ...overrides,
  } as ReviewRound;
}

describe("EvidenceContextSummary", () => {
  it("renders nothing when no rounds", () => {
    const { container } = render(<EvidenceContextSummary rounds={[]} />);
    expect(container.firstChild).toBeNull();
  });

  it("shows latest round classification and assessment", () => {
    render(<EvidenceContextSummary rounds={[makeRound()]} />);
    expect(screen.getByTestId(selectors.review.evidenceContextSummary)).toBeInTheDocument();
    expect(screen.getByText(/Round 1/)).toBeInTheDocument();
    expect(screen.getByText("Tests failed in deployment-manager")).toBeInTheDocument();
  });

  it("lists evidence items by title", () => {
    render(<EvidenceContextSummary rounds={[makeRound()]} />);
    expect(screen.getByText("Homepage screenshot")).toBeInTheDocument();
  });
});
