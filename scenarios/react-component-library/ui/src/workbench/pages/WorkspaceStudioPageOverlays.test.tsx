import { fireEvent, screen } from "@testing-library/react";
import { vi } from "vitest";

import { renderWithProviders } from "../../test-utils/renderWithProviders";
import { selectors } from "../../consts/selectors";
import type { DesignDocument, DesignFrame } from "../design-types";
import { WorkspaceStudioPageOverlays } from "./WorkspaceStudioPageOverlays";

vi.mock("../useFocusTrap", () => ({
  useFocusTrap: () => ({ current: null }),
}));

vi.mock("../studio/HtmlImport", () => ({
  HtmlImport: ({
    onClose,
    onImport,
  }: {
    onClose: () => void;
    onImport: (template: unknown, name: string) => void;
  }) => (
    <section aria-label="HTML import dialog">
      <button onClick={() => onImport({ id: "imported-template" }, "Imported page")}>
        Import page
      </button>
      <button onClick={onClose}>Cancel import</button>
    </section>
  ),
}));

vi.mock("../templates/TemplateRenderer", () => ({
  TemplateRenderer: ({ template }: { template: { id: string } }) => (
    <div data-testid="presented-template">{template.id}</div>
  ),
}));

const frame: DesignFrame = {
  id: "phone-frame",
  name: "Phone",
  device: "phone",
  spec: { id: "page", type: "page", props: {}, children: [] },
};
const design: DesignDocument = {
  id: "design-1",
  name: "Design",
  updatedAt: "2025-01-01T00:00:00Z",
  revision: 1,
  frames: [frame],
  theme: {
    bg: "#fff",
    surface: "#eee",
    surfaceAlt: "#ddd",
    ink: "#111",
    muted: "#666",
    line: "#ccc",
    accent: "#06c",
    accentInk: "#fff",
    accentSoft: "#def",
    secondary: "#456",
  },
};

function renderOverlays(
  overrides: Partial<Parameters<typeof WorkspaceStudioPageOverlays>[0]> = {},
) {
  const props = {
    htmlImportOpen: false,
    capturePatchOpen: false,
    onCloseHtmlImport: vi.fn(),
    onCloseCapturePatch: vi.fn(),
    onImportHtml: vi.fn(),
    onApplyCapturePatch: vi.fn(),
    readOnly: false,
    notice: null,
    design,
    frame,
    presenting: false,
    onClosePresentation: vi.fn(),
    presentScale: 1,
    ...overrides,
  };
  return { ...renderWithProviders(<WorkspaceStudioPageOverlays {...props} />), props };
}

describe("WorkspaceStudioPageOverlays", () => {
  it("renders no overlay content while import, notice and presentation are closed", () => {
    renderOverlays();

    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
    expect(screen.queryByRole("region")).not.toBeInTheDocument();
  });

  it("opens the patch handoff from the workspace overlay stack", () => {
    renderOverlays({ capturePatchOpen: true });

    expect(screen.getByTestId(selectors.workspaceStudio.capturePatchDialog)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.workspaceStudio.capturePatchFile)).toBeInTheDocument();
  });

  it("forwards the imported template and closes the import surface", () => {
    const { props } = renderOverlays({ htmlImportOpen: true });

    fireEvent.click(screen.getByRole("button", { name: "Import page" }));

    expect(props.onImportHtml).toHaveBeenCalledWith("Imported page", { id: "imported-template" });
    expect(props.onCloseHtmlImport).toHaveBeenCalledOnce();
  });

  it("renders notices as a status message and presents a closable frame dialog", () => {
    const { props } = renderOverlays({
      notice: "Saved changes",
      presenting: true,
      presentScale: 0.75,
    });

    expect(screen.getByRole("status")).toHaveTextContent("Saved changes");
    expect(screen.getByRole("dialog", { name: "workbench.studio.presentLabel" })).toHaveAttribute(
      "aria-modal",
      "true",
    );
    expect(screen.getByTestId("presented-template")).toHaveTextContent("design-1:phone-frame");
    const presentFrame = document.querySelector<HTMLElement>(".studio-present-frame");
    expect(presentFrame).toHaveStyle({ width: "390px" });
    expect(presentFrame?.style.zoom).toBe("0.75");

    fireEvent.click(screen.getByRole("button", { name: "workbench.studio.closePresent" }));
    expect(props.onClosePresentation).toHaveBeenCalledOnce();
  });
});
