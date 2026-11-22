import { fireEvent, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "@vrooli/api-base/testing";
import { strings } from "../consts/strings";
import { CreatePage } from "./CreatePage";

const mocks = vi.hoisted(() => ({ listPostTypes: vi.fn(), createDraft: vi.fn(), listProjects: vi.fn(), createProject: vi.fn() }));
vi.mock("../api/posttypes", () => ({ posttypesClient: { listPostTypes: mocks.listPostTypes } }));
vi.mock("../api/artifacts", () => ({ artifactsClient: { createDraft: mocks.createDraft } }));
vi.mock("../api/projects", () => ({ projectsClient: { listProjects: mocks.listProjects, createProject: mocks.createProject } }));
vi.mock("@vrooli/react-component-library/Select/1", () => ({
  Select: ({ value, onChange, options, required }: { value: string; onChange: (event: React.ChangeEvent<HTMLSelectElement>) => void; options: Array<{ value: string; label: string }>; required?: boolean }) => <select aria-label={strings.pages.create.postType} value={value} onChange={onChange} required={required}>{options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select>,
}));
vi.mock("@vrooli/react-component-library/Textarea/1", () => ({
  Textarea: ({ value, onChange, rows, className }: { value: string; onChange: (event: React.ChangeEvent<HTMLTextAreaElement>) => void; rows?: number; className?: string }) => <textarea aria-label={strings.pages.create.body} value={value} onChange={onChange} rows={rows} className={className} />,
}));

function renderCreate() {
  return renderWithProviders(
    <MemoryRouter initialEntries={["/create"]} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
      <CreatePage />
    </MemoryRouter>,
    { withoutRouter: true },
  );
}

describe("CreatePage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.listPostTypes.mockResolvedValue({ postTypes: [{ id: "dev-log", status: "active" }, { id: "stub", status: "v0" }] });
    mocks.listProjects.mockResolvedValue({ projects: [{ id: "project-o14", name: "O14" }] });
    mocks.createDraft.mockResolvedValue({ draft: { id: "saved-native-draft" } });
    mocks.createProject.mockResolvedValue({ project: { id: "project-new", name: "New project" } });
  });

  it("saves an exact campaign-free native draft without invoking release actions", async () => {
    renderCreate();
    const input = await screen.findByLabelText(strings.pages.create.body);
    fireEvent.change(input, { target: { value: "A short observation." } });
    fireEvent.click(screen.getByRole("button", { name: strings.pages.create.save }));

    await waitFor(() => expect(mocks.createDraft).toHaveBeenCalledWith({
      campaignId: "",
      projectId: "",
      postTypeId: "dev-log",
      body: "A short observation.",
      channel: "x-twitter",
      format: "post",
    }));
    expect(await screen.findByText(strings.pages.create.savedDetail)).toBeInTheDocument();
    expect(mocks.listPostTypes).toHaveBeenCalledOnce();
  });

  it("creates a named project and saves the draft into it", async () => {
    renderCreate();
    await screen.findByLabelText(strings.pages.create.project);
    fireEvent.change(screen.getByLabelText(strings.pages.create.newProjectName), { target: { value: "  New project  " } });
    fireEvent.click(screen.getByRole("button", { name: strings.pages.create.createProject }));
    await waitFor(() => expect(mocks.createProject).toHaveBeenCalledWith({ name: "New project" }));
    fireEvent.change(screen.getByLabelText(strings.pages.create.body), { target: { value: "A project note." } });
    fireEvent.click(screen.getByRole("button", { name: strings.pages.create.save }));
    await waitFor(() => expect(mocks.createDraft).toHaveBeenCalledWith(expect.objectContaining({ projectId: "project-new" })));
  });

  it("does not invent a post type when the active registry is empty", async () => {
    mocks.listPostTypes.mockResolvedValue({ postTypes: [{ id: "stub", status: "v0" }] });
    renderCreate();
    expect(await screen.findByText(strings.pages.create.noTypes)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: strings.pages.create.save })).not.toBeInTheDocument();
    expect(mocks.createDraft).not.toHaveBeenCalled();
  });

  it("keeps an empty simple post from being saved", async () => {
    renderCreate();
    await screen.findByLabelText(strings.pages.create.body);
    expect(screen.getByRole("button", { name: strings.pages.create.save })).toBeDisabled();
    expect(mocks.createDraft).not.toHaveBeenCalled();
  });

  it("keeps Unassigned available when projects cannot be loaded", async () => {
    mocks.listProjects.mockRejectedValueOnce(new Error("project API offline"));
    renderCreate();
    expect(await screen.findByText(strings.pages.create.projectLoadError)).toBeInTheDocument();
    expect(screen.getByLabelText(strings.pages.create.project)).toBeInTheDocument();
  });

  it("shows project creation and draft save errors without implying success", async () => {
    mocks.createProject.mockRejectedValueOnce(new Error("project rejected"));
    mocks.createDraft.mockRejectedValueOnce(new Error("save rejected"));
    renderCreate();
    await screen.findByLabelText(strings.pages.create.project);
    fireEvent.change(screen.getByLabelText(strings.pages.create.newProjectName), { target: { value: "New project" } });
    fireEvent.click(screen.getByRole("button", { name: strings.pages.create.createProject }));
    expect(await screen.findByRole("alert")).toHaveTextContent("project rejected");

    fireEvent.change(screen.getByLabelText(strings.pages.create.body), { target: { value: "A short observation." } });
    fireEvent.click(screen.getByRole("button", { name: strings.pages.create.save }));
    expect(await screen.findByText("save rejected")).toBeInTheDocument();
  });

  it("shows a post-type load failure", async () => {
    mocks.listPostTypes.mockRejectedValueOnce(new Error("post type service offline"));
    renderCreate();
    expect(await screen.findByRole("alert")).toHaveTextContent("post type service offline");
    expect(mocks.createDraft).not.toHaveBeenCalled();
  });
});
