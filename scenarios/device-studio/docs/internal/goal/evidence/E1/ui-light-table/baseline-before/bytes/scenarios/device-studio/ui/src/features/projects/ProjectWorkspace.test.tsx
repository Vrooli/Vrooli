import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

vi.mock("../../api/projects", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/projects")>();
  return { ...actual, projectsClient: { listProjects: vi.fn(), createProject: vi.fn(), getProject: vi.fn(), renameProject: vi.fn() } };
});
import { projectsClient } from "../../api/projects";
import { ProjectWorkspace } from "./ProjectWorkspace";

const page = (projects: Array<{ id: string; name: string; revision: bigint }> = [], nextPageToken = "") => ({ projects, nextPageToken });

describe("ProjectWorkspace", () => {
  beforeEach(async () => { await setLocale("en"); });
  afterEach(() => { cleanup(); vi.clearAllMocks(); });

  it("shows loading, empty and API error states", async () => {
    let resolve!: (value: ReturnType<typeof page>) => void;
    vi.mocked(projectsClient.listProjects).mockReturnValueOnce(new Promise((done) => { resolve = done; }) as never);
    const view = renderWithProviders(<ProjectWorkspace />);
    expect(screen.getByTestId(selectors.projects.surface)).toHaveAttribute("data-experience-state", "loading");
    resolve(page());
    await screen.findByTestId(selectors.projects.empty);
    expect(screen.getByTestId(selectors.projects.surface)).toHaveAttribute("data-experience-state", "empty");
    view.unmount();

    vi.mocked(projectsClient.listProjects).mockRejectedValueOnce(new Error("offline"));
    renderWithProviders(<ProjectWorkspace />);
    expect(await screen.findByTestId(selectors.projects.error)).toHaveTextContent("offline");
    expect(screen.getByTestId(selectors.projects.surface)).toHaveAttribute("data-experience-state", "error");
  });

  it("creates, lists and opens a durable project without clipping long names", async () => {
    const user = userEvent.setup();
    vi.mocked(projectsClient.listProjects).mockResolvedValueOnce(page() as never);
    vi.mocked(projectsClient.listProjects).mockResolvedValue(page([{ id: "p1", name: "A project with a deliberately long name for narrow screens", revision: 1n }]) as never);
    vi.mocked(projectsClient.createProject).mockResolvedValue({ project: { id: "p1", name: "A project with a deliberately long name for narrow screens", revision: 1n } } as never);
    renderWithProviders(<ProjectWorkspace />);
    const input = await screen.findByTestId(selectors.projects.nameInput);
    await user.type(input, "Summer study");
    await user.click(screen.getByTestId(selectors.projects.createButton));
    await waitFor(() => expect(projectsClient.createProject).toHaveBeenCalledWith({ name: "Summer study" }));
    await waitFor(() => expect(screen.getByTestId(selectors.projects.list)).toHaveTextContent("A project with a deliberately long name for narrow screens"));
    expect(screen.getByTestId(selectors.projects.openButton({ id: "p1" }))).toHaveAttribute("href", "/projects/p1");
    expect(screen.getByTestId(selectors.projects.nameInput)).toHaveValue("");
  });
});
