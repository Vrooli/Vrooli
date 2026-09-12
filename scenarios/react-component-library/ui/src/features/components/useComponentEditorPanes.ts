import { useEffect, useState } from "react";

export const EDITOR_PANES = ["details", "files", "preview"] as const;
export type WorkspacePane = (typeof EDITOR_PANES)[number];
export type FilesView = "tree" | "source" | "diff";

const PANEL_LAYOUT_STORAGE_KEY = "rcl.component-editor.split-view.v1";
const DEFAULT_DESKTOP_PANEL_LAYOUT = { primary: 50, secondary: 50 };

function loadSplitLayout(): Record<string, number> {
  try {
    const raw = window.localStorage.getItem(PANEL_LAYOUT_STORAGE_KEY);
    return raw
      ? { ...DEFAULT_DESKTOP_PANEL_LAYOUT, ...JSON.parse(raw) }
      : DEFAULT_DESKTOP_PANEL_LAYOUT;
  } catch {
    return DEFAULT_DESKTOP_PANEL_LAYOUT;
  }
}

export function useComponentEditorPanes({
  activePane,
  onActivePaneChange,
  renderable,
  desktopLayout,
  comparison,
  setSelectedFile,
}: {
  activePane?: WorkspacePane;
  onActivePaneChange?: (pane: WorkspacePane) => void;
  renderable: boolean;
  desktopLayout: boolean;
  comparison?: unknown;
  setSelectedFile: (path: string) => void;
}) {
  const [uncontrolledPane, setUncontrolledPane] = useState<WorkspacePane>("files");
  const currentPane = activePane ?? uncontrolledPane;
  const [splitView, setSplitView] = useState(false);
  const [secondaryPane, setSecondaryPane] = useState<WorkspacePane>(
    renderable ? "preview" : "files",
  );
  const [splitLayout, setSplitLayout] = useState<Record<string, number>>(loadSplitLayout);
  const [filesView, setFilesView] = useState<FilesView>("source");
  const [wordWrap, setWordWrap] = useState<"on" | "off">("on");
  const [fontSize, setFontSize] = useState(13);

  const availablePanes = EDITOR_PANES.filter((pane) => pane !== "preview" || renderable);
  const selectPane = (pane: WorkspacePane) => {
    if (pane === "preview" && !renderable) return;
    onActivePaneChange ? onActivePaneChange(pane) : setUncontrolledPane(pane);
  };
  const visiblePanes = splitView
    ? [currentPane, secondaryPane].filter((pane, index, panes) => panes.indexOf(pane) === index)
    : [currentPane];
  const toggleSplitView = () => {
    if (!desktopLayout) return;
    if (!splitView && secondaryPane === currentPane) {
      setSecondaryPane(availablePanes.find((pane) => pane !== currentPane) ?? currentPane);
    }
    setSplitView((current) => !current);
  };
  const selectSplitPane = (index: number, pane: WorkspacePane) => {
    if (pane === "preview" && !renderable) return;
    if (index === 0) {
      selectPane(pane);
      if (pane === secondaryPane) setSecondaryPane(currentPane);
      return;
    }
    setSecondaryPane(
      pane === currentPane
        ? (availablePanes.find((candidate) => candidate !== currentPane) ?? pane)
        : pane,
    );
  };
  const saveDesktopPanelLayout = (layout: Record<string, number>) => {
    if (desktopLayout) setSplitLayout((current) => ({ ...current, ...layout }));
  };
  const selectFile = (path: string) => {
    setSelectedFile(path);
    setFilesView("source");
  };

  useEffect(() => {
    try {
      window.localStorage.setItem(PANEL_LAYOUT_STORAGE_KEY, JSON.stringify(splitLayout));
    } catch {
      // Split sizing is a convenience; editing never depends on storage access.
    }
  }, [splitLayout]);
  useEffect(() => {
    if (!desktopLayout) setSplitView(false);
  }, [desktopLayout]);
  useEffect(() => {
    if (comparison) {
      setFilesView("diff");
      selectPane("files");
    }
  }, [comparison]);

  return {
    currentPane,
    splitView,
    secondaryPane,
    splitLayout,
    filesView,
    wordWrap,
    fontSize,
    setFilesView,
    setWordWrap,
    setFontSize,
    availablePanes,
    visiblePanes,
    toggleSplitView,
    selectSplitPane,
    saveDesktopPanelLayout,
    selectFile,
  };
}
