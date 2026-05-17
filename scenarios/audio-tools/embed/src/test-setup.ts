/**
 * Vitest setup for the embed package.
 *
 * Registers @testing-library/jest-dom matchers (`.toBeInTheDocument()`,
 * `.toHaveAccessibleName()`, etc.) and clears localStorage between tests.
 *
 * Mirrors the shape of `ui/src/test-setup.ts` minus i18next — the embed
 * surface is opinion-free about translation and its components accept
 * caller-supplied labels directly. Tests assert on those caller labels.
 */
import "@testing-library/jest-dom/vitest";
import { afterEach, beforeEach } from "vitest";
import { cleanup } from "@testing-library/react";

beforeEach(() => {
  if (typeof window !== "undefined") {
    window.localStorage.clear();
  }
});

// React Testing Library doesn't auto-cleanup when `globals: false` and
// vitest's globals aren't enabled — call it explicitly so every test sees
// a fresh DOM.
afterEach(() => {
  cleanup();
});
