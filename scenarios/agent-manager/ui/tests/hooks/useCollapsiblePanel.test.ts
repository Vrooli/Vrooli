import assert from "node:assert/strict";
import { act, renderHook } from "@testing-library/react";
import { afterEach, test } from "vitest";
import { useCollapsiblePanel } from "../../src/hooks/useCollapsiblePanel.js";

afterEach(() => localStorage.clear());

test("useCollapsiblePanel restores state and persists explicit collapse, expand, and toggle actions", () => {
  localStorage.setItem("legacy.panel", "true");
  const { result } = renderHook(() => useCollapsiblePanel({ storageKey: "ignored", persistKey: "legacy.panel" }));
  assert.equal(result.current.isCollapsed, true);
  act(() => result.current.expand());
  assert.equal(result.current.isCollapsed, false);
  assert.equal(localStorage.getItem("legacy.panel"), "false");
  act(() => result.current.collapse());
  assert.equal(result.current.isCollapsed, true);
  act(() => result.current.toggle());
  assert.equal(result.current.isCollapsed, false);
  assert.equal(localStorage.getItem("legacy.panel"), "false");
});
