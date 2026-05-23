import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { WorkspaceSubNav } from "./WorkspaceSubNav";

afterEach(() => cleanup());

describe("WorkspaceSubNav", () => {
  it("renders all five sections, with conflicts as the only navigable link in Phase 4", () => {
    renderWithProviders(
      <MemoryRouter
        initialEntries={["/targets/demo"]}
        future={{ v7_startTransition: true, v7_relativeSplatPath: true }}
      >
        <WorkspaceSubNav scenario="demo" />
      </MemoryRouter>,
      { withoutRouter: true },
    );

    expect(screen.getByTestId(selectors.layout.subnav)).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.layout.workspaceSubnavLink({ key: "conflicts" })),
    ).toHaveAttribute("href", "/targets/demo/conflicts");

    for (const key of ["graph", "manifest", "apply", "analytics"] as const) {
      const chip = screen.getByTestId(selectors.layout.workspaceSubnavLink({ key }));
      expect(chip.tagName).toBe("SPAN");
      expect(chip.getAttribute("aria-disabled")).toBe("true");
    }
  });

  it("encodes the scenario in the conflicts link", () => {
    renderWithProviders(
      <MemoryRouter
        initialEntries={["/targets/demo"]}
        future={{ v7_startTransition: true, v7_relativeSplatPath: true }}
      >
        <WorkspaceSubNav scenario="needs/encoding" />
      </MemoryRouter>,
      { withoutRouter: true },
    );

    expect(
      screen.getByTestId(selectors.layout.workspaceSubnavLink({ key: "conflicts" })),
    ).toHaveAttribute("href", "/targets/needs%2Fencoding/conflicts");
  });
});
