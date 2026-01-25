import { createContext } from "react";

export type DocsTab = "getting-started" | "node-reference" | "schema-reference" | "shortcuts";

export interface AssetUploadModalConfig {
  folder: string;
  projectId: string;
}

export interface ModalState {
  // Modal visibility
  showAIModal: boolean;
  showProjectModal: boolean;
  showWorkflowCreationModal: boolean;
  showAssetUploadModal: boolean;
  assetUploadConfig: AssetUploadModalConfig | null;
  showDocs: boolean;
  docsInitialTab: DocsTab;
}

export interface ModalActions {
  // AI Modal
  openAIModal: () => void;
  closeAIModal: () => void;

  // Project Modal
  openProjectModal: () => void;
  closeProjectModal: () => void;

  // Workflow Creation Modal
  openWorkflowCreationModal: () => void;
  closeWorkflowCreationModal: () => void;

  // Asset Upload Modal
  openAssetUploadModal: (config: AssetUploadModalConfig) => void;
  closeAssetUploadModal: () => void;

  // Docs Modal
  openDocs: (tab?: DocsTab) => void;
  closeDocs: () => void;

  // Bulk operations
  closeAllModals: () => void;
}

export type ModalContextValue = ModalState & ModalActions;

export const ModalContext = createContext<ModalContextValue | null>(null);
