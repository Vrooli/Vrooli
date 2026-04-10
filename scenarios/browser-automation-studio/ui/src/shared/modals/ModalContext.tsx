/**
 * Modal Coordination Context
 *
 * Centralizes modal state management that was previously scattered in App.tsx.
 * Provides a single source of truth for modal visibility and coordinates
 * modal closure on navigation changes.
 */
import { useState, useCallback, useEffect, type ReactNode } from "react";
import {
  ModalContext,
  type DocsTab,
  type AssetUploadModalConfig,
  type ModalContextValue,
} from "./ModalContextBase";

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
  const [showAssetUploadModal, setShowAssetUploadModal] = useState(false);
  const [assetUploadConfig, setAssetUploadConfig] = useState<AssetUploadModalConfig | null>(null);
  const [showDocs, setShowDocs] = useState(false);
  const [docsInitialTab, setDocsInitialTab] = useState<DocsTab>("getting-started");

  // Close modals when view changes (navigation occurred)
  useEffect(() => {
    setShowAIModal(false);
    setShowProjectModal(false);
    setShowWorkflowCreationModal(false);
    setShowAssetUploadModal(false);
    setAssetUploadConfig(null);
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

  // Asset Upload Modal actions
  const openAssetUploadModal = useCallback((config: AssetUploadModalConfig) => {
    setAssetUploadConfig(config);
    setShowAssetUploadModal(true);
  }, []);

  const closeAssetUploadModal = useCallback(() => {
    setShowAssetUploadModal(false);
    setAssetUploadConfig(null);
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
    setShowAssetUploadModal(false);
    setAssetUploadConfig(null);
    setShowDocs(false);
  }, []);

  const value: ModalContextValue = {
    // State
    showAIModal,
    showProjectModal,
    showWorkflowCreationModal,
    showAssetUploadModal,
    assetUploadConfig,
    showDocs,
    docsInitialTab,
    // Actions
    openAIModal,
    closeAIModal,
    openProjectModal,
    closeProjectModal,
    openWorkflowCreationModal,
    closeWorkflowCreationModal,
    openAssetUploadModal,
    closeAssetUploadModal,
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
