/**
 * Modal Coordination Context
 *
 * Centralizes modal state management that was previously scattered in App.tsx.
 * Provides a single source of truth for modal visibility and coordinates
 * modal closure on navigation changes.
 */
import { createContext, useContext, useState, useCallback, useEffect, type ReactNode } from "react";

export type DocsTab = "getting-started" | "node-reference" | "schema-reference" | "shortcuts";

interface ModalState {
  // Modal visibility
  showAIModal: boolean;
  showProjectModal: boolean;
  showWorkflowCreationModal: boolean;
  showDocs: boolean;
  docsInitialTab: DocsTab;
}

interface ModalActions {
  // AI Modal
  openAIModal: () => void;
  closeAIModal: () => void;

  // Project Modal
  openProjectModal: () => void;
  closeProjectModal: () => void;

  // Workflow Creation Modal
  openWorkflowCreationModal: () => void;
  closeWorkflowCreationModal: () => void;

  // Docs Modal
  openDocs: (tab?: DocsTab) => void;
  closeDocs: () => void;

  // Bulk operations
  closeAllModals: () => void;
}

type ModalContextValue = ModalState & ModalActions;

const ModalContext = createContext<ModalContextValue | null>(null);

interface ModalProviderProps {
  children: ReactNode;
  /** Current view - when this changes, modals auto-close */
  currentView: string | null;
}

/**
 * Provider that manages all modal state and provides actions to open/close modals.
 * Automatically closes modals when navigation occurs.
 */
export function ModalProvider({ children, currentView }: ModalProviderProps) {
  const [showAIModal, setShowAIModal] = useState(false);
  const [showProjectModal, setShowProjectModal] = useState(false);
  const [showWorkflowCreationModal, setShowWorkflowCreationModal] = useState(false);
  const [showDocs, setShowDocs] = useState(false);
  const [docsInitialTab, setDocsInitialTab] = useState<DocsTab>("getting-started");

  // Close modals when view changes (navigation occurred)
  useEffect(() => {
    setShowAIModal(false);
    setShowProjectModal(false);
    setShowWorkflowCreationModal(false);
    // Note: Docs modal intentionally stays open across navigation
    // as users may want to reference docs while navigating
  }, [currentView]);

  // AI Modal actions
  const openAIModal = useCallback(() => {
    setShowAIModal(true);
  }, []);

  const closeAIModal = useCallback(() => {
    setShowAIModal(false);
  }, []);

  // Project Modal actions
  const openProjectModal = useCallback(() => {
    setShowProjectModal(true);
  }, []);

  const closeProjectModal = useCallback(() => {
    setShowProjectModal(false);
  }, []);

  // Workflow Creation Modal actions
  const openWorkflowCreationModal = useCallback(() => {
    setShowWorkflowCreationModal(true);
  }, []);

  const closeWorkflowCreationModal = useCallback(() => {
    setShowWorkflowCreationModal(false);
  }, []);

  // Docs Modal actions
  const openDocs = useCallback((tab: DocsTab = "getting-started") => {
    setDocsInitialTab(tab);
    setShowDocs(true);
  }, []);

  const closeDocs = useCallback(() => {
    setShowDocs(false);
  }, []);

  // Bulk close
  const closeAllModals = useCallback(() => {
    setShowAIModal(false);
    setShowProjectModal(false);
    setShowWorkflowCreationModal(false);
    setShowDocs(false);
  }, []);

  const value: ModalContextValue = {
    // State
    showAIModal,
    showProjectModal,
    showWorkflowCreationModal,
    showDocs,
    docsInitialTab,
    // Actions
    openAIModal,
    closeAIModal,
    openProjectModal,
    closeProjectModal,
    openWorkflowCreationModal,
    closeWorkflowCreationModal,
    openDocs,
    closeDocs,
    closeAllModals,
  };

  return (
    <ModalContext.Provider value={value}>
      {children}
    </ModalContext.Provider>
  );
}

/**
 * Hook to access modal state and actions.
 * Must be used within a ModalProvider.
 */
export function useModals(): ModalContextValue {
  const context = useContext(ModalContext);
  if (!context) {
    throw new Error("useModals must be used within a ModalProvider");
  }
  return context;
}

/**
 * Hook to check if any modal is currently open.
 * Useful for keyboard shortcut context detection.
 */
export function useIsAnyModalOpen(): boolean {
  const { showAIModal, showProjectModal, showWorkflowCreationModal, showDocs } = useModals();
  return showAIModal || showProjectModal || showWorkflowCreationModal || showDocs;
}
