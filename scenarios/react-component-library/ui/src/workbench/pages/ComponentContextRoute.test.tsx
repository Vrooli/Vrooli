import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { Route, Routes, useLocation, useSearchParams } from "react-router-dom";

import { getWorkspace, WorkspaceProvenance, type Workspace } from "../../api/workspaces";
import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import { ComponentContextRoute } from "./ComponentContextRoute";
import {
  createComponentWorkspace,
  createComponentWorkspaceFromTemplate,
} from "../workspaces/componentContext";
import { loadComponentBundle } from "../data";
import type { Template } from "../templates/spec";
import { readLegacyTemplateForMigration } from "../templates/readLegacyTemplate";

vi.mock("../data", () => ({
  loadComponentBundle: vi.fn().mockResolvedValue({
    stories: [
      { id: "story-1", version: "1.2.3" },
      { id: "story-new", version: "2.0.0" },
    ],
  }),
}));

vi.mock("../../api/workspaces", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/workspaces")>();
  return { ...actual, getWorkspace: vi.fn() };
});

vi.mock("../workspaces/componentContext", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../workspaces/componentContext")>();
  return {
    ...actual,
    createComponentWorkspace: vi.fn().mockResolvedValue({ id: "ws_component_context" }),
    createComponentWorkspaceFromTemplate: vi.fn().mockResolvedValue({ id: "ws_imported_context" }),
  };
});

vi.mock("../templates/readLegacyTemplate", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../templates/readLegacyTemplate")>();
  return { ...actual, readLegacyTemplateForMigration: vi.fn().mockReturnValue(undefined) };
});

const mockedGetWorkspace = vi.mocked(getWorkspace);
const mockedCreateComponentWorkspace = vi.mocked(createComponentWorkspace);
const mockedCreateComponentWorkspaceFromTemplate = vi.mocked(createComponentWorkspaceFromTemplate);
const mockedLoadComponentBundle = vi.mocked(loadComponentBundle);
const mockedReadLegacyTemplate = vi.mocked(readLegacyTemplateForMigration);

const legacyTemplate: Template = {
  id: "my-custom-layout",
  name: "My saved layout",
  tagline: "A browser draft",
  category: "portfolio",
  slotHint: "Hero",
  idea: "A saved browser-only composition",
  palette: {
    bg: "#fff",
    surface: "#fff",
    surfaceAlt: "#eee",
    ink: "#111",
    muted: "#666",
    line: "#ddd",
    accent: "#f80",
    accentInk: "#111",
    accentSoft: "#fee",
    secondary: "#08f",
  },
  display: "sans",
  radius: 12,
  spec: { id: "legacy-page", type: "page", props: {} },
};

function Location() {
  const location = useLocation();
  return <span data-testid="location">{location.pathname + location.search}</span>;
}

function SwitchContext() {
  const [, setParams] = useSearchParams();
  return (
    <button
      type="button"
      onClick={() => setParams({ template: "marlow", component: "cmp-new", story: "story-new" })}
    >
      Switch component context
    </button>
  );
}

function renderRoute(search: string) {
  return renderWithProviders(
    <Routes>
      <Route
        path="/design"
        element={
          <>
            <ComponentContextRoute />
            <SwitchContext />
            <Location />
          </>
        }
      />
    </Routes>,
    { initialEntries: [`/design${search}`] },
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  mockedLoadComponentBundle.mockResolvedValue({
    stories: [
      { id: "story-1", version: "1.2.3" },
      { id: "story-new", version: "2.0.0" },
    ],
  } as Awaited<ReturnType<typeof loadComponentBundle>>);
  mockedReadLegacyTemplate.mockReturnValue(undefined);
  mockedCreateComponentWorkspaceFromTemplate.mockResolvedValue({
    id: "ws_imported_context",
  } as Workspace);
});

describe("ComponentContextRoute", () => {
  it("persists the exact selected component version from a deep link", async () => {
    mockedGetWorkspace.mockResolvedValue({
      id: "ws_curated_aperture",
      provenance: WorkspaceProvenance.CURATED,
    } as Workspace);
    mockedCreateComponentWorkspace.mockClear();
    renderRoute("?template=aperture&component=cmp-1&story=story-1&version=1.2.3");

    await waitFor(() =>
      expect(mockedCreateComponentWorkspace).toHaveBeenCalledWith({
        templateWorkspace: expect.objectContaining({ id: "ws_curated_aperture" }),
        componentId: "cmp-1",
        storyId: "story-1",
        componentVersion: "1.2.3",
      }),
    );
  });

  it("does not create a workspace for a stale story reference", async () => {
    mockedGetWorkspace.mockResolvedValue({
      id: "ws_curated_aperture",
      provenance: WorkspaceProvenance.CURATED,
    } as Workspace);
    mockedLoadComponentBundle.mockResolvedValue({ stories: [] } as unknown as Awaited<
      ReturnType<typeof loadComponentBundle>
    >);
    mockedCreateComponentWorkspace.mockClear();

    renderRoute("?template=aperture&component=cmp-1&story=removed-story");

    expect(await screen.findByRole("alert")).toHaveTextContent(
      /Story “removed-story” is unavailable/,
    );
    expect(screen.getByRole("alert")).toHaveTextContent(/No workspace was created/);
    expect(mockedCreateComponentWorkspace).not.toHaveBeenCalled();
  });

  it("shows a migration error instead of reopening an unmatched legacy editor", async () => {
    mockedGetWorkspace.mockResolvedValue(null);
    renderRoute("?template=my-custom-layout&component=cmp-1&story=story-1");

    expect(await screen.findByRole("alert")).toHaveTextContent(/my-custom-layout/);
    expect(screen.getByRole("alert")).toHaveTextContent(/curated workspace or browser cache entry/);
    expect(screen.getByRole("alert")).not.toHaveTextContent(/saved legacy template/);
    expect(screen.getByRole("alert")).toHaveTextContent(/No workspace was created/);
    expect(screen.queryByTestId("legacy-template-preview")).not.toBeInTheDocument();
    expect(screen.getByTestId("location")).toHaveTextContent("template=my-custom-layout");
  });

  it("asks before copying a browser-only template into a workspace", async () => {
    mockedGetWorkspace.mockResolvedValue(null);
    mockedReadLegacyTemplate.mockReturnValue(legacyTemplate);
    mockedCreateComponentWorkspaceFromTemplate.mockClear();

    renderRoute("?template=my-custom-layout&component=cmp-1&story=story-1");

    expect(
      await screen.findByRole("heading", { name: strings.workbench.studio.legacyImportTitle }),
    ).toBeVisible();
    expect(screen.getByText(strings.workbench.studio.legacyImportBody)).toBeVisible();
    expect(mockedCreateComponentWorkspaceFromTemplate).not.toHaveBeenCalled();

    fireEvent.click(
      screen.getByRole("button", { name: strings.workbench.studio.legacyImportConfirm }),
    );
    await waitFor(() =>
      expect(mockedCreateComponentWorkspaceFromTemplate).toHaveBeenCalledWith({
        template: legacyTemplate,
        componentId: "cmp-1",
        storyId: "story-1",
        componentVersion: "1.2.3",
      }),
    );
  });

  it("does not fork a stale template after the component-context URL changes", async () => {
    let resolveOldLookup!: (workspace: Workspace | null) => void;
    const oldLookup = new Promise<Workspace | null>((resolve) => {
      resolveOldLookup = resolve;
    });
    const marlow = {
      id: "ws_curated_marlow",
      provenance: WorkspaceProvenance.CURATED,
    } as Workspace;
    mockedGetWorkspace.mockImplementation((id) =>
      id === "ws_curated_aperture" ? oldLookup : Promise.resolve(marlow),
    );
    mockedCreateComponentWorkspace.mockClear();
    renderRoute("?template=aperture&component=cmp-old&story=story-old");

    fireEvent.click(screen.getByRole("button", { name: "Switch component context" }));
    await waitFor(() => expect(mockedGetWorkspace).toHaveBeenCalledWith("ws_curated_marlow"));
    resolveOldLookup({
      id: "ws_curated_aperture",
      provenance: WorkspaceProvenance.CURATED,
    } as Workspace);

    await waitFor(() => expect(mockedCreateComponentWorkspace).toHaveBeenCalledTimes(1));
    expect(mockedCreateComponentWorkspace).toHaveBeenCalledWith({
      templateWorkspace: marlow,
      componentId: "cmp-new",
      storyId: "story-new",
      componentVersion: "2.0.0",
    });
  });
});
