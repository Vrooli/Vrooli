import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Home } from "lucide-react";

import { renderWithProviders } from "../../../../../ui/src/test-utils";
import { BottomNav } from "./BottomNav.tsx";

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

  it("renders href items and prevents disabled selections", async () => {
    const user = userEvent.setup();
    const onItemSelect = vi.fn();
    renderWithProviders(
      <BottomNav
        label="Primary navigation"
        items={[
          {
            id: "home",
            label: "Home",
            icon: <Home aria-hidden />,
            href: "/",
            disabled: true,
            ariaLabel: "Home disabled",
            testId: "home-link",
          },
        ]}
        onItemSelect={onItemSelect}
        className="custom-nav"
        itemClassName="custom-item"
      />,
    );

    const link = screen.getByTestId("home-link");
    expect(link).toHaveAttribute("href", "/");
    expect(link).toHaveAttribute("aria-disabled", "true");

    await user.click(link);

    expect(onItemSelect).not.toHaveBeenCalled();
  });
});
