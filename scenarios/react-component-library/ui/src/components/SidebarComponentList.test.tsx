import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { makeComponent, makeListComponentsResponse } from "../features/components/mocks/factories";
import { makeComponentsMocks } from "../features/components/mocks/components";

vi.mock("../api/components", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/components")>();
  return { ...actual, ...makeComponentsMocks() };
});

import { SidebarComponentList } from "./SidebarComponentList";

describe("SidebarComponentList", () => {
  beforeEach(() => vi.clearAllMocks());
  afterEach(() => cleanup());

  it("groups components by slot and category and collapses an individual group", async () => {
    const { componentsClient } = await import("../api/components");
    vi.mocked(componentsClient.listComponents).mockResolvedValueOnce(
      makeListComponentsResponse({
        components: [
          makeComponent({ id: "button", displayName: "Button", slot: "ui-primitive", category: "actions" }),
          makeComponent({ id: "input", displayName: "Input", slot: "ui-primitive", category: "forms" }),
          makeComponent({ id: "dialog", displayName: "Dialog", slot: "ui-pattern", category: "overlays" }),
        ],
      }),
    );

    const user = userEvent.setup();
    renderWithProviders(<SidebarComponentList />);

    const actionsGroup = await screen.findByTestId("sidebar-component-group-ui-primitive-actions");
    expect(screen.getByTestId("sidebar-component-group-ui-primitive-forms")).toBeInTheDocument();
    expect(screen.getByTestId("sidebar-component-group-ui-pattern-overlays")).toBeInTheDocument();
    expect(screen.getByTestId("sidebar-component-button")).toHaveAttribute("href", "/components/button");

    const toggle = screen.getByRole("button", { name: /ui primitive · actions/i });
    expect(toggle).toHaveAttribute("aria-expanded", "true");
    await user.click(toggle);
    await waitFor(() => expect(toggle).toHaveAttribute("aria-expanded", "false"));
    expect(actionsGroup.querySelector("ul")).toHaveClass("hidden");
  });
});
