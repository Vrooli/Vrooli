import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";

const retrieval = vi.hoisted(() => ({ search: vi.fn() }));
vi.mock("../../api/retrieval", () => ({ retrievalClient: retrieval }));

import { SearchCard } from "./SearchCard";

describe("SearchCard [REQ:SIG-P0-009] [REQ:SIG-P0-010]", () => {
  beforeEach(() => retrieval.search.mockResolvedValue({ results: [{ signal: { id: "sig-dropped", sourceUrl: "https://example.test", extractedContent: "A preserved signal", captureNote: "", tags: ["research"] }, disposition: "dropped", categoryId: "reading", score: 0.91 }] }));
  afterEach(() => { cleanup(); vi.clearAllMocks(); });

  it("searches the full journal with structured filters, including dropped signals", async () => {
    const user = userEvent.setup();
    renderWithProviders(<SearchCard />);
    await user.type(screen.getByLabelText("Search text"), "preserved idea");
    await user.type(screen.getByLabelText("Search tags"), "research, inbox");
    await user.selectOptions(screen.getByLabelText("Disposition filter"), "dropped");
    await user.click(screen.getByRole("button", { name: "Search journal" }));
    await waitFor(() => expect(retrieval.search).toHaveBeenCalledWith(expect.objectContaining({ filter: expect.objectContaining({ text: "preserved idea", tags: ["research", "inbox"], disposition: "dropped" }) })));
    expect(await screen.findByText("A preserved signal")).toBeInTheDocument();
    expect(screen.getByText(/Search includes every captured signal/)).toBeInTheDocument();
  });

  it("renders fallback result fields and pages through a cursor", async () => {
    retrieval.search
      .mockResolvedValueOnce({
        results: [{ signal: { id: "sig-text", sourceKind: "text", captureNote: "A captured note", tags: [] }, disposition: "", categoryId: "", score: 0 }],
        nextPageAfter: "next-page",
      })
      .mockResolvedValueOnce({ results: [{ signal: { id: "sig-next", sourceKind: "image", extractedContent: "Second page", tags: ["visual"] }, disposition: "triaged", categoryId: "visuals", score: 0.4 }], nextPageAfter: "" });
    const user = userEvent.setup();
    renderWithProviders(<SearchCard />);
    await user.type(screen.getByLabelText("Captured after"), "2025-01-01");
    await user.type(screen.getByLabelText("Captured before"), "2025-01-02");
    await user.click(screen.getByRole("button", { name: "Search journal" }));
    expect(await screen.findByText("A captured note")).toBeInTheDocument();
    expect(screen.getByText("text")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Load more results" }));
    expect(await screen.findByText("Second page")).toBeInTheDocument();
    expect(screen.getByText("Tags: visual")).toBeInTheDocument();
    expect(screen.getByText("Relevance: 0.40")).toBeInTheDocument();
  });

});
