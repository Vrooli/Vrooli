import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "@vrooli/api-base/testing";
import { strings } from "../consts/strings";
import { LearnPage } from "./LearnPage";

const mocks = vi.hoisted(() => ({ listPublishRecords: vi.fn(), getDraftMetrics: vi.fn() }));
vi.mock("../api/ledger", () => ({ ledgerClient: { listPublishRecords: mocks.listPublishRecords, getDraftMetrics: mocks.getDraftMetrics } }));

describe("LearnPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.listPublishRecords.mockResolvedValue({ publishRecords: [
      { id: "record-measured", draftId: "draft-measured", publishedUrl: "https://example.test/post", platformPostId: "post-1" },
      { id: "record-unknown", draftId: "draft-unknown", publishedUrl: "", platformPostId: "post-2" },
    ] });
    mocks.getDraftMetrics.mockImplementation(({ draftId }: { draftId: string }) => Promise.resolve(draftId === "draft-measured"
      ? { hasMeasurements: true, readings: [{ metric: "impressions", value: 0, state: "measured", sampleCount: 1, lastObservedAt: "2026-11-20T10:00:00Z" }] }
      : { hasMeasurements: false, readings: [] }));
  });

  it("keeps a measured zero distinct from a missing observation", async () => {
    renderWithProviders(<LearnPage />, { withoutRouter: true });
    expect(await screen.findByText("draft-measured")).toBeInTheDocument();
    expect(screen.getByText("impressions: 0")).toBeInTheDocument();
    expect(screen.getByText(strings.pages.learn.unknownMeasurements)).toBeInTheDocument();
    expect(mocks.getDraftMetrics).toHaveBeenCalledWith({ draftId: "draft-measured", staleAfterDays: 30 });
    expect(mocks.getDraftMetrics).toHaveBeenCalledWith({ draftId: "draft-unknown", staleAfterDays: 30 });
  });

  it("keeps per-item measurement errors unknown and shows empty and load-error states", async () => {
    mocks.listPublishRecords.mockResolvedValueOnce({ publishRecords: [{ id: "record-error", draftId: "draft-error", publishedUrl: "", platformPostId: "post-error" }] });
    mocks.getDraftMetrics.mockRejectedValueOnce(new Error("metric API offline"));
    renderWithProviders(<LearnPage />, { withoutRouter: true });
    expect(await screen.findByText(strings.pages.learn.measurementsUnavailable)).toBeInTheDocument();
    cleanup();

    mocks.listPublishRecords.mockResolvedValueOnce({ publishRecords: [] });
    renderWithProviders(<LearnPage />, { withoutRouter: true });
    expect(await screen.findByText(strings.pages.learn.emptyTitle)).toBeInTheDocument();
    cleanup();

    mocks.listPublishRecords.mockRejectedValueOnce(new Error("ledger offline")).mockResolvedValueOnce({ publishRecords: [] });
    renderWithProviders(<LearnPage />, { withoutRouter: true });
    expect(await screen.findByRole("alert")).toHaveTextContent("ledger offline");
    fireEvent.click(screen.getByRole("button", { name: strings.pages.learn.retry }));
    await waitFor(() => expect(screen.getByText(strings.pages.learn.emptyTitle)).toBeInTheDocument());
  });
});
