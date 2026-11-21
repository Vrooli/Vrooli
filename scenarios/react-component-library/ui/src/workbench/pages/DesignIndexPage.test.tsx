import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import type { UseQueryResult } from "@tanstack/react-query";

import { renderWithProviders } from "../../test-utils";

vi.mock("../workspaces/useWorkspaces", () => ({
  useWorkspaces: vi.fn(),
}));

vi.mock("../../api/workspaces", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/workspaces")>();
  return {
    ...actual,
    forkWorkspace: vi.fn(),
  };
});

import {
  WorkspaceProvenance,
  type WorkspaceSummary,
} from "../../api/workspaces";
import { strings } from "../../consts/strings";
import * as workspaceHook from "../workspaces/useWorkspaces";
import { DesignIndexPage } from "./DesignIndexPage";

function queryResult<T>(
  value: T | undefined,
  options: { isLoading?: boolean; isError?: boolean } = {},
): UseQueryResult<T> {
  return {
    data: value,
    error: options.isError ? new Error("unavailable") : null,
    isError: options.isError ?? false,
    isLoading: options.isLoading ?? false,
    isFetching: false,
    isSuccess: !options.isError && value !== undefined,
    refetch: vi.fn(),
  } as unknown as UseQueryResult<T>;
}

const WORKSPACE_NAME = "Harbor Workspace";
const CURATED_NAME = "Curated onboarding";

const workspaces: WorkspaceSummary[] = [
  {
    $typeName: "vrooli.react_component_library.v1.workspace.WorkspaceSummary",
    id: "ws_harbor",
    name: WORKSPACE_NAME,
    summary: "Four-step onboarding",
    provenance: WorkspaceProvenance.AUTHORED,
    tags: [],
    bindingScenario: "harbor",
    bindingPage: "welcome",
    pageCount: 4,
    displayCount: 12,
    stateCount: 16,
    revision: 2,
    createdAt: "2026-11-18T10:00:00Z",
    updatedAt: "2026-11-18T12:00:00Z",
    originWorkspaceId: "",
  },
  {
    $typeName: "vrooli.react_component_library.v1.workspace.WorkspaceSummary",
    id: "ws_curated",
    name: CURATED_NAME,
    summary: "",
    provenance: WorkspaceProvenance.CURATED,
    tags: [],
    bindingScenario: "",
    bindingPage: "",
    pageCount: 2,
    displayCount: 6,
    stateCount: 8,
    revision: 1,
    createdAt: "2026-11-17T10:00:00Z",
    updatedAt: "2026-11-17T12:00:00Z",
    originWorkspaceId: "",
  },
];

describe("DesignIndexPage", () => {
  beforeEach(() => {
    vi.mocked(workspaceHook.useWorkspaces).mockReturnValue(queryResult(workspaces));
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("lists authored workspaces with counts and studio open actions", async () => {
    renderWithProviders(<DesignIndexPage />);

    expect(await screen.findByText(WORKSPACE_NAME)).toBeInTheDocument();
    expect(screen.getAllByTestId("design-workspace-row")).toHaveLength(1);
    expect(screen.getByText(/12 displays/)).toBeInTheDocument();

    const openLinks = screen.getAllByRole("link", {
      name: strings.workbench.designIndex.open,
    });
    expect(openLinks[0]).toHaveAttribute("href", "/design?workspace=ws_harbor");
  });

  it("shows curated templates and their fork action", async () => {
    renderWithProviders(<DesignIndexPage />);
    await screen.findByText(WORKSPACE_NAME);

    expect(screen.getAllByTestId("design-template-card")).toHaveLength(1);
    expect(screen.getByText(CURATED_NAME)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: strings.workbench.designIndex.useTemplate }),
    ).toBeInTheDocument();
  });

  it("shows a loading state while the workspace list is pending", () => {
    vi.mocked(workspaceHook.useWorkspaces).mockReturnValue(
      queryResult<WorkspaceSummary[]>(undefined, { isLoading: true }),
    );
    renderWithProviders(<DesignIndexPage />);
    expect(screen.getByTestId("design-index-loading")).toBeInTheDocument();
  });

  it("shows an honest empty state when no authored workspace exists", async () => {
    vi.mocked(workspaceHook.useWorkspaces).mockReturnValue(queryResult([]));
    renderWithProviders(<DesignIndexPage />);
    expect(
      await screen.findByText(strings.workbench.designIndex.emptyTitle),
    ).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.designIndex.noTemplates)).toBeInTheDocument();
  });

  it("reports an unavailable workspace service instead of an empty list", async () => {
    vi.mocked(workspaceHook.useWorkspaces).mockReturnValue(
      queryResult<WorkspaceSummary[]>(undefined, { isError: true }),
    );
    renderWithProviders(<DesignIndexPage />);
    expect(
      await screen.findByText(strings.workbench.designIndex.errorTitle),
    ).toBeInTheDocument();
  });
});
