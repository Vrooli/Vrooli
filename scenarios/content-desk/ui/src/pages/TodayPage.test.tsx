import { fireEvent, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "@vrooli/api-base/testing";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { TodayPage } from "./TodayPage";

const mocks = vi.hoisted(() => ({ listDrafts: vi.fn() }));

vi.mock("../api/artifacts", () => ({ artifactsClient: { listDrafts: mocks.listDrafts } }));

function SelectedDraft() {
  const location = useLocation();
  const selected = new URLSearchParams(location.search).get("draft");
  return <p>Selected draft: {selected}</p>;
}

function renderToday() {
  return renderWithProviders(
    <MemoryRouter initialEntries={["/today"]} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
      <Routes>
        <Route path="/today" element={<TodayPage />} />
        <Route path="/content" element={<p>Content collection</p>} />
        <Route path="/" element={<SelectedDraft />} />
        <Route path="/review" element={<SelectedDraft />} />
      </Routes>
    </MemoryRouter>,
    { withoutRouter: true },
  );
}

describe("TodayPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.listDrafts.mockResolvedValue({ drafts: [] });
  });

  it("shows a quiet workspace without inventing a schedule", async () => {
    renderToday();
    expect(await screen.findByText(strings.today.quietTitle)).toBeInTheDocument();
    expect(screen.getByText(strings.today.quietDescription)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: strings.today.viewWork }));
    expect(await screen.findByText("Content collection")).toBeInTheDocument();
  });

  it("resumes the exact native draft identity", async () => {
    mocks.listDrafts.mockResolvedValue({ drafts: [{ id: "draft-native-12", status: "drafted", campaignId: "", channel: "LinkedIn", body: "" }] });
    renderToday();

    fireEvent.click(await screen.findByRole("button", { name: strings.today.resumeButton }));
    expect(await screen.findByText("Selected draft: draft-native-12")).toBeInTheDocument();
  });

  it("opens the exact reviewed draft for the operator decision", async () => {
    mocks.listDrafts.mockResolvedValue({ drafts: [{ id: "draft-awaiting-operator", status: "reviewed", campaignId: "campaign-2", channel: "LinkedIn", body: "" }] });
    renderToday();

    expect(await screen.findByText(strings.today.reviewDescription)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: strings.today.reviewWorkbench }));
    expect(await screen.findByText("Selected draft: draft-awaiting-operator")).toBeInTheDocument();
  });

  it("keeps approved drafts out of the waiting list and marks them release-ready", async () => {
    mocks.listDrafts.mockResolvedValue({ drafts: [{ id: "draft-approved", status: "approved", campaignId: "", channel: "x-twitter", body: "Ready" }] });
    renderToday();

    expect(await screen.findByText(strings.today.releaseReadyTitle)).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: strings.today.waiting })).toBeInTheDocument();
    expect(screen.getAllByText("draft-approved")).toHaveLength(1);
  });

  it("distinguishes an unavailable native draft read from an empty workspace", async () => {
    mocks.listDrafts.mockRejectedValue(new Error("backend unavailable"));
    renderToday();

    expect(await screen.findByText(strings.today.unavailableTitle)).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent("backend unavailable");
    expect(screen.queryByText(strings.today.quietTitle)).not.toBeInTheDocument();
    expect(screen.queryByText(strings.today.releaseReadyTitle)).not.toBeInTheDocument();
    expect(screen.getByTestId(selectors.pages.today)).toBeInTheDocument();
  });

  it("hides stale release-ready records after a failed refresh", async () => {
    mocks.listDrafts
      .mockResolvedValueOnce({ drafts: [{ id: "stale-approved", status: "approved", campaignId: "", channel: "x-twitter", body: "Ready" }] })
      .mockRejectedValueOnce(new Error("refresh failed"));
    renderToday();

    expect(await screen.findByText(strings.today.releaseReadyTitle)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: strings.today.refresh }));
    expect(await screen.findByText(strings.today.unavailableTitle)).toBeInTheDocument();
    expect(screen.queryByText(strings.today.releaseReadyTitle)).not.toBeInTheDocument();
    expect(screen.getByText(strings.today.openItems)).toBeInTheDocument();
  });
});
