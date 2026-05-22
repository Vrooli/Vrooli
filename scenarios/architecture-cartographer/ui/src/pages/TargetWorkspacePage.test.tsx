import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
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

  it("renders the placeholder note about upcoming sub-sections", () => {
    renderAt("/targets/demo");
    // cimode renders key paths; the placeholder note exists to advertise
    // that sub-nav sections (Graph, Manifest, …) arrive in later phases.
    expect(
      screen.getByText(strings.pages.targetWorkspace.subnavComingSoon),
    ).toBeInTheDocument();
  });
});
