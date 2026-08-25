/* eslint-disable no-restricted-syntax */
import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Home } from "lucide-react";

import { renderWithProviders } from "../../test-utils";
import { BottomNav } from "@vrooli/react-component-library/BottomNav/1.2.0";

describe("BottomNav", () => {
  it("renders labels and marks the active item", () => {
    renderWithProviders(
      <BottomNav
        label="Primary navigation"
        items={[
          { id: "home", label: "Home", icon: <Home aria-hidden />, active: true, testId: "home-tab" },
        ]}
      />,
    );

    expect(screen.getByTestId("home-tab")).toHaveAttribute("aria-current", "page");
    expect(screen.getByText("Home")).toBeInTheDocument();
  });

  it("calls onItemSelect without navigating for controlled items", async () => {
    const user = userEvent.setup();
    const onItemSelect = vi.fn();
    renderWithProviders(
      <BottomNav
        label="Primary navigation"
        items={[
          { id: "home", label: "Home", icon: <Home aria-hidden />, testId: "home-tab" },
        ]}
        onItemSelect={onItemSelect}
      />,
    );

    await user.click(screen.getByTestId("home-tab"));

    expect(onItemSelect).toHaveBeenCalledTimes(1);
    expect(onItemSelect.mock.calls[0]?.[0]).toMatchObject({ id: "home" });
  });

  it("supports links and prevents disabled items from invoking handlers", async () => {
    const user = userEvent.setup();
    const onItemSelect = vi.fn();
    renderWithProviders(
      <BottomNav
        label="Primary navigation"
        items={[
          { id: "link", label: "Link", icon: <Home aria-hidden />, href: "/link", testId: "link-tab" },
          { id: "disabled", label: "Disabled", icon: <Home aria-hidden />, disabled: true, testId: "disabled-tab" },
        ]}
        onItemSelect={onItemSelect}
      />,
    );
    await user.click(screen.getByTestId("link-tab"));
    await user.click(screen.getByTestId("disabled-tab"));
    expect(onItemSelect).toHaveBeenCalledTimes(1);
  });
});
