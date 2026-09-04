import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../../../../ui/src/test-utils";
import { Menu, MenuContent, MenuItem, MenuSubmenu, MenuTrigger } from "../../../../components/Menu/versions/2.0.0/Menu";

describe("Menu submenu gesture contract", () => {
  it("defaults to the existing click trigger and accepts hover as an accelerator", () => {
    renderWithProviders(
      <Menu defaultOpen>
        <MenuTrigger>Open</MenuTrigger>
        <MenuContent>
          <MenuSubmenu label="More">
            <MenuItem>Child</MenuItem>
          </MenuSubmenu>
          <MenuSubmenu label="Hover" trigger="hover">
            <MenuItem>Child</MenuItem>
          </MenuSubmenu>
        </MenuContent>
      </Menu>,
    );
    expect(screen.getAllByRole("menu").length).toBeGreaterThan(0);
    expect(document.querySelector('[data-rcl-menu-submenu][data-trigger="hover"]')).toBeTruthy();
  });

  it("keeps the keyboard route available", () => {
    renderWithProviders(
      <Menu defaultOpen>
        <MenuTrigger>Open</MenuTrigger>
        <MenuContent><MenuSubmenu label="More"><MenuItem>Child</MenuItem></MenuSubmenu></MenuContent>
      </Menu>,
    );
    const trigger = screen.getByRole("menuitem", { name: "More" });
    fireEvent.keyDown(trigger, { key: "ArrowRight" });
    expect(trigger).toHaveAttribute("aria-haspopup", "menu");
  });
});
