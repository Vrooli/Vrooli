import { describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { queryCorpus } from "../api/documentManager";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { renderWithProviders } from "../test-utils";
import { ReaderPage } from "./ReaderPage";

vi.mock("../api/documentManager", () => ({
  queryCorpus: vi.fn(),
}));

// [REQ:DOC-P0-021]
describe("ReaderPage", () => {
  it("[REQ:DOC-P0-021] exposes anchored results, locality, partial state, and keyboard traversal", async () => {
    vi.mocked(queryCorpus).mockResolvedValue({
      partial: true,
      results: [
        { unit_id: "u1", document_hash: "doc-1", anchor_uri: "vrooli-anchor:1/logical/doc-1#p1", score: 1.2 },
        { unit_id: "u2", document_hash: "doc-1", anchor_uri: "vrooli-anchor:1/logical/doc-1#p2", score: -0.2 },
      ],
    });
    const user = userEvent.setup();
    renderWithProviders(<ReaderPage />);

    const query = screen.getByTestId(selectors.reader.query);
    await user.type(query, "privacy");

    await waitFor(() => expect(screen.getByTestId(selectors.reader.partial)).toBeInTheDocument());
    expect(screen.getByTestId(selectors.reader.unit({ index: 0 }))).toHaveAttribute("aria-controls", "source-u1");
    expect(screen.getAllByTestId(selectors.reader.locality)).toHaveLength(2);

    await user.click(query);
    await user.keyboard("{ArrowDown}");
    expect(screen.getByTestId(selectors.reader.unit({ index: 1 }))).toHaveFocus();
    await user.click(query);
    await user.keyboard("{ArrowUp}");
    expect(screen.getByTestId(selectors.reader.unit({ index: 0 }))).toHaveFocus();
  });

  it("keeps the initial reader empty and handles query failure", async () => {
    vi.mocked(queryCorpus).mockRejectedValue(new Error("retrieval unavailable"));
    const user = userEvent.setup();
    renderWithProviders(<ReaderPage />);

    expect(screen.getByText(strings.pages.reader.noMatches)).toBeInTheDocument();
    await user.keyboard("{ArrowDown}{ArrowUp}");
    await user.type(screen.getByTestId(selectors.reader.query), "missing");
    await waitFor(() => expect(screen.getByText(strings.pages.reader.noMatches)).toBeInTheDocument());
  });
});
