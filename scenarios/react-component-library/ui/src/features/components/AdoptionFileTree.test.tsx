import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import { selectors } from "../../consts/selectors";

vi.mock("../../api/adoptions", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/adoptions")>();
  return {
    ...actual,
    adoptionsClient: {
      ...actual.adoptionsClient,
      resolveAdoptionPath: vi.fn(),
    },
  };
});

import { AdoptionFileTree } from "./AdoptionFileTree";
import { ADOPTION_TEMPLATES } from "./adoptionTemplates";
import { adoptionsClient } from "../../api/adoptions";

// The real web-console DrawerShell adoption: entry .tsx under components/, the
// two hooks under hooks/. This is the placement the tree must reproduce.
const drawerShellResolved = {
  path: "ui/src/components/DrawerShell.tsx",
  source: 2,
  slot: "shared-component",
  warnings: [],
  template: "react-vite",
  manifestResolved: true,
  files: [
    {
      libraryPath: "DrawerShell.tsx",
      targetPath: "ui/src/components/DrawerShell.tsx",
      slot: "shared-component",
      source: 2,
      slotSource: "entry",
      isEntry: true,
      warnings: [],
    },
    {
      libraryPath: "useFocusTrap.ts",
      targetPath: "ui/src/hooks/useFocusTrap.ts",
      slot: "hook",
      source: 2,
      slotSource: "heuristic",
      isEntry: false,
      warnings: [],
    },
    {
      libraryPath: "useEscapeKey.ts",
      targetPath: "ui/src/hooks/useEscapeKey.ts",
      slot: "hook",
      source: 2,
      slotSource: "heuristic",
      isEntry: false,
      warnings: [],
    },
  ],
};

const flatFiles = [
  { path: "DrawerShell.tsx", isEntry: true },
  { path: "useFocusTrap.ts", isEntry: false },
  { path: "useEscapeKey.ts", isEntry: false },
];

function renderTree(overrides: Partial<Parameters<typeof AdoptionFileTree>[0]> = {}) {
  return renderWithProviders(
    <AdoptionFileTree
      componentId="cid-1"
      version={undefined}
      files={flatFiles}
      selectedFile=""
      onSelectFile={vi.fn()}
      template="react-vite"
      templates={ADOPTION_TEMPLATES}
      onSelectTemplate={vi.fn()}
      {...overrides}
    />,
  );
}

describe("AdoptionFileTree", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the DrawerShell files at their template-resolved target paths", async () => {
    vi.mocked(adoptionsClient.resolveAdoptionPath).mockResolvedValue(drawerShellResolved as never);
    renderTree();

    const tree = await screen.findByTestId(selectors.components.editor.fileTree);
    expect(tree).toBeInTheDocument();

    const nodes = await screen.findAllByTestId(selectors.components.editor.fileTreeNode);
    const nodeFor = (basename: string) =>
      nodes.find((node) => node.textContent.includes(basename))!;
    // Entry lands in components/, hooks land in hooks/ — matching the real
    // web-console adoption record.
    expect(nodeFor("DrawerShell.tsx")).toHaveAttribute("data-slot", "shared-component");
    expect(nodeFor("DrawerShell.tsx")).toHaveAttribute("data-entry", "true");
    expect(nodeFor("useFocusTrap.ts")).toHaveAttribute("data-slot", "hook");
    expect(nodeFor("useEscapeKey.ts")).toHaveAttribute("data-slot", "hook");

    // Directory labels appear for the two slot dirs (regex matchers are the
    // copy-independent form the lint rule allows for structural labels).
    expect(screen.getByText(/^hooks$/)).toBeInTheDocument();
    expect(screen.getByText(/^components$/)).toBeInTheDocument();
  });

  it("opens the entry file as the current buffer (empty selection) and companions by basename", async () => {
    vi.mocked(adoptionsClient.resolveAdoptionPath).mockResolvedValue(drawerShellResolved as never);
    const onSelectFile = vi.fn();
    renderTree({ onSelectFile });

    const nodes = await screen.findAllByTestId(selectors.components.editor.fileTreeNode);
    const entry = nodes.find((n) => n.getAttribute("data-entry") === "true")!;
    const hook = nodes.find((n) => n.textContent.includes("useFocusTrap.ts"))!;

    await userEvent.click(entry);
    expect(onSelectFile).toHaveBeenLastCalledWith("");

    await userEvent.click(hook);
    expect(onSelectFile).toHaveBeenLastCalledWith("useFocusTrap.ts");
  });

  it("falls back to the flat file-tab row when no manifest resolves", async () => {
    vi.mocked(adoptionsClient.resolveAdoptionPath).mockResolvedValue({
      ...drawerShellResolved,
      manifestResolved: false,
      files: [],
    } as never);
    renderTree();

    const tabs = await screen.findByTestId(selectors.components.editor.fileTabs);
    expect(tabs).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.components.editor.fileTree)).not.toBeInTheDocument();
    // Every version file is reachable as a tab.
    expect(screen.getByRole("tab", { name: "DrawerShell.tsx" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "useFocusTrap.ts" })).toBeInTheDocument();
  });

  it("shows the template as a static label when only one template is available", async () => {
    vi.mocked(adoptionsClient.resolveAdoptionPath).mockResolvedValue(drawerShellResolved as never);
    renderTree();

    const seam = await screen.findByTestId(selectors.components.editor.templateSelect);
    // Single template -> not a <select>.
    expect(seam.tagName).not.toBe("SELECT");
    expect(seam).toHaveTextContent("react-vite");
  });

  it("offers a template <select> and reports the choice when multiple templates exist", async () => {
    vi.mocked(adoptionsClient.resolveAdoptionPath).mockResolvedValue(drawerShellResolved as never);
    const onSelectTemplate = vi.fn();
    renderTree({
      templates: [
        { id: "react-vite", label: "react-vite" },
        { id: "next-app", label: "next-app" },
      ],
      onSelectTemplate,
    });

    // Wait for the placement query's rerender before retaining the select;
    // the initial fallback and resolved tree each render the seam.
    await screen.findByTestId(selectors.components.editor.fileTree);
    const seam = await screen.findByTestId(selectors.components.editor.templateSelect);
    expect(seam.tagName).toBe("SELECT");

    const user = userEvent.setup();
    await user.selectOptions(seam, "next-app");
    expect(onSelectTemplate).toHaveBeenCalledWith("next-app");
  });

  it("switches files from the flat fallback tabs, mapping the entry to the current buffer", async () => {
    vi.mocked(adoptionsClient.resolveAdoptionPath).mockResolvedValue({
      ...drawerShellResolved,
      manifestResolved: false,
      files: [],
    } as never);
    const onSelectFile = vi.fn();
    renderTree({ onSelectFile });

    await screen.findByTestId(selectors.components.editor.fileTabs);
    // The entry tab maps to the empty current-buffer selection.
    await userEvent.click(screen.getByRole("tab", { name: "DrawerShell.tsx" }));
    expect(onSelectFile).toHaveBeenLastCalledWith("");
    // A companion tab maps to its own path.
    await userEvent.click(screen.getByRole("tab", { name: "useEscapeKey.ts" }));
    expect(onSelectFile).toHaveBeenLastCalledWith("useEscapeKey.ts");
  });
});
