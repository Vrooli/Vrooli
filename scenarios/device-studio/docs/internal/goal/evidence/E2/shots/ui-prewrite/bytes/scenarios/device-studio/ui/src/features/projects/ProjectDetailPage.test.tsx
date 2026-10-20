import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ConnectError, Code } from "@connectrpc/connect";
import { ProjectConflictSchema } from "@vrooli/proto-types/device-studio/v1/projects/projects_pb";
import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { TestAppRouter } from "../../app/routes";
import { Link, MemoryRouter, Route, Routes } from "react-router-dom";
import { ProjectDetailPage } from "./ProjectDetailPage";

vi.mock("../../api/projects", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/projects")>();
  return { ...actual, projectsClient: { listProjects: vi.fn(), createProject: vi.fn(), getProject: vi.fn(), renameProject: vi.fn(), listSourceMedia: vi.fn().mockResolvedValue({ sourceMedia: [] }), getSourceMedia: vi.fn() }, uploadSourceMedia: vi.fn(), getSourceMediaContent: vi.fn() };
});
import { projectsClient } from "../../api/projects";

const project = (name: string, revision: bigint) => ({ id: "p1", name, revision, createdAt: undefined, updatedAt: undefined });
describe("ProjectDetailPage", () => {
  beforeEach(async () => { await setLocale("en"); });
  afterEach(() => { cleanup(); vi.clearAllMocks(); });

  it("shows loading and get errors", async () => {
    let reject!: (error: Error) => void;
    vi.mocked(projectsClient.getProject).mockReturnValueOnce(new Promise((_, fail) => { reject = fail; }) as never);
    renderWithProviders(<TestAppRouter initialEntries={["/projects/p1"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.projects.loading)).toBeInTheDocument();
    reject(new Error("project unavailable"));
    expect(await screen.findByTestId(selectors.projects.error)).toHaveTextContent("project unavailable");
  });

  it("uses the revision captured when editing began after a background refetch", async () => {
    const user = userEvent.setup();
    vi.mocked(projectsClient.getProject).mockResolvedValue({ project: project("Original", 1n) } as never);
    vi.mocked(projectsClient.renameProject).mockResolvedValue({ project: project("My draft", 3n) } as never);
    const view = renderWithProviders(<TestAppRouter initialEntries={["/projects/p1"]} />, { withoutRouter: true });
    const input = await screen.findByTestId(selectors.projects.renameInput);
    await user.clear(input);
    await user.type(input, "My draft");
    view.queryClient.setQueryData(["project", "p1"], { project: project("Remote update", 2n) });
    await screen.findByText("Revision 2");
    await user.click(screen.getByTestId(selectors.projects.renameButton));
    await waitFor(() => expect(projectsClient.renameProject).toHaveBeenCalledWith({ id: "p1", name: "My draft", expectedRevision: 1n }));
  });

  it("clears the draft and pins the new project's revision after a route change", async () => {
    const user = userEvent.setup();
    vi.mocked(projectsClient.getProject)
      .mockResolvedValueOnce({ project: project("First project", 1n) } as never)
      .mockResolvedValueOnce({ project: { ...project("Second project", 7n), id: "p2" } } as never);
    let resolveFirstRename!: (value: { project: ReturnType<typeof project> }) => void;
    vi.mocked(projectsClient.renameProject)
      .mockReturnValueOnce(new Promise((resolve) => { resolveFirstRename = resolve; }) as never)
      .mockResolvedValueOnce({ project: { ...project("Second renamed", 8n), id: "p2" } } as never);
    renderWithProviders(
      <MemoryRouter initialEntries={["/projects/p1"]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
        <Link data-testid="switch-project" to="/projects/p2">Switch project</Link>
        <Routes><Route path="/projects/:projectId" element={<ProjectDetailPage />} /></Routes>
      </MemoryRouter>,
      { withoutRouter: true },
    );
    const input = await screen.findByTestId(selectors.projects.renameInput);
    await user.clear(input);
    await user.type(input, "Unsaved first draft");
    await user.click(screen.getByTestId(selectors.projects.renameButton));
    expect(projectsClient.renameProject).toHaveBeenCalledWith({ id: "p1", name: "Unsaved first draft", expectedRevision: 1n });
    await user.click(screen.getByTestId("switch-project"));
    await waitFor(() => expect(screen.getByTestId(selectors.projects.renameInput)).toHaveValue("Second project"));
    resolveFirstRename({ project: project("Late first rename", 2n) });
    await waitFor(() => expect(screen.getByTestId(selectors.projects.renameInput)).toHaveValue("Second project"));
    expect(screen.queryByTestId(selectors.projects.conflict)).not.toBeInTheDocument();
    await user.clear(screen.getByTestId(selectors.projects.renameInput));
    await user.type(screen.getByTestId(selectors.projects.renameInput), "Second renamed");
    await user.click(screen.getByTestId(selectors.projects.renameButton));
    await waitFor(() => expect(projectsClient.renameProject).toHaveBeenCalledWith({ id: "p2", name: "Second renamed", expectedRevision: 7n }));
  });

  it("retains the attempted name through a typed revision conflict and reloads current revision", async () => {
    const user = userEvent.setup();
    vi.mocked(projectsClient.getProject).mockResolvedValue({ project: project("Original", 1n) } as never);
    const conflict = new ConnectError("project changed", Code.Aborted, undefined, [{ desc: ProjectConflictSchema, value: { projectId: "p1", expectedRevision: 1n, currentRevision: 2n } }]);
    vi.mocked(projectsClient.renameProject)
      .mockRejectedValueOnce(conflict)
      .mockResolvedValueOnce({ project: project("Attempted name", 3n) } as never)
      .mockResolvedValueOnce({ project: project("Attempted again", 4n) } as never);
    renderWithProviders(<TestAppRouter initialEntries={["/projects/p1"]} />, { withoutRouter: true });
    const input = await screen.findByTestId(selectors.projects.renameInput);
    await user.clear(input);
    await user.type(input, "Attempted name");
    await user.click(screen.getByTestId(selectors.projects.renameButton));
    expect(await screen.findByTestId(selectors.projects.conflict)).toBeInTheDocument();
    expect(input).toHaveValue("Attempted name");
    vi.mocked(projectsClient.getProject).mockResolvedValueOnce({ project: project("Remote name", 2n) } as never);
    await user.click(screen.getByTestId(selectors.projects.reloadButton));
    await waitFor(() => expect(screen.getByTestId(selectors.projects.renameInput)).toHaveValue("Attempted name"));
    vi.mocked(projectsClient.getProject).mockResolvedValueOnce({ project: project("Attempted name", 3n) } as never);
    await user.click(screen.getByTestId(selectors.projects.renameButton));
    await waitFor(() => expect(projectsClient.renameProject).toHaveBeenLastCalledWith({ id: "p1", name: "Attempted name", expectedRevision: 2n }));
    await waitFor(() => expect(screen.getByTestId(selectors.projects.renameInput)).toHaveValue("Attempted name"));
    await user.clear(screen.getByTestId(selectors.projects.renameInput));
    await user.type(screen.getByTestId(selectors.projects.renameInput), "Attempted again");
    await user.click(screen.getByTestId(selectors.projects.renameButton));
    await waitFor(() => expect(projectsClient.renameProject).toHaveBeenLastCalledWith({ id: "p1", name: "Attempted again", expectedRevision: 3n }));
  });
});
