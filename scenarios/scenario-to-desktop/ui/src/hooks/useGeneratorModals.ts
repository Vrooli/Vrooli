/**
 * Hook for managing Generator form modal states.
 * Consolidates modal open/close state into a single hook.
 */

import { useCallback, useState } from "react";

export interface ModalStates {
  scenario: boolean;
  template: boolean;
  framework: boolean;
  deployment: boolean;
}

export interface UseGeneratorModalsReturn {
  modals: ModalStates;
  openModal: (modal: keyof ModalStates) => void;
  closeModal: (modal: keyof ModalStates) => void;
  toggleModal: (modal: keyof ModalStates) => void;
}

/**
 * Hook for managing all generator form modal states.
 */
export function useGeneratorModals(): UseGeneratorModalsReturn {
  const [modals, setModals] = useState<ModalStates>({
    scenario: false,
    template: false,
    framework: false,
    deployment: false,
  });

  const openModal = useCallback((modal: keyof ModalStates) => {
    setModals((prev) => ({ ...prev, [modal]: true }));
  }, []);

  const closeModal = useCallback((modal: keyof ModalStates) => {
    setModals((prev) => ({ ...prev, [modal]: false }));
  }, []);

  const toggleModal = useCallback((modal: keyof ModalStates) => {
    setModals((prev) => ({ ...prev, [modal]: !prev[modal] }));
  }, []);

  return { modals, openModal, closeModal, toggleModal };
}
