import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { WorkspaceStudioPageContent } from "./WorkspaceStudioPageContent";

const mocks = vi.hoisted(() => ({
  studio: null as Record<string, unknown> | null,
  view: null as Record<string, unknown> | null,
  left: null as Record<string, unknown> | null,
  canvas: null as Record<string, unknown> | null,
  inspector: null as Record<string, unknown> | null,
}));

vi.mock("../../hooks/useGlobalKeydown", () => ({ useGlobalKeydown: vi.fn() }));
vi.mock("../../i18n", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../i18n")>();
  return {
    ...actual,
    useTranslation: () => ({
      t: (_key: string, opts?: { defaultValue?: string; name?: string }) =>
        (opts?.defaultValue ?? _key).replace("{{name}}", opts?.name ?? ""),
    }),
  };
});
vi.mock("../useFocusTrap", () => ({ useFocusTrap: () => ({ current: null }) }));
vi.mock("./useWorkspaceStudioViewport", () => ({
  useWorkspaceStudioViewport: () => ({
    canvasRef: { current: null },
    narrow: false,
    fitScale: 0.5,
    scales: {},
    audits: {},
    frameScale: 0.5,
    presentScale: 0.5,
  }),
}));
vi.mock("../studio/useWorkspaceStudio", () => ({ useWorkspaceStudio: () => mocks.studio }));
vi.mock("./useWorkspaceStudioCapture", () => ({
  useWorkspaceStudioCapture: () => ({ capturingState: false, captureSelectedState: vi.fn() }),
}));
vi.mock("./WorkspaceStudioLoadState", () => ({
  WorkspaceStudioLoadState: ({ ready }: { ready: boolean }) => (
    <div data-testid="load-state">{String(ready)}</div>
  ),
}));
vi.mock("./WorkspaceStudioConflictBanner", () => ({ WorkspaceStudioConflictBanner: () => null }));
vi.mock("./WorkspaceStudioPageHeader", () => ({
  WorkspaceStudioPageHeader: (props: Record<string, unknown>) => {
    return (
      <>
        <button onClick={props.onPresent as () => void}>Present</button>
        <button onClick={props.onImportHtml as () => void}>Open HTML import</button>
      </>
    );
  },
}));
vi.mock("./WorkspaceStudioLeftPane", () => ({
  WorkspaceStudioLeftPane: (props: Record<string, unknown>) => {
    mocks.left = props;
    return <div />;
  },
}));
vi.mock("./WorkspaceStudioCanvasArea", () => ({
  WorkspaceStudioCanvasArea: (props: Record<string, unknown>) => {
    mocks.canvas = props;
    return <button onClick={props.onToggleGuides as () => void}>Toggle guides</button>;
  },
}));
vi.mock("./WorkspaceStudioInspectorPane", () => ({
  WorkspaceStudioInspectorPane: (props: Record<string, unknown>) => {
    mocks.inspector = props;
    return <button onClick={props.onCloseSheet as () => void}>Close right panel</button>;
  },
}));
vi.mock("./WorkspaceStudioPageControls", () => ({
  WorkspaceStudioPageControls: (props: Record<string, unknown>) => {
    return (
      <>
        <button onClick={() => (props.onOpenLeft as (tab: string) => void)("layers")}>
          Open mobile panel
        </button>
        <button onClick={() => (props.onOpenRight as (tab: string) => void)("checks")}>
          Open checks panel
        </button>
        <button onClick={props.onRetrySave as () => void}>Retry save</button>
      </>
    );
  },
}));
vi.mock("./WorkspaceStudioPageOverlays", () => ({
  WorkspaceStudioPageOverlays: (props: Record<string, unknown>) => {
    mocks.view = props;
    return (
      <>
        {typeof props.notice === "string" ? <div role="status">{props.notice}</div> : null}
        {props.presenting ? (
          <button onClick={props.onClosePresentation as () => void}>Close presentation</button>
        ) : null}
        {props.htmlImportOpen ? (
          <>
            <button onClick={props.onCloseHtmlImport as () => void}>Close HTML import</button>
            <button
              onClick={() => {
                (props.onImportHtml as (name: string, template: unknown) => void)("Imported page", {
                  spec: { id: "imported", type: "page", props: {}, children: [] },
                });
                (props.onCloseHtmlImport as () => void)();
              }}
            >
              Import template
            </button>
          </>
        ) : null}
      </>
    );
  },
}));

const frame = {
  id: "frame",
  name: "Desktop",
  device: "desktop",
  spec: { id: "root", type: "page", props: {}, children: [] },
};
const page = { id: "page", name: "Home", route: "/", pageTypeId: "", sortOrder: 0, displays: [] };
const display = { id: "frame", device: "desktop", width: 1280, height: 720, states: [] };
const workspace = { id: "workspace", name: "Workspace", tags: [], pages: [page] };
const design = {
  id: "design",
  name: "Home",
  updatedAt: "",
  revision: 1,
  frames: [frame],
  theme: {},
};

beforeEach(() => {
  mocks.studio = {
    ready: true,
    loadError: null,
    workspace,
    page,
    display,
    doc: design,
    frames: [frame],
    state: null,
    dirty: false,
    saving: false,
    saveFailed: false,
    saved: undefined,
    conflict: null,
    selectedId: null,
    readOnly: false,
    history: { index: 0, entries: [] },
    selectAddress: vi.fn(),
    selectDisplay: vi.fn(),
    undo: vi.fn(),
    redo: vi.fn(),
    save: vi.fn(),
    addPage: vi.fn(),
  };
});

function renderPage() {
  return renderWithProviders(<WorkspaceStudioPageContent workspaceId="workspace" />);
}

describe("WorkspaceStudioPageContent composition callbacks", () => {
  it("opens and closes presentation and toggles canvas guides", () => {
    renderPage();
    fireEvent.click(screen.getByRole("button", { name: "Present" }));
    expect(mocks.view?.presenting).toBe(true);
    fireEvent.click(screen.getByRole("button", { name: "Close presentation" }));
    expect(mocks.view?.presenting).toBe(false);
    fireEvent.click(screen.getByRole("button", { name: "Toggle guides" }));
    expect(mocks.canvas?.guides).toBe(true);
    fireEvent.click(screen.getByRole("button", { name: "Retry save" }));
    expect(mocks.studio?.save as ReturnType<typeof vi.fn>).toHaveBeenCalledOnce();
  });

  it("opens a mobile sheet and closes it from its backdrop", () => {
    renderPage();
    fireEvent.click(screen.getByRole("button", { name: "Open mobile panel" }));
    expect(mocks.left?.leftTab).toBe("layers");
    expect(screen.getByRole("button", { name: "Close panel" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Close panel" }));
    expect(screen.queryByRole("button", { name: "Close panel" })).not.toBeInTheDocument();
  });

  it("opens and closes the right sheet and imports an HTML template as a new page", () => {
    renderPage();
    fireEvent.click(screen.getByRole("button", { name: "Open checks panel" }));
    expect(mocks.inspector?.rightTab).toBe("checks");
    fireEvent.click(screen.getByRole("button", { name: "Close right panel" }));
    expect(screen.queryByRole("button", { name: "Close panel" })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Open HTML import" }));
    expect(mocks.view?.htmlImportOpen).toBe(true);
    fireEvent.click(screen.getByRole("button", { name: "Import template" }));
    expect(mocks.studio?.addPage).toHaveBeenCalledWith(
      "Imported page",
      expect.objectContaining({ id: "imported" }),
    );
    expect(mocks.view?.htmlImportOpen).toBe(false);
    expect(screen.getByRole("status")).toHaveTextContent("Imported Imported page as a new page.");
  });
});
