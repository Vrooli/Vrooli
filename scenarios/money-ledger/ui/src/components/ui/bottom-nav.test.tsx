import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Home } from "lucide-react";

import { renderWithProviders } from "../../test-utils";
import { BottomNav } from "./bottom-nav";

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
    expect(screen.getByText(/Home/)).toBeInTheDocument();
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

  it("supports link items and prevents selection of disabled links", async () => {
    const user = userEvent.setup();
    const onItemSelect = vi.fn();
    renderWithProviders(
      <BottomNav
        label="Primary navigation"
        items={[
          { id: "disabled", label: "Disabled", icon: <Home aria-hidden />, href: "/disabled", disabled: true, testId: "disabled-link" },
          { id: "settings", label: "Settings", icon: <Home aria-hidden />, href: "/settings", testId: "settings-link" },
        ]}
        onItemSelect={onItemSelect}
      />,
    );

    await user.click(screen.getByTestId("disabled-link"));
    await user.click(screen.getByTestId("settings-link"));

    expect(onItemSelect).toHaveBeenCalledTimes(1);
    expect(onItemSelect.mock.calls[0]?.[0]).toMatchObject({ id: "settings", href: "/settings" });
    expect(screen.getByTestId("disabled-link")).toHaveAttribute("aria-disabled", "true");
  });
});
