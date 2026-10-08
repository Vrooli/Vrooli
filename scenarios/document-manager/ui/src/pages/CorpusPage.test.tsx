import { describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";

import { listCollections, listDocuments } from "../api/documentManager";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { renderWithProviders } from "../test-utils";
import { CorpusPage } from "./CorpusPage";

vi.mock("../api/documentManager", () => ({
  listCollections: vi.fn(),
  listDocuments: vi.fn(),
}));

// [REQ:DOC-P0-023]
describe("CorpusPage", () => {
  it("[REQ:DOC-P0-023] renders live collection privacy and document locality", async () => {
    vi.mocked(listCollections).mockResolvedValue({ collections: [{ id: "c1", name: "Research", default_privacy_class: 2, federated: false }, { id: "c2", name: "Shared", default_privacy_class: 2, federated: true }] });
    vi.mocked(listDocuments).mockResolvedValue({ documents: [{ id: "d1", content_sha256: "sha", source_name: "brief.pdf", detected_mime: "application/pdf", privacy_class: 2 }, { id: "d2", content_sha256: "sha2", source_name: "", detected_mime: "text/plain", privacy_class: 1 }] });
    renderWithProviders(<CorpusPage />);

    await waitFor(() => expect(screen.getByTestId(selectors.corpus.documents)).toBeInTheDocument());
    expect(screen.getByTestId(selectors.corpus.collections)).toBeInTheDocument();
    expect(screen.getAllByTestId(selectors.corpus.locality)).toHaveLength(2);
  });

  it("renders the empty state when the corpus has no records", async () => {
    vi.mocked(listCollections).mockResolvedValue({ collections: [] });
    vi.mocked(listDocuments).mockResolvedValue({ documents: [] });
    renderWithProviders(<CorpusPage />);

    await waitFor(() => expect(screen.getByTestId(selectors.corpus.empty)).toBeInTheDocument());
    expect(screen.getByText(strings.pages.corpus.empty)).toBeInTheDocument();
  });

  it("renders a request error", async () => {
    vi.mocked(listCollections).mockRejectedValue(new Error("corpus unavailable"));
    vi.mocked(listDocuments).mockResolvedValue({ documents: [] });
    renderWithProviders(<CorpusPage />);

    await waitFor(() => expect(screen.getByTestId(selectors.corpus.error)).toHaveTextContent("corpus unavailable"));
  });

  it("uses the generic error copy for an unknown rejection", async () => {
    vi.mocked(listCollections).mockRejectedValue("offline");
    vi.mocked(listDocuments).mockResolvedValue({ documents: [] });
    renderWithProviders(<CorpusPage />);

    await waitFor(() => expect(screen.getByTestId(selectors.corpus.error)).toHaveTextContent("errors.unknown"));
  });
});
