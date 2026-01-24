/**
 * Tests for useGeneratorModals hook.
 * Tests modal state management for generator form.
 */

import { describe, it, expect } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useGeneratorModals, type ModalStates } from "./useGeneratorModals";

describe("useGeneratorModals", () => {
  describe("initial state", () => {
    it("all modals start closed", () => {
      const { result } = renderHook(() => useGeneratorModals());

      expect(result.current.modals).toEqual({
        scenario: false,
        template: false,
        framework: false,
        deployment: false,
      });
    });
  });

  describe("openModal", () => {
    it("opens scenario modal", () => {
      const { result } = renderHook(() => useGeneratorModals());

      act(() => {
        result.current.openModal("scenario");
      });

      expect(result.current.modals.scenario).toBe(true);
      expect(result.current.modals.template).toBe(false);
    });

    it("opens template modal", () => {
      const { result } = renderHook(() => useGeneratorModals());

      act(() => {
        result.current.openModal("template");
      });

      expect(result.current.modals.template).toBe(true);
    });

    it("opens framework modal", () => {
      const { result } = renderHook(() => useGeneratorModals());

      act(() => {
        result.current.openModal("framework");
      });

      expect(result.current.modals.framework).toBe(true);
    });

    it("opens deployment modal", () => {
      const { result } = renderHook(() => useGeneratorModals());

      act(() => {
        result.current.openModal("deployment");
      });

      expect(result.current.modals.deployment).toBe(true);
    });

    it("opening one modal does not affect others", () => {
      const { result } = renderHook(() => useGeneratorModals());

      act(() => {
        result.current.openModal("scenario");
        result.current.openModal("template");
      });

      expect(result.current.modals.scenario).toBe(true);
      expect(result.current.modals.template).toBe(true);
      expect(result.current.modals.framework).toBe(false);
      expect(result.current.modals.deployment).toBe(false);
    });
  });

  describe("closeModal", () => {
    it("closes an open modal", () => {
      const { result } = renderHook(() => useGeneratorModals());

      act(() => {
        result.current.openModal("scenario");
      });

      expect(result.current.modals.scenario).toBe(true);

      act(() => {
        result.current.closeModal("scenario");
      });

      expect(result.current.modals.scenario).toBe(false);
    });

    it("closing a closed modal keeps it closed", () => {
      const { result } = renderHook(() => useGeneratorModals());

      act(() => {
        result.current.closeModal("template");
      });

      expect(result.current.modals.template).toBe(false);
    });

    it("closing one modal does not affect others", () => {
      const { result } = renderHook(() => useGeneratorModals());

      act(() => {
        result.current.openModal("scenario");
        result.current.openModal("template");
      });

      act(() => {
        result.current.closeModal("scenario");
      });

      expect(result.current.modals.scenario).toBe(false);
      expect(result.current.modals.template).toBe(true);
    });
  });

  describe("toggleModal", () => {
    it("toggles closed modal to open", () => {
      const { result } = renderHook(() => useGeneratorModals());

      expect(result.current.modals.framework).toBe(false);

      act(() => {
        result.current.toggleModal("framework");
      });

      expect(result.current.modals.framework).toBe(true);
    });

    it("toggles open modal to closed", () => {
      const { result } = renderHook(() => useGeneratorModals());

      act(() => {
        result.current.openModal("deployment");
      });

      expect(result.current.modals.deployment).toBe(true);

      act(() => {
        result.current.toggleModal("deployment");
      });

      expect(result.current.modals.deployment).toBe(false);
    });

    it("double toggle returns to original state", () => {
      const { result } = renderHook(() => useGeneratorModals());

      const originalState = result.current.modals.scenario;

      act(() => {
        result.current.toggleModal("scenario");
        result.current.toggleModal("scenario");
      });

      expect(result.current.modals.scenario).toBe(originalState);
    });
  });

  describe("type safety", () => {
    it("accepts all valid modal keys", () => {
      const { result } = renderHook(() => useGeneratorModals());

      const modalKeys: (keyof ModalStates)[] = [
        "scenario",
        "template",
        "framework",
        "deployment",
      ];

      modalKeys.forEach((key) => {
        act(() => {
          result.current.openModal(key);
        });
        expect(result.current.modals[key]).toBe(true);
      });
    });
  });

  describe("callback stability", () => {
    it("openModal function is stable across renders", () => {
      const { result, rerender } = renderHook(() => useGeneratorModals());

      const firstOpenModal = result.current.openModal;

      rerender();

      expect(result.current.openModal).toBe(firstOpenModal);
    });

    it("closeModal function is stable across renders", () => {
      const { result, rerender } = renderHook(() => useGeneratorModals());

      const firstCloseModal = result.current.closeModal;

      rerender();

      expect(result.current.closeModal).toBe(firstCloseModal);
    });

    it("toggleModal function is stable across renders", () => {
      const { result, rerender } = renderHook(() => useGeneratorModals());

      const firstToggleModal = result.current.toggleModal;

      rerender();

      expect(result.current.toggleModal).toBe(firstToggleModal);
    });
  });
});
