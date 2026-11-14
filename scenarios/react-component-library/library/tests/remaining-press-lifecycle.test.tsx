import { useRef, useState } from "react";
import { act, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../ui/src/test-utils";
import { useFocusReturn } from "../hooks/useFocusReturn/versions/1.1.2/useFocusReturn";
import { Presence } from "../primitives/Presence/versions/1.0.6/Presence";
import { SidebarShell } from "../components/SidebarShell/versions/2.7.6/SidebarShell";
import { Popover, PopoverContent, PopoverTrigger } from "@vrooli/react-component-library/Popover/1";

afterEach(() => { vi.useRealTimers(); vi.restoreAllMocks(); });

describe("retained content input ownership", () => {
  it("makes exiting descendants inert even when they explicitly opt into pointer events", () => {
    const subject = (present: boolean) => <Presence present={present} initial={false}>
      <button style={{ pointerEvents: "auto" }}>Retained action</button>
    </Presence>;
    const { rerender } = renderWithProviders(subject(true), { withoutRouter: true });
    rerender(subject(false));
    expect(screen.getByText("Retained action").closest("[inert]")).not.toBeNull();
    rerender(subject(true));
    expect(screen.getByText("Retained action").closest("[inert]")).toBeNull();
  });

  it("finishes an exit at its original deadline while parents replace callbacks", () => {
    vi.useFakeTimers();
    const oldComplete = vi.fn();
    const currentComplete = vi.fn();
    const subject = (present: boolean, callback: () => void) => <Presence present={present} initial={false} duration={180} onExitComplete={callback}>Leaving</Presence>;
    const { rerender } = renderWithProviders(subject(true, oldComplete), { withoutRouter: true });
    rerender(subject(false, oldComplete));
    act(() => { vi.advanceTimersByTime(100); });
    rerender(subject(false, currentComplete));
    act(() => { vi.advanceTimersByTime(81); });
    expect(screen.queryByText("Leaving")).not.toBeInTheDocument();
    expect(oldComplete).not.toHaveBeenCalled();
    expect(currentComplete).toHaveBeenCalledOnce();
  });

  it("removes a real popover's retained surface from input immediately", () => {
    const subject = (open: boolean) => <Popover open={open} restoreFocus={false}>
      <PopoverTrigger>Open options</PopoverTrigger>
      <PopoverContent initialFocus="none"><button>Popover action</button></PopoverContent>
    </Popover>;
    const { rerender } = renderWithProviders(subject(true), { withoutRouter: true });
    rerender(subject(false));
    expect(screen.getByText("Popover action").closest("[inert]")).not.toBeNull();
    rerender(subject(true));
    expect(screen.getByText("Popover action").closest("[inert]")).toBeNull();
  });
});

function Sidebar({ onClose = () => {} }: { onClose?: () => void }) {
  const [open, setOpen] = useState(true);
  return <SidebarShell mobileOpen={open} mode="overlay" testId="sidebar" mobileLabel="Navigation" closeLabel="Close navigation" onMobileClose={() => { onClose(); setOpen(false); }}>
    <button>Inside drawer</button>
  </SidebarShell>;
}

describe("sidebar backdrop activation", () => {
  it("dismisses on the first primary press without waiting for a synthesized click", () => {
    const onClose = vi.fn();
    renderWithProviders(<Sidebar onClose={onClose} />, { withoutRouter: true });
    fireEvent.pointerDown(screen.getByTestId("sidebar-backdrop"), { button: 0, pointerType: "touch", isPrimary: true });
    expect(screen.queryByTestId("sidebar-backdrop")).not.toBeInTheDocument();
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("leaves secondary pointer presses alone and keeps keyboard click dismissal", () => {
    const onClose = vi.fn();
    renderWithProviders(<Sidebar onClose={onClose} />, { withoutRouter: true });
    const backdrop = screen.getByTestId("sidebar-backdrop");
    fireEvent.pointerDown(backdrop, { button: 2, pointerType: "mouse", isPrimary: true });
    expect(onClose).not.toHaveBeenCalled();
    fireEvent.pointerDown(backdrop, { button: 0, pointerType: "touch", isPrimary: false });
    expect(onClose).not.toHaveBeenCalled();
    fireEvent.click(backdrop, { detail: 0 });
    expect(onClose).toHaveBeenCalledOnce();
  });
});


describe("sidebar keyboard ownership", () => {
  it("moves focus from the current text field into the drawer on open, without refocusing on content updates", () => {
    vi.useFakeTimers();
    const subject = (open: boolean, revision = 0) => <>
      <input aria-label="Terminal input" />
      <SidebarShell mobileOpen={open} mode="overlay" mobileLabel="Navigation" closeLabel="Close navigation" onMobileClose={() => {}}>
        <input aria-label="Drawer input" /><span>{revision}</span>
      </SidebarShell>
    </>;
    const { rerender } = renderWithProviders(subject(false), { withoutRouter: true });
    screen.getByRole("textbox", { name: "Terminal input" }).focus();
    rerender(subject(true));
    act(() => { vi.advanceTimersByTime(32); });
    expect(screen.getByRole("dialog", { name: "Navigation" })).toHaveFocus();
    expect(screen.getByRole("textbox", { name: "Terminal input" })).not.toHaveFocus();
    screen.getByRole("textbox", { name: "Drawer input" }).focus();
    rerender(subject(true, 1));
    expect(screen.getByRole("textbox", { name: "Drawer input" })).toHaveFocus();
  });

  it("does not steal text field focus for persistent desktop navigation", () => {
    const subject = (open: boolean) => <>
      <input aria-label="Composer" />
      <SidebarShell mobileOpen={open} mode="persistent" mobileLabel="Navigation" closeLabel="Close navigation" onMobileClose={() => {}}>Navigation</SidebarShell>
    </>;
    const { rerender } = renderWithProviders(subject(false), { withoutRouter: true });
    screen.getByRole("textbox", { name: "Composer" }).focus();
    rerender(subject(true));
    expect(screen.getByRole("textbox", { name: "Composer" })).toHaveFocus();
  });
});


function FocusOwner({ active, reference = 0 }: { active: boolean; reference?: number }) {
  const target = useRef<HTMLButtonElement>(null);
  useFocusReturn(active, reference === 0 ? target : { current: target.current });
  return <><input aria-label="Current field" /><button ref={target}>Return target {reference}</button><button>Dialog action</button></>;
}

describe("focus restoration lifecycle", () => {
  it("does not move focus when a closed surface mounts or its ref object changes", () => {
    const field = document.createElement("input");
    document.body.append(field);
    field.focus();
    try {
      const { rerender } = renderWithProviders(<FocusOwner active={false} />, { withoutRouter: true });
      expect(field).toHaveFocus();
      rerender(<FocusOwner active={false} reference={1} />);
      expect(field).toHaveFocus();
    } finally { field.remove(); }
  });

  it("captures the opener once despite ref object changes during an open lifetime", () => {
    const { rerender } = renderWithProviders(<FocusOwner active={false} />, { withoutRouter: true });
    const field = screen.getByRole("textbox", { name: "Current field" });
    field.focus();
    rerender(<FocusOwner active />);
    screen.getByRole("button", { name: "Dialog action" }).focus();
    rerender(<FocusOwner active reference={1} />);
    expect(screen.getByRole("button", { name: "Dialog action" })).toHaveFocus();
    const target = screen.getByRole("button", { name: "Return target 1" });
    const focus = vi.spyOn(target, "focus");
    rerender(<FocusOwner active={false} reference={1} />);
    expect(target).toHaveFocus();
    expect(focus).toHaveBeenCalledWith({ preventScroll: true });
    expect(focus).toHaveBeenCalledOnce();
    field.focus();
    rerender(<FocusOwner active={false} reference={2} />);
    expect(field).toHaveFocus();
  });
});


function CapturedFocusOwner({ active }: { active: boolean }) {
  useFocusReturn(active, { current: null });
  return <><input aria-label="Opener" /><button>Inner action</button></>;
}

it("does not recapture an inner control when an open surface rerenders", () => {
  const { rerender } = renderWithProviders(<CapturedFocusOwner active={false} />, { withoutRouter: true });
  const opener = screen.getByRole("textbox", { name: "Opener" });
  opener.focus();
  rerender(<CapturedFocusOwner active />);
  screen.getByRole("button", { name: "Inner action" }).focus();
  rerender(<CapturedFocusOwner active />);
  rerender(<CapturedFocusOwner active={false} />);
  expect(opener).toHaveFocus();
});


describe("sidebar exit input ownership", () => {
  it("keeps a closed sliding panel inert and restores input when reopened", () => {
    const subject = (open: boolean) => <SidebarShell mobileOpen={open} mode="overlay" mobileLabel="Navigation" closeLabel="Close navigation" onMobileClose={() => {}}>
      <button>Navigation target</button>
    </SidebarShell>;
    const { rerender } = renderWithProviders(subject(true), { withoutRouter: true });
    expect(screen.getByText("Navigation target").closest("[inert]")).toBeNull();
    rerender(subject(false));
    expect(screen.getByText("Navigation target").closest("[inert]")).not.toBeNull();
    rerender(subject(true));
    expect(screen.getByText("Navigation target").closest("[inert]")).toBeNull();
  });

  it("keeps a responsive desktop panel interactive even when its mobile drawer is closed", () => {
    const original = window.matchMedia;
    vi.spyOn(window, "matchMedia").mockImplementation((query) => ({
      ...original(query), matches: query === "(min-width: 768px)",
    }));
    renderWithProviders(<SidebarShell mobileOpen={false} mode="responsive" mobileLabel="Navigation" closeLabel="Close navigation" onMobileClose={() => {}}>
      <button>Desktop navigation</button>
    </SidebarShell>, { withoutRouter: true });
    expect(screen.getByText("Desktop navigation").closest("[inert]")).toBeNull();
    expect(screen.getByRole("complementary", { hidden: true })).toBeInTheDocument();
  });

  it("treats a responsive desktop panel as persistent after the viewport grows with the drawer open", () => {
    const original = window.matchMedia;
    vi.spyOn(window, "matchMedia").mockImplementation((query) => ({
      ...original(query), matches: query === "(min-width: 768px)",
    }));
    renderWithProviders(<SidebarShell mobileOpen mode="responsive" mobileLabel="Navigation" closeLabel="Close navigation" onMobileClose={() => {}}>
      <button>Desktop navigation</button>
    </SidebarShell>, { withoutRouter: true });
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(screen.queryByTestId("navigation.sidebar-backdrop")).not.toBeInTheDocument();
    expect(screen.getByRole("complementary")).toBeInTheDocument();
  });
});

it("waits through a hidden opening frame before taking dialog focus", () => {
  vi.useFakeTimers();
  let visible = false;
  const computedStyle = window.getComputedStyle.bind(window);
  vi.spyOn(window, "getComputedStyle").mockImplementation((element) => {
    const style = computedStyle(element);
    return element.hasAttribute("data-rcl-sidebar-shell")
      ? new Proxy(style, { get: (target, property) => property === "visibility" ? (visible ? "visible" : "hidden") : Reflect.get(target, property) })
      : style;
  });
  const subject = (open: boolean) => <>
    <input aria-label="Previous field" />
    <SidebarShell mobileOpen={open} mode="overlay" mobileLabel="Navigation" closeLabel="Close navigation" onMobileClose={() => {}}>Navigation</SidebarShell>
  </>;
  const { rerender } = renderWithProviders(subject(false), { withoutRouter: true });
  const field = screen.getByRole("textbox", { name: "Previous field" });
  field.focus();
  rerender(subject(true));
  act(() => { vi.advanceTimersByTime(16); });
  expect(field).toHaveFocus();
  visible = true;
  act(() => { vi.advanceTimersByTime(32); });
  expect(screen.getByRole("dialog", { name: "Navigation" })).toHaveFocus();
});
