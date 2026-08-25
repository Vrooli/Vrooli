import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Home } from "lucide-react";

import { renderWithProviders } from "../../test-utils";
import { BottomNav } from "@vrooli/react-component-library/BottomNav/1.2.0";

const homeLabel = "Home";

describe("BottomNav", () => {
  it("renders labels and marks the active item", () => {
    renderWithProviders(
      <BottomNav
        label="Primary navigation"
        items={[
          { id: "home", label: homeLabel, icon: <Home aria-hidden />, active: true, testId: "home-tab" },
        ]}
      />,
    );

    expect(screen.getByTestId("home-tab")).toHaveAttribute("aria-current", "page");
    expect(screen.getByText(homeLabel)).toBeInTheDocument();
    expect(screen.getByRole("navigation")).toHaveClass("fixed", "bottom-0");
  });

  it("calls onItemSelect without navigating for controlled items", async () => {
    const user = userEvent.setup();
    const onItemSelect = vi.fn();
    renderWithProviders(
      <BottomNav
        label="Primary navigation"
        items={[
          { id: "home", label: homeLabel, icon: <Home aria-hidden />, testId: "home-tab" },
        ]}
        onItemSelect={onItemSelect}
      />,
    );

    await user.click(screen.getByTestId("home-tab"));

    expect(onItemSelect).toHaveBeenCalledTimes(1);
    expect(onItemSelect.mock.calls[0]?.[0]).toMatchObject({ id: "home" });
  });

  it("renders href items and prevents selection when an item is disabled", async () => {
    const user = userEvent.setup();
    const onItemSelect = vi.fn();
    renderWithProviders(
      <BottomNav
        label="Primary navigation"
        className="custom-nav"
        activeItemClassName="custom-active"
        inactiveItemClassName="custom-inactive"
        itemClassName="custom-item"
        items={[
          { id: "home", label: homeLabel, icon: <Home aria-hidden />, href: "/", active: true, testId: "home-link" },
          { id: "disabled", label: "Disabled", icon: <Home aria-hidden />, href: "/disabled", disabled: true, testId: "disabled-link" },
        ]}
        onItemSelect={onItemSelect}
      />,
    );

    expect(screen.getByRole("navigation")).toHaveClass("custom-nav");
    expect(screen.getByTestId("home-link")).toHaveAttribute("href", "/");
    expect(screen.getByTestId("home-link")).toHaveClass("custom-active", "custom-item");
    expect(screen.getByTestId("disabled-link")).toHaveAttribute("aria-disabled", "true");
    expect(screen.getByTestId("disabled-link")).toHaveClass("custom-inactive");

    await user.click(screen.getByTestId("disabled-link"));
    expect(onItemSelect).not.toHaveBeenCalled();
  });
});
