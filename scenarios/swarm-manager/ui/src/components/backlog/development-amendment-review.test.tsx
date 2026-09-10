import { create } from "@bufbuild/protobuf";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { DevelopmentArtifactSchema, PreviewDevelopmentRequestSchema } from "@vrooli/proto-types/swarm-manager/v1/api/transition_pb";
import { renderWithProviders } from "../../test-utils";
import { DevelopmentAmendmentReview } from "./development-amendment-review";

describe("development amendment review [REQ:SWM-P0-017]", () => {
  it("exposes outcome weakening, expanded scope, budget and artifact changes", () => {
    const previous = create(PreviewDevelopmentRequestSchema, { maxTokens: 10n, acceptanceAllow: ["scenarios/example/**"], outcomes: [{ id: "latency", criterion: "p95 below 500ms", evidenceSource: "audio-tools" }] });
    const proposed = create(PreviewDevelopmentRequestSchema, { maxTokens: 20n, acceptanceAllow: ["scenarios/**"] });
    const retained = [create(DevelopmentArtifactSchema, { path: "old.md", sha256: "old" })];
    const resolved = [create(DevelopmentArtifactSchema, { path: "new.md", sha256: "new" })];
    renderWithProviders(<DevelopmentAmendmentReview previous={previous} proposed={proposed} retained={retained} resolved={resolved} />);
    for (const field of ["acceptance_allow", "max_tokens", "outcomes", "Artifact: new.md", "Artifact: old.md"]) expect(screen.getByText(field)).toBeInTheDocument();
    expect(screen.getByRole("table")).toBeInTheDocument();
    expect(screen.getByText(/p95 below 500ms/)).toBeInTheDocument();
    expect(screen.getByText("Removed")).toBeInTheDocument();
  });
});
