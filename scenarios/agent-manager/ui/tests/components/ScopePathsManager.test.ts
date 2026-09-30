import assert from "node:assert/strict";
import { fireEvent, screen } from "@testing-library/react";
import { createElement } from "react";
import { test, vi } from "vitest";
import { ScopePathsManager } from "../../src/components/ScopePathsManager.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

test("ScopePathsManager adds, deduplicates, removes, and suggests explicit scope paths", () => {
  const onRoot = vi.fn(); const onPaths = vi.fn();
  renderWithProviders(createElement(ScopePathsManager, {
    projectRoot: "/repo", onProjectRootChange: onRoot, scopePaths: [], onScopePathsChange: onPaths,
    defaultProjectRoot: "/repo", defaultScopePaths: ["api", "cli"], requireScopePaths: true,
  }));
  fireEvent.change(screen.getByLabelText("Project Root"), { target: { value: "/next" } });
  assert.deepEqual(onRoot.mock.calls, [["/next"]]);
  fireEvent.click(screen.getByRole("button", { name: /api/ }));
  assert.deepEqual(onPaths.mock.calls.at(-1), [["api"]]);
  fireEvent.change(screen.getByLabelText(/Scope Paths/), { target: { value: " cli " } });
  fireEvent.keyDown(screen.getByLabelText(/Scope Paths/), { key: "Enter" });
  assert.deepEqual(onPaths.mock.calls.at(-1), [["cli"]]);

  const rendered = renderWithProviders(createElement(ScopePathsManager, {
    projectRoot: "/repo", onProjectRootChange: onRoot, scopePaths: ["api"], onScopePathsChange: onPaths,
  }));
  fireEvent.click(rendered.getByRole("button", { name: "Remove api" }));
  assert.deepEqual(onPaths.mock.calls.at(-1), [[]]);
});
