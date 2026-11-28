import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AppShell } from "@vrooli/react-component-library/AppShell/2";

const viewport = vi.hoisted(() => ({ desktop: false }));
vi.mock("@vrooli/react-component-library/useMediaQuery/1", async (original) => ({
  ...await original<object>(),
  useBreakpoint: () => viewport.desktop,
}));

const items = [
  { id: "home", label: "Home", href: "/", icon: null, testId: "home-link" },
  { id: "settings", label: "Settings", href: "/settings", icon: null, mobile: false, testId: "settings-link" },
];

afterEach(() => { cleanup(); viewport.desktop = false; });

describe("AppShell utility reachability", () => {
  it("keeps session and theme controls in the shell-owned phone header", () => {
    const signIn = vi.fn();
    render(<AppShell brand="Example" items={items} utility={<><button onClick={signIn}>Sign in</button><select aria-label="Theme"><option>Dark</option></select></>}><p>Content</p></AppShell>);
    // jsdom does not apply viewport media rules; browser validation checks
    // visibility. This test checks the slot location and retained callbacks.
    const button = screen.getByText("Sign in");
    expect(button.closest("[data-rcl-app-shell-header]")).not.toBeNull();
    expect(screen.getByLabelText("Theme").closest("[data-rcl-app-shell-header]")).not.toBeNull();
    fireEvent.click(button);
    expect(signIn).toHaveBeenCalledOnce();
    expect(screen.queryByTestId("settings-link-tab")).toBeNull();
    expect(screen.getByTestId("home-link-tab")).toBeInTheDocument();
    expect(screen.getByTestId("settings-link")).toBeInTheDocument();
  });

  it("keeps utility in the desktop column without a redundant header", () => {
    viewport.desktop = true;
    const { container } = render(<AppShell brand="Example" items={items} utility={<button>Sign in</button>}><p>Content</p></AppShell>);
    expect(screen.getByRole("button", { name: "Sign in" }).closest("[data-rcl-app-shell-utility]")).not.toBeNull();
    expect(container.querySelector("[data-rcl-app-shell-header]")).toBeNull();
  });

  it("preserves the header-free default and skip navigation", () => {
    const { container } = render(<AppShell brand="Example" items={items} skipLabel="Skip to content"><p>Content</p></AppShell>);
    expect(container.querySelector("[data-rcl-app-shell-header]")).toBeNull();
    expect(screen.getByRole("link", { name: "Skip to content" })).toHaveAttribute("href", `#${screen.getByRole("main").id}`);
    expect(screen.getAllByRole("navigation").length).toBeGreaterThan(0);
  });
});
