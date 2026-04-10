/**
 * Modal coordination feature
 *
 * Provides centralized modal state management for the app.
 */
export { ModalProvider } from "./ModalContext";
export { useModals, useIsAnyModalOpen } from "./ModalHooks";
export type { DocsTab } from "./ModalContextBase";
