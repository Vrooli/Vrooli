import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { WorkspaceSubNav } from "./WorkspaceSubNav";

afterEach(() => cleanup());

describe("WorkspaceSubNav", () => {
  it("renders every workspace section as a navigable link", () => {
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
    for (const key of ["graph", "domains", "conflicts", "apply", "analytics"] as const) {
      expect(
        screen.getByTestId(selectors.layout.workspaceSubnavLink({ key })),
      ).toHaveAttribute("href", `/targets/demo/${key}`);
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
