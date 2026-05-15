import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { BottomNav } from "../../../library/components/BottomNav/versions/1.0.0/BottomNav";

describe("BottomNav", () => {
  afterEach(() => cleanup());

  it("renders link items with active page state", () => {
    render(
      <BottomNav
        label="Primary"
        items={[
          { id: "dashboard", href: "/", label: "Dashboard", icon: <span aria-hidden>1</span>, testId: "dash" },
          {
            id: "components",
            href: "/components",
            label: "Components",
            icon: <span aria-hidden>2</span>,
            active: true,
            testId: "components",
          },
        ]}
      />,
    );

    expect(screen.getByTestId("bottom-nav")).toHaveAttribute("aria-label", "Primary");
    expect(screen.getByTestId("bottom-nav").className).toContain("pb-safe");
    expect(screen.getByTestId("bottom-nav").className).toContain("pl-safe");
    expect(screen.getByTestId("bottom-nav").className).toContain("pr-safe");
    expect(screen.getByTestId("dash")).toHaveAttribute("href", "/");
    expect(screen.getByTestId("components")).toHaveAttribute("aria-current", "page");
    expect(screen.getByTestId("components").className).toContain("text-app-primary");
  });

  it("routes selection through onItemSelect", async () => {
    const onItemSelect = vi.fn();
    const user = userEvent.setup();

    render(
      <BottomNav
        label="Primary"
        onItemSelect={onItemSelect}
        items={[
          {
            id: "settings",
            href: "/settings",
            label: "Settings",
            icon: <span aria-hidden>3</span>,
            testId: "settings",
          },
        ]}
      />,
    );

    await user.click(screen.getByTestId("settings"));

    expect(onItemSelect).toHaveBeenCalledTimes(1);
    expect(onItemSelect.mock.calls[0]?.[0]).toMatchObject({ id: "settings", href: "/settings" });
  });

  it("suppresses selection for disabled items", async () => {
    const onItemSelect = vi.fn();
    const user = userEvent.setup();

    render(
      <BottomNav
        label="Primary"
        onItemSelect={onItemSelect}
        items={[
          {
            id: "disabled",
            href: "/disabled",
            label: "Disabled",
            icon: <span aria-hidden>4</span>,
            disabled: true,
            testId: "disabled",
          },
        ]}
      />,
    );

    await user.click(screen.getByTestId("disabled"));

    expect(screen.getByTestId("disabled")).toHaveAttribute("aria-disabled", "true");
    expect(onItemSelect).not.toHaveBeenCalled();
  });
});
