import { fireEvent, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import type { SelectHTMLAttributes } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { strings } from "../consts/strings";
import { ContentPage } from "./ContentPage";

const mocks = vi.hoisted(() => ({ listDrafts: vi.fn(), getDraftCurrentRevision: vi.fn(), getDraftRevision: vi.fn(), listDraftAttachments: vi.fn(), attachReleasedAsset: vi.fn(), listProjects: vi.fn(), listCollections: vi.fn(), listCollectionItems: vi.fn(), createCollection: vi.fn(), addCollectionItem: vi.fn(), removeCollectionItem: vi.fn() }));
vi.mock("../api/artifacts", () => ({ artifactsClient: { listDrafts: mocks.listDrafts, getDraftCurrentRevision: mocks.getDraftCurrentRevision, getDraftRevision: mocks.getDraftRevision, listDraftAttachments: mocks.listDraftAttachments, attachReleasedAsset: mocks.attachReleasedAsset } }));
vi.mock("../api/projects", () => ({ projectsClient: { listProjects: mocks.listProjects } }));
vi.mock("../api/relationships", () => ({ relationshipsClient: { listCollections: mocks.listCollections, listCollectionItems: mocks.listCollectionItems, createCollection: mocks.createCollection, addCollectionItem: mocks.addCollectionItem, removeCollectionItem: mocks.removeCollectionItem } }));
type MockSelectProps = SelectHTMLAttributes<HTMLSelectElement> & { options: Array<{ value: string; label: string }> };
vi.mock("@vrooli/react-component-library/Select/1", () => ({
  Select: ({ options, ...props }: MockSelectProps) => <select {...props}>{options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select>,
}));
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

function comboboxAt(index: number): HTMLElement {
  const element = screen.getAllByRole("combobox", { hidden: true })[index];
  if (!element) throw new Error(`missing combobox at index ${index}`);
  return element;
}

function nativeSelectByLabel(label: string): HTMLSelectElement {
  const element = screen.getAllByLabelText(label).find((candidate) => candidate.tagName === "SELECT");
  if (!(element instanceof HTMLSelectElement)) throw new Error(`missing native select for ${label}`);
  return element;
}

function pinRevisionButton(): HTMLElement {
  const element = screen.getAllByRole("button", { name: strings.pages.content.pinCurrentRevision })[0];
  if (!element) throw new Error("missing save revision button");
  return element;
}

describe("ContentPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.listDrafts.mockResolvedValue({ drafts: [
      { id: "draft-one", status: "draft", postTypeId: "dev-log", campaignId: "", projectId: "project-a", channel: "x-twitter", format: "post", body: "A small product note." },
      { id: "draft-two", status: "approved", postTypeId: "case-study", campaignId: "o14", projectId: "unassigned", channel: "linkedin", format: "article", body: "A useful launch story." },
    ] });
    mocks.listProjects.mockResolvedValue({ projects: [{ id: "project-a", name: "Project A" }] });
    mocks.listCollections.mockResolvedValue({ collections: [{ id: "collection-a", projectId: "project-a", name: "Reading list" }] });
    mocks.listCollectionItems.mockResolvedValue({ items: [] });
    mocks.createCollection.mockResolvedValue({ collection: { id: "collection-new", projectId: "project-a", name: "New list" } });
    mocks.getDraftCurrentRevision.mockResolvedValue({ currentRevision: { revisionId: "rev-one", body: "A small product note." } });
    mocks.getDraftRevision.mockResolvedValue({ revision: { draftId: "draft-one", revisionId: "rev-one", body: "Stored exact wording." } });
    mocks.listDraftAttachments.mockResolvedValue({ attachments: [] });
    mocks.attachReleasedAsset.mockResolvedValue({ attachment: { id: "attachment-one" } });
    mocks.addCollectionItem.mockResolvedValue({ item: {} });
    mocks.removeCollectionItem.mockResolvedValue({ item: {} });
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

  it("keeps the persistent Content surface accessible with collection controls", async () => {
    const { container } = renderContent();
    await screen.findByText((content) => content.includes("draft-one"));
    await expectNoA11yViolations(container);
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
    fireEvent.change(nativeSelectByLabel(strings.pages.content.projectLabel), { target: { value: "unassigned" } });
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

  it("creates project collections and pins the exact current revision", async () => {
    renderContent();
    await screen.findByText("draft-one");
    await waitFor(() => expect(nativeSelectByLabel(strings.pages.content.collectionProject)).toHaveValue("project-a"));
    expect(mocks.listCollections).toHaveBeenCalledWith({ projectId: "project-a" });
    fireEvent.change(screen.getByLabelText(strings.pages.content.collectionName), { target: { value: "New list" } });
    fireEvent.click(screen.getByRole("button", { name: strings.pages.content.createCollection }));
    expect(await screen.findByText(strings.pages.content.collectionCreated)).toBeInTheDocument();
    expect(mocks.createCollection).toHaveBeenCalledWith({ projectId: "project-a", name: "New list" });

    const collectionSelect = comboboxAt(3);
    fireEvent.change(collectionSelect, { target: { value: "collection-a" } });
    fireEvent.click(pinRevisionButton());
    expect(await screen.findByText(strings.pages.content.revisionPinned)).toBeInTheDocument();
    expect(mocks.getDraftCurrentRevision).toHaveBeenCalledWith({ id: "draft-one" });
    expect(mocks.addCollectionItem).toHaveBeenCalledWith({ item: { collectionId: "collection-a", draftId: "draft-one", revisionId: "rev-one" } });
    fireEvent.click(await screen.findByRole("button", { name: /viewSavedRevision/ }));
    expect(await screen.findByText("Stored exact wording.")).toBeInTheDocument();
    expect(mocks.getDraftRevision).toHaveBeenCalledWith({ draftId: "draft-one", revisionId: "rev-one" });
  });

  it("does not pin a stale body after reading the exact current revision", async () => {
    mocks.getDraftCurrentRevision.mockResolvedValueOnce({ currentRevision: { revisionId: "rev-new", body: "Changed after list load" } });
    renderContent();
    await screen.findByText("draft-one");
    await waitFor(() => expect(nativeSelectByLabel(strings.pages.content.collectionProject)).toHaveValue("project-a"));
    fireEvent.change(comboboxAt(3), { target: { value: "collection-a" } });
    fireEvent.click(pinRevisionButton());
    expect(await screen.findByText(strings.pages.content.staleRevision)).toBeInTheDocument();
    expect(mocks.addCollectionItem).not.toHaveBeenCalled();
  });

  it("does not write duplicate exact-revision membership", async () => {
    mocks.listCollectionItems.mockResolvedValueOnce({ items: [{ collectionId: "collection-a", draftId: "draft-one", revisionId: "rev-one" }] });
    renderContent();
    await screen.findByText("draft-one");
    await waitFor(() => expect(nativeSelectByLabel(strings.pages.content.collectionProject)).toHaveValue("project-a"));
    fireEvent.change(comboboxAt(3), { target: { value: "collection-a" } });
    fireEvent.click(pinRevisionButton());
    expect(await screen.findByText(strings.pages.content.revisionAlreadyPinned)).toBeInTheDocument();
    expect(mocks.addCollectionItem).not.toHaveBeenCalled();
  });

  it("filters the durable Content list by a saved collection and preserves URL scope", async () => {
    mocks.listCollectionItems.mockResolvedValueOnce({ items: [{ collectionId: "collection-a", draftId: "draft-one", revisionId: "rev-one" }] });
    renderContent();
    await screen.findByText("draft-two");
    await waitFor(() => expect(screen.getByText(/Reading list · 1/)).toBeInTheDocument());
    fireEvent.change(nativeSelectByLabel(strings.pages.content.collectionFilter), { target: { value: "collection-a" } });
    expect(screen.getByText("draft-one")).toBeInTheDocument();
    expect(screen.queryByText("draft-two")).not.toBeInTheDocument();
    expect(screen.getByTestId("location")).toHaveTextContent("collection=collection-a");
  });

  it("removes only the saved revision reference and retains its draft", async () => {
    mocks.listCollectionItems.mockResolvedValueOnce({ items: [{ collectionId: "collection-a", draftId: "draft-one", revisionId: "rev-one" }] });
    renderContent();
    await screen.findByText("draft-one");
    fireEvent.click(await screen.findByRole("button", { name: strings.pages.content.removeSavedRevision }));
    expect(await screen.findByText(strings.pages.content.revisionRemoved)).toBeInTheDocument();
    expect(mocks.removeCollectionItem).toHaveBeenCalledWith({ item: { collectionId: "collection-a", draftId: "draft-one", revisionId: "rev-one" } });
    expect(screen.getByText("draft-one")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /viewSavedRevision/ })).not.toBeInTheDocument();
  });

  it("preserves released Asset Studio reference attachment in Content", async () => {
    renderContent();
    await screen.findByText("draft-one");
    const manageReferences = screen.getAllByRole("button", { name: strings.pages.content.manageAssetReferences })[0];
    if (!manageReferences) throw new Error("missing Asset Studio reference control");
    fireEvent.click(manageReferences);
    expect(await screen.findByText(strings.pages.content.noAssetReferences)).toBeInTheDocument();
    expect(mocks.listDraftAttachments).toHaveBeenCalledWith({ draftId: "draft-one" });
    fireEvent.change(screen.getByLabelText(strings.pages.content.assetId), { target: { value: "asset-42" } });
    fireEvent.change(screen.getByLabelText(strings.pages.content.assetRole), { target: { value: "hero" } });
    fireEvent.change(screen.getByLabelText(strings.pages.content.assetAspectRatio), { target: { value: "16:9" } });
    fireEvent.change(screen.getByLabelText(strings.pages.content.assetPosition), { target: { value: "1.5" } });
    expect(screen.getByRole("button", { name: strings.pages.content.attachReleasedAsset })).toBeDisabled();
    fireEvent.change(screen.getByLabelText(strings.pages.content.assetPosition), { target: { value: "2" } });
    fireEvent.change(screen.getByLabelText(strings.pages.content.assetAltText), { target: { value: "Product screenshot" } });
    fireEvent.click(screen.getByRole("button", { name: strings.pages.content.attachReleasedAsset }));
    expect(await screen.findByText(strings.pages.content.assetAttached)).toBeInTheDocument();
    expect(mocks.attachReleasedAsset).toHaveBeenCalledWith({ draftId: "draft-one", assetId: "asset-42", role: "hero", aspectRatio: "16:9", altText: "Product screenshot", position: 2 });
  });

  it("does not report attachment confirmation when the post-save list refresh fails", async () => {
    mocks.listDraftAttachments.mockResolvedValueOnce({ attachments: [] }).mockRejectedValueOnce(new Error("refresh failed"));
    renderContent();
    await screen.findByText((content) => content.includes("draft-one"));
    const manageReferences = screen.getAllByRole("button", { name: strings.pages.content.manageAssetReferences })[0];
    if (!manageReferences) throw new Error("missing Asset Studio reference control");
    fireEvent.click(manageReferences);
    await screen.findByText(strings.pages.content.noAssetReferences);
    fireEvent.change(screen.getByLabelText(strings.pages.content.assetId), { target: { value: "asset-42" } });
    fireEvent.change(screen.getByLabelText(strings.pages.content.assetAltText), { target: { value: "Product screenshot" } });
    fireEvent.click(screen.getByRole("button", { name: strings.pages.content.attachReleasedAsset }));
    expect(await screen.findByRole("alert")).toHaveTextContent(strings.pages.content.assetLoadError);
    expect(screen.queryByText(strings.pages.content.assetAttached)).not.toBeInTheDocument();
  });
});
