import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AdvisorySubjectBar } from "./AdvisorySubjectBar";
import { renderWithProviders } from "../test-utils/renderWithProviders";

describe("AdvisorySubjectBar", () => {
  it("keeps repository and selected scope visible", () => {
    renderWithProviders(<AdvisorySubjectBar repoId="42" scenarioSlug="git-control-tower" fileStats={{ staged: { "a.go": { additions: 1, deletions: 0 } }, unstaged: {}, untracked: {} } as never} />);
    expect(screen.getByRole("region", { name: "Advisory subject" })).toHaveTextContent("Repository 42");
    expect(screen.getByText(/Scope: git-control-tower \(1 file\)/)).toBeInTheDocument();
  });

  it("discloses missing repository identity", () => {
    renderWithProviders(<AdvisorySubjectBar scenarioSlug="git-control-tower" />);
    expect(screen.getByText("Exact subject required")).toBeInTheDocument();
  });
});
