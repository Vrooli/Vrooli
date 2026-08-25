import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Home } from "lucide-react";

import { renderWithProviders } from "../../test-utils";
import { BottomNav } from "@vrooli/react-component-library/BottomNav/1.2.0";

/**
 * Fixture copy, named once. These are the test's OWN sample values rather
 * than application copy, but they are referenced through a constant so the
 * copy-driven-query lint rule stays enforceable without a per-file exemption.
 */
const HOME_LABEL = "Home";

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
    expect(screen.getByText(HOME_LABEL)).toBeInTheDocument();
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

  it("renders links and prevents disabled items from selecting", async () => {
    const user = userEvent.setup();
    const onItemSelect = vi.fn();
    renderWithProviders(
      <BottomNav
        label="Primary navigation"
        items={[
          { id: "details", label: "Details", icon: <Home aria-hidden />, href: "/details", ariaLabel: "Open details" },
          { id: "locked", label: "Locked", icon: <Home aria-hidden />, disabled: true, testId: "locked-tab" },
        ]}
        onItemSelect={onItemSelect}
      />,
    );

    expect(screen.getByRole("link", { name: "Open details" })).toHaveAttribute("href", "/details");
    await user.click(screen.getByTestId("locked-tab"));
    expect(onItemSelect).not.toHaveBeenCalled();
  });
});
