import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Home } from "lucide-react";

import { renderWithProviders } from "../../test-utils";
import { BottomNav } from "./bottom-nav";

describe("BottomNav", () => {
  it("renders labels and marks the active item", async () => {
    const user = userEvent.setup();
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
    await user.click(screen.getByTestId("home-tab"));
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

  it("supports links and prevents disabled items from invoking the callback", async () => {
    const user = userEvent.setup();
    const onItemSelect = vi.fn();
    renderWithProviders(
      <BottomNav
        label="Primary navigation"
        items={[
          { id: "docs", label: "Docs", icon: <Home aria-hidden />, href: "/docs", testId: "docs-link" },
          { id: "locked", label: "Locked", icon: <Home aria-hidden />, disabled: true, testId: "locked-button" },
        ]}
        onItemSelect={onItemSelect}
      />,
    );

    expect(screen.getByTestId("docs-link")).toHaveAttribute("href", "/docs");
    await user.click(screen.getByTestId("docs-link"));
    expect(onItemSelect).toHaveBeenCalledWith(expect.objectContaining({ id: "docs" }), expect.anything());
    await user.click(screen.getByTestId("locked-button"));
    expect(onItemSelect).toHaveBeenCalledTimes(1);
  });
});
