import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AdvisoryDraftComposer } from "./AdvisoryDraftComposer";
import { renderWithProviders } from "../test-utils/renderWithProviders";

describe("AdvisoryDraftComposer", () => {
  it("keeps the draft explicitly local and non-publishing", () => {
    renderWithProviders(<AdvisoryDraftComposer scenarioSlug="git-control-tower" repoId="42" />);
    expect(screen.getByRole("region", { name: "Advisory draft composer" })).toHaveTextContent("Draft only");
    expect(screen.getByLabelText("Draft kind")).toHaveValue("pr-draft");
  });
});
