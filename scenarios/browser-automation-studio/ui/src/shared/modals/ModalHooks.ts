import { useContext } from "react";
import { ModalContext, type ModalContextValue } from "./ModalContextBase";

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
  const { showAIModal, showProjectModal, showWorkflowCreationModal, showAssetUploadModal, showDocs } = useModals();
  return showAIModal || showProjectModal || showWorkflowCreationModal || showAssetUploadModal || showDocs;
}
