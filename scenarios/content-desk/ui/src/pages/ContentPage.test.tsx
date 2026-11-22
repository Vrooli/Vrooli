import { fireEvent, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "@vrooli/api-base/testing";
import { strings } from "../consts/strings";
import { ContentPage } from "./ContentPage";

const mocks = vi.hoisted(() => ({ listDrafts: vi.fn(), listProjects: vi.fn() }));
vi.mock("../api/artifacts", () => ({ artifactsClient: { listDrafts: mocks.listDrafts } }));
vi.mock("../api/projects", () => ({ projectsClient: { listProjects: mocks.listProjects } }));
function Location() { const location = useLocation(); return <output data-testid="location">{location.pathname}{location.search}</output>; }

function renderContent(initialEntry = "/content") {
  return renderWithProviders(
    <MemoryRouter initialEntries={[initialEntry]} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
      <Routes>
        <Route path="/content" element={<ContentPage />} />
        <Route path="/review" element={<div data-testid="page-review" />} />
      </Routes>
      <Location />
    </MemoryRouter>,
    { withoutRouter: true },
  );
}

describe("ContentPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.listDrafts.mockResolvedValue({ drafts: [
      { id: "draft-one", status: "draft", postTypeId: "dev-log", campaignId: "", projectId: "project-a", channel: "x-twitter", format: "post", body: "A small product note." },
      { id: "draft-two", status: "approved", postTypeId: "case-study", campaignId: "o14", projectId: "unassigned", channel: "linkedin", format: "article", body: "A useful launch story." },
    ] });
    mocks.listProjects.mockResolvedValue({ projects: [{ id: "project-a", name: "Project A" }] });
  });

  it("loads the native draft collection and filters by text", async () => {
    renderContent();
    expect(await screen.findByText("draft-one")).toBeInTheDocument();
    expect(screen.getByText("draft-two")).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText(strings.pages.content.searchLabel), { target: { value: "launch" } });
    expect(screen.queryByText("draft-one")).not.toBeInTheDocument();
    expect(screen.getByText("draft-two")).toBeInTheDocument();
    expect(mocks.listDrafts).toHaveBeenCalledWith({});
  });

  it("filters status and opens an exact draft in Review", async () => {
    renderContent();
    await screen.findByText("draft-one");
    fireEvent.click(screen.getByRole("button", { name: strings.pages.content.status.approved }));
    expect(screen.queryByText("draft-one")).not.toBeInTheDocument();
    expect(screen.getByText("draft-two")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: strings.pages.content.open }));
    expect(await screen.findByTestId("page-review")).toBeInTheDocument();
    expect(screen.getByTestId("location")).toHaveTextContent("/review?draft=draft-two&return=%2Fcontent%3Fstatus%3Dapproved");
  });

  it("filters by project and keeps search, project, and status in the URL", async () => {
    renderContent();
    await screen.findByText("draft-one");
    fireEvent.click(screen.getByRole("button", { name: strings.pages.content.status.approved }));
    fireEvent.change(screen.getByLabelText(strings.pages.content.projectLabel), { target: { value: "unassigned" } });
    fireEvent.change(screen.getByRole("searchbox"), { target: { value: "launch" } });
    expect(screen.queryByText("draft-one")).not.toBeInTheDocument();
    expect(screen.getByText("draft-two")).toBeInTheDocument();
    expect(screen.getByTestId("location")).toHaveTextContent("status=approved");
    expect(screen.getByTestId("location")).toHaveTextContent("project=unassigned");
    expect(screen.getByTestId("location")).toHaveTextContent("q=launch");
  });

  it("shows the empty and no-match states without inventing records", async () => {
    mocks.listDrafts.mockResolvedValueOnce({ drafts: [] });
    const { unmount } = renderContent();
    expect(await screen.findByText(strings.pages.content.emptyTitle)).toBeInTheDocument();
    unmount();

    mocks.listDrafts.mockResolvedValueOnce({ drafts: [{ id: "saved-draft", status: "draft", postTypeId: "dev-log", body: "Known note." }] });
    renderContent();
    await screen.findByText("saved-draft");
    fireEvent.change(screen.getByLabelText(strings.pages.content.searchLabel), { target: { value: "no matching text" } });
    expect(screen.getByText(strings.pages.content.noMatchTitle)).toBeInTheDocument();
  });

  it("reports a load error and retries the native collection read", async () => {
    mocks.listDrafts.mockRejectedValueOnce(new Error("draft service offline")).mockResolvedValueOnce({ drafts: [] });
    renderContent();
    expect(await screen.findByRole("alert")).toHaveTextContent("draft service offline");
    fireEvent.click(screen.getByRole("button", { name: strings.pages.content.retry }));
    await waitFor(() => expect(screen.getByText(strings.pages.content.emptyTitle)).toBeInTheDocument());
    expect(mocks.listDrafts).toHaveBeenCalledTimes(2);
  });
});
