import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { Route, Routes } from "react-router-dom";
import { RecordDetailsPage } from "./RecordDetailsPage";
import { renderWithProviders } from "../test-utils";
import { recordsService } from "../services/records-service";
import type { RecordItem } from "../types";

function makeRecord(overrides: Partial<RecordItem> = {}): RecordItem {
  return {
    id: "rec-1",
    kind: "fix",
    scenario: "web-console",
    backlogRef: "fix/silence-race",
    milestoneId: "voice-reliability",
    supersedes: "rec-old",
    supersededBy: "rec-new",
    trigger: "voice auto-stop silence race",
    approach: "debounce the VAD events",
    ruledOut: [],
    filesChanged: [],
    outcome: "shipped",
    stub: false,
    createdAt: "2026-06-09T12:00:00Z",
    ...overrides,
  };
}

function renderPage(id = "rec-1") {
  return renderWithProviders(
    <Routes>
      <Route path="/records/:recordId" element={<RecordDetailsPage />} />
    </Routes>,
    { initialEntries: [`/records/${id}`] },
  );
}

describe("RecordDetailsPage reference links", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("renders backlog, milestone, and supersede refs as in-app links to the right routes", async () => {
    vi.spyOn(recordsService, "get").mockResolvedValue(makeRecord());
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("record-backlog-link")).toBeInTheDocument();
    });
    expect(screen.getByTestId("record-backlog-link")).toHaveAttribute("href", "/backlog/fix/silence-race");
    expect(screen.getByTestId("record-milestone-link")).toHaveAttribute("href", "/milestones/voice-reliability");
    expect(screen.getByTestId("record-supersedes-link")).toHaveAttribute("href", "/records/rec-old");
    expect(screen.getByTestId("record-superseded-by-link")).toHaveAttribute("href", "/records/rec-new");
  });

  it("omits the milestone link when the record has no milestone", async () => {
    vi.spyOn(recordsService, "get").mockResolvedValue(makeRecord({ milestoneId: undefined }));
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("record-backlog-link")).toBeInTheDocument();
    });
    expect(screen.queryByTestId("record-milestone-link")).toBeNull();
  });
});
