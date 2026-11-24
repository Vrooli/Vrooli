import { screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";

import { getWorkspace, listWorkspaces } from "../../api/workspaces";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import { useWorkspace, useWorkspaces } from "./useWorkspaces";
import type { Workspace, WorkspaceSummary } from "../../api/workspaces";

vi.mock("../../api/workspaces", () => ({
  getWorkspace: vi.fn(),
  listWorkspaces: vi.fn(),
}));

function WorkspaceListProbe() {
  const query = useWorkspaces();
  return <output>{query.error?.message ?? query.data?.length ?? "loading"}</output>;
}

function WorkspaceProbe({ id }: { id: string | null }) {
  const query = useWorkspace(id);
  return (
    <output>
      {query.error?.message ?? query.data?.id ?? (id ? "loading" : query.fetchStatus)}
    </output>
  );
}

describe("workspace query hooks", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("loads workspace summaries with the shared list limit", async () => {
    vi.mocked(listWorkspaces).mockResolvedValue([{ id: "harbor" }] as WorkspaceSummary[]);

    renderWithProviders(<WorkspaceListProbe />);

    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("1"));
    expect(listWorkspaces).toHaveBeenCalledWith({ limit: 200 });
  });

  it("loads one workspace by id", async () => {
    vi.mocked(getWorkspace).mockResolvedValue({ id: "harbor" } as Workspace);

    renderWithProviders(<WorkspaceProbe id="harbor" />);

    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("harbor"));
    expect(getWorkspace).toHaveBeenCalledWith("harbor");
  });

  it("surfaces a failed workspace list request without retrying", async () => {
    vi.mocked(listWorkspaces).mockRejectedValue(new Error("workspace list unavailable"));

    renderWithProviders(<WorkspaceListProbe />);

    await waitFor(() =>
      expect(screen.getByRole("status")).toHaveTextContent("workspace list unavailable"),
    );
    expect(listWorkspaces).toHaveBeenCalledTimes(1);
  });

  it("surfaces a failed selected workspace request", async () => {
    vi.mocked(getWorkspace).mockRejectedValue(new Error("workspace unavailable"));

    renderWithProviders(<WorkspaceProbe id="missing" />);

    await waitFor(() =>
      expect(screen.getByRole("status")).toHaveTextContent("workspace unavailable"),
    );
    expect(getWorkspace).toHaveBeenCalledTimes(1);
  });

  it("keeps the query idle when no workspace id is selected", async () => {
    renderWithProviders(<WorkspaceProbe id={null} />);

    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("idle"));
    expect(getWorkspace).not.toHaveBeenCalled();
  });
});
