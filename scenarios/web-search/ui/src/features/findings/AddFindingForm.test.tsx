import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { FindingSource } from "@vrooli/proto-types/web-search/v1/findings/findings_pb";

vi.mock("../../api/clients", () => ({
  findingsClient: { addFinding: vi.fn() },
  liveSearchClient: { search: vi.fn() },
}));

import { findingsClient } from "../../api/clients";
import { AddFindingForm } from "./AddFindingForm";

const findingsKey = ["findings", "all", false] as const;

describe("AddFindingForm", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("requires a claim before submitting", () => {
    renderWithProviders(<AddFindingForm findingsKey={findingsKey} />);
    fireEvent.click(screen.getByTestId(selectors.findings.addSubmit));
    expect(findingsClient.addFinding).not.toHaveBeenCalled();
    expect(screen.getByText(strings.findings.claimRequired)).toBeInTheDocument();
  });

  it("submits the claim, confidence, query and citation rows", async () => {
    vi.mocked(findingsClient.addFinding).mockResolvedValue({ finding: { id: "f1" } } as never);

    renderWithProviders(<AddFindingForm findingsKey={findingsKey} />);
    fireEvent.change(screen.getByTestId(selectors.findings.addClaim), {
      target: { value: "A new claim" },
    });
    fireEvent.change(screen.getByTestId(selectors.findings.addConfidence), {
      target: { value: "0.7" },
    });
    fireEvent.change(screen.getByTestId(selectors.findings.addQuery), {
      target: { value: "a query" },
    });
    fireEvent.click(screen.getByText(strings.findings.addCitationRow));
    fireEvent.change(screen.getByTestId(selectors.findings.addCitationUrl), {
      target: { value: "https://src.example" },
    });
    fireEvent.change(screen.getByTestId(selectors.findings.addCitationTitle), {
      target: { value: "Source" },
    });
    fireEvent.click(screen.getByTestId(selectors.findings.addSubmit));

    await waitFor(() => {
      expect(findingsClient.addFinding).toHaveBeenCalledWith({
        claim: "A new claim",
        confidence: 0.7,
        query: "a query",
        source: FindingSource.MANUAL,
        briefId: "",
        citations: [{ url: "https://src.example", title: "Source" }],
      });
    });
  });

  it("drops citation rows with a blank URL", async () => {
    vi.mocked(findingsClient.addFinding).mockResolvedValue({ finding: { id: "f1" } } as never);

    renderWithProviders(<AddFindingForm findingsKey={findingsKey} />);
    fireEvent.change(screen.getByTestId(selectors.findings.addClaim), {
      target: { value: "Claim" },
    });
    fireEvent.click(screen.getByText(strings.findings.addCitationRow));
    // leave URL blank
    fireEvent.click(screen.getByTestId(selectors.findings.addSubmit));

    await waitFor(() => {
      expect(findingsClient.addFinding).toHaveBeenCalledWith(
        expect.objectContaining({ citations: [] }),
      );
    });
  });
});
