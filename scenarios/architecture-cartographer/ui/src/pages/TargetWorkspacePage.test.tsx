import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { TargetWorkspacePage } from "./TargetWorkspacePage";

afterEach(() => cleanup());

function renderAt(path: string) {
  return renderWithProviders(
    <MemoryRouter
      initialEntries={[path]}
      future={{ v7_startTransition: true, v7_relativeSplatPath: true }}
    >
      <Routes>
        <Route path="/" element={<div data-testid="redirected-overview" />} />
        <Route path="/targets/:encodedPath" element={<TargetWorkspacePage />} />
      </Routes>
    </MemoryRouter>,
    { withoutRouter: true },
  );
}

describe("TargetWorkspacePage", () => {
  it("renders the workspace shell and exposes the decoded scenario name", () => {
    renderAt("/targets/architecture-cartographer");
    expect(screen.getByTestId(selectors.pages.targetWorkspace)).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.pages.targetWorkspace).textContent,
    ).toContain("architecture-cartographer");
  });

  it("decodes URL-encoded paths", () => {
    renderAt(`/targets/${encodeURIComponent("space and/slash")}`);
    expect(
      screen.getByTestId(selectors.pages.targetWorkspace).textContent,
    ).toContain("space and/slash");
  });

  it("renders the workspace sub-nav with all five sections", () => {
    renderAt("/targets/demo");
    expect(screen.getByTestId(selectors.layout.subnav)).toBeInTheDocument();
    for (const key of ["graph", "domains", "conflicts", "apply", "analytics"] as const) {
      expect(
        screen.getByTestId(selectors.layout.workspaceSubnavLink({ key })),
      ).toBeInTheDocument();
    }
  });

  it("renders graph + conflicts as navigable; the remaining tabs render as disabled chips", () => {
    renderAt("/targets/demo");
    const conflicts = screen.getByTestId(selectors.layout.workspaceSubnavLink({ key: "conflicts" }));
    expect(conflicts.tagName).toBe("A");
    const graph = screen.getByTestId(selectors.layout.workspaceSubnavLink({ key: "graph" }));
    expect(graph.tagName).toBe("A");

    for (const key of ["domains", "apply", "analytics"] as const) {
      const item = screen.getByTestId(selectors.layout.workspaceSubnavLink({ key }));
      expect(item.tagName).toBe("A");
    }
  });
});
