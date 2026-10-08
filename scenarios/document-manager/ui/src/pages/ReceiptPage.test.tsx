import { describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";

import { listDocuments } from "../api/documentManager";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { renderWithProviders } from "../test-utils";
import { ReceiptPage } from "./ReceiptPage";

vi.mock("../api/documentManager", () => ({
  listDocuments: vi.fn(),
}));

// [REQ:DOC-P0-014]
describe("ReceiptPage", () => {
  it("[REQ:DOC-P0-014] renders a local residency summary and processing timeline", async () => {
    vi.mocked(listDocuments).mockResolvedValue({ documents: [{ id: "d1", content_sha256: "sha", source_name: "brief.pdf", detected_mime: "application/pdf", privacy_class: 2 }, { id: "d2", content_sha256: "sha2", source_name: "", detected_mime: "text/plain", privacy_class: 1 }] });
    renderWithProviders(<ReceiptPage />);

    await waitFor(() => expect(screen.getByTestId(selectors.receipt.timeline)).toBeInTheDocument());
    expect(screen.getByTestId(selectors.receipt.residencySummary)).toBeInTheDocument();
  });

  it("renders the empty timeline when no documents are present", async () => {
    vi.mocked(listDocuments).mockResolvedValue({ documents: [] });
    renderWithProviders(<ReceiptPage />);

    await waitFor(() => expect(screen.getByText(strings.pages.receipt.empty)).toBeInTheDocument());
    expect(screen.getByTestId(selectors.receipt.residencySummary)).toBeInTheDocument();
  });

  it("falls back to an empty timeline when loading fails", async () => {
    vi.mocked(listDocuments).mockRejectedValue(new Error("receipt unavailable"));
    renderWithProviders(<ReceiptPage />);

    await waitFor(() => expect(screen.getByText(strings.pages.receipt.empty)).toBeInTheDocument());
  });
});
