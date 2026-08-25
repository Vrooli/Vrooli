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

  it("renders links and prevents disabled-item selection", async () => {
    const user = userEvent.setup();
    const onItemSelect = vi.fn();
    renderWithProviders(
      <BottomNav
        label="Primary navigation"
        items={[
          { id: "journal", label: "Journal", icon: <Home aria-hidden />, href: "/journal", ariaLabel: "Open journal", testId: "journal-tab" },
          { id: "locked", label: "Locked", icon: <Home aria-hidden />, href: "/locked", disabled: true, testId: "locked-tab" },
        ]}
        onItemSelect={onItemSelect}
      />,
    );

    expect(screen.getByTestId("journal-tab")).toHaveAttribute("href", "/journal");
    expect(screen.getByTestId("locked-tab")).toHaveAttribute("aria-disabled", "true");
    await user.click(screen.getByTestId("locked-tab"));
    expect(onItemSelect).not.toHaveBeenCalled();
  });
});
