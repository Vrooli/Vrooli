/**
 * App-level Keyboard Shortcuts
 *
 * Extracts keyboard shortcut setup from App.tsx for better separation of concerns.
 * Manages shortcut context detection and action registration.
 */
import { useMemo } from "react";
import {
  useKeyboardShortcutHandler,
  useRegisterShortcuts,
  useShortcutContext,
} from "@hooks/useKeyboardShortcuts";
import { type ShortcutContext } from "@stores/keyboardShortcutsStore";
import type { AppView } from "@/routing";
import type { Project } from "@stores/projectStore";

interface UseAppShortcutsParams {
  // Current app state for context detection
  currentView: AppView | null;
  showAIModal: boolean;
  showProjectModal: boolean;
  showDocs: boolean;
  showTour: boolean;

  // Actions
  openDocs: (tab?: "getting-started" | "node-reference" | "schema-reference" | "shortcuts") => void;
  closeDocs: () => void;
  closeAIModal: () => void;
  closeProjectModal: () => void;
  navigateToDashboard: () => void;
  navigateToSettings: () => void;
  openProject: (project: Project, options?: { replace?: boolean }) => void;
  openProjectModal: () => void;
  openAIModal: () => void;
  openTour: () => void;
  handleStartRecording: () => void;

  // Current context
  currentProject: Project | null;
}

/**
 * Hook that sets up all app-level keyboard shortcuts.
 * Call this once at the app root level.
 */
export function useAppShortcuts({
  currentView,
  showAIModal,
  showProjectModal,
  showDocs,
  showTour,
  openDocs,
  closeDocs,
  closeAIModal,
  closeProjectModal,
  navigateToDashboard,
  navigateToSettings,
  openProject,
  openProjectModal,
  openAIModal,
  openTour,
  handleStartRecording,
  currentProject,
}: UseAppShortcutsParams): void {
  // Set up the global keyboard shortcut handler (must be called once at root)
  useKeyboardShortcutHandler();

  // Map current view to shortcut context
  const shortcutContext = useMemo<ShortcutContext>(() => {
    // Check for open modals first
    if (showAIModal || showProjectModal || showDocs || showTour) {
      return "modal";
    }
    switch (currentView) {
      case "dashboard":
      case "all-workflows":
      case "all-executions":
        return "dashboard";
      case "project-detail":
        return "project-detail";
      case "project-workflow":
        return "workflow-builder";
      case "settings":
        return "settings";
      case "record-mode":
        return "modal"; // Record mode uses modal context for focused recording experience
      default:
        return "dashboard";
    }
  }, [currentView, showAIModal, showProjectModal, showDocs, showTour]);

  // Set the active shortcut context
  useShortcutContext(shortcutContext);

  // Register all shortcut actions
  const shortcutActions = useMemo(
    () => ({
      // Global shortcuts
      "show-shortcuts": () => {
        openDocs("shortcuts");
      },
      "close-modal": () => {
        if (showDocs) {
          closeDocs();
        } else if (showAIModal) {
          closeAIModal();
        } else if (showProjectModal) {
          closeProjectModal();
        } else if (currentView === "project-workflow" && currentProject) {
          openProject(currentProject);
        } else if (currentView === "project-detail" || currentView === "settings") {
          navigateToDashboard();
        }
      },
      "global-search": () => {
        // In workflow builder context, focus the node palette search
        // Otherwise, open the dashboard global search modal
        const nodePaletteSearch = document.querySelector<HTMLInputElement>(
          '[data-testid="node-palette-search-input"]'
        );
        if (nodePaletteSearch && shortcutContext === "workflow-builder") {
          nodePaletteSearch.focus();
        } else {
          // Dispatch event for Dashboard to open search modal
          window.dispatchEvent(new CustomEvent("open-global-search"));
        }
      },
      "open-settings": () => navigateToSettings(),

      // Dashboard shortcuts
      "new-project": () => openProjectModal(),
      "go-home": () => navigateToDashboard(),
      "open-tutorial": () => openTour(),

      // Project detail shortcuts
      "new-workflow": () => openAIModal(),
      "start-recording": () => {
        handleStartRecording();
      },
    }),
    [
      showDocs,
      showAIModal,
      showProjectModal,
      currentView,
      currentProject,
      shortcutContext,
      openDocs,
      closeDocs,
      closeAIModal,
      closeProjectModal,
      openProject,
      navigateToDashboard,
      navigateToSettings,
      openProjectModal,
      openAIModal,
      openTour,
      handleStartRecording,
    ]
  );

  useRegisterShortcuts(shortcutActions);
}
