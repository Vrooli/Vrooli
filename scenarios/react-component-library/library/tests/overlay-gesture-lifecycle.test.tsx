import { fireEvent, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../ui/src/test-utils";
import { ViewportEnvironmentProvider, type ViewportEnvironmentSnapshot } from "@vrooli/react-component-library/useViewportEnvironment/1";
import { useOverlaySurface } from "../components/useOverlaySurface/versions/1.4.8/useOverlaySurface";

function Surface({ onClose, revision = 0, open = true }: { onClose: () => void; revision?: number; open?: boolean }) {
  const overlay = useOverlaySurface({ open, kind: "drawer", onOpenChange: onClose, dismiss: { swipe: "bottom" } });
  return <div {...overlay.rootProps} data-testid="overlay-root">
    <section {...overlay.surfaceProps} data-testid="surface">
      <button {...overlay.grabberProps} data-testid="grabber">Dismiss</button>
      <span>{revision}</span>
    </section>
  </div>;
}

function NestedSurfaces({ onChildClose }: { onChildClose: () => void }) {
  const [parentOpen, setParentOpen] = useState(true);
  const [childOpen, setChildOpen] = useState(false);
  // Deliberately pass inline controlled callbacks. Opening the child rerenders
  // this parent, matching the callback-identity change that used to reorder
  // the lower layer above the visually top drawer.
  const parent = useOverlaySurface({ open: parentOpen, onOpenChange: (next) => { setParentOpen(next); }, kind: "drawer", dismiss: { swipe: "bottom" } });
  const child = useOverlaySurface({ open: childOpen, onOpenChange: (next) => { setChildOpen(next); if (!next) onChildClose(); }, kind: "drawer", dismiss: { swipe: "bottom" } });
  return <>
    {parent.present ? <div {...parent.rootProps} data-testid="nested-parent-root">
      <section {...parent.surfaceProps} data-testid="nested-parent-surface">
        <button type="button" data-testid="nested-open-child" onClick={() => { setChildOpen(true); }}>Open child</button>
      </section>
    </div> : null}
    {child.present ? <div {...child.rootProps} data-testid="nested-child-root">
      <section {...child.surfaceProps} data-testid="nested-child-surface">
        <button type="button" {...child.grabberProps} data-testid="nested-child-grabber">Dismiss</button>
      </section>
    </div> : null}
  </>;
}

let eventTime = 0;
function pointer(target: EventTarget, type: string, y: number, id = 1) {
  const event = new Event(type, { bubbles: true, cancelable: true });
  Object.defineProperties(event, {
    timeStamp: { value: eventTime += 100 },
    pointerId: { value: id }, pointerType: { value: "touch" }, button: { value: 0 },
    clientX: { value: 50 }, clientY: { value: y },
  });
  fireEvent(target, event);
}

const viewport: ViewportEnvironmentSnapshot = {
  layoutWidth: 390, layoutHeight: 844, visibleWidth: 390, visibleHeight: 844,
  offsetLeft: 0, offsetTop: 0, scale: 1, keyboardInset: 0,
  keyboardVisible: false, reachesScreenBottom: true,
};

describe("overlay gesture lifecycle", () => {
  it("consumes focused dismissal before underlying window shortcuts can also dismiss", () => {
    const onClose = vi.fn(), underlyingShortcut = vi.fn();
    renderWithProviders(<Surface onClose={onClose} />, { withoutRouter: true });
    window.addEventListener("keydown", underlyingShortcut);
    try {
      fireEvent.keyDown(screen.getByTestId("grabber"), { key: "Escape" });
      expect(onClose).toHaveBeenCalledOnce();
      expect(underlyingShortcut).not.toHaveBeenCalled();
    } finally { window.removeEventListener("keydown", underlyingShortcut); }
  });

  it("removes closed exit surfaces from pointer hit testing immediately and restores input on reopen", () => {
    const onClose = vi.fn();
    const { rerender } = renderWithProviders(<Surface onClose={onClose} />, { withoutRouter: true });
    expect(screen.getByTestId("overlay-root")).not.toHaveAttribute("inert");
    rerender(<Surface onClose={onClose} open={false} />);
    // Presence retains the painting for the transition, but it must not eat
    // the first press on navigation or view controls underneath it.
    expect(screen.getByTestId("surface")).toHaveAttribute("data-state", "closed");
    expect(screen.getByTestId("overlay-root")).toHaveAttribute("inert");
    rerender(<Surface onClose={onClose} />);
    expect(screen.getByTestId("overlay-root")).not.toHaveAttribute("inert");
  });

  it("aborts on a real viewport change and ignores the rest of that pointer stream", () => {
    const onClose = vi.fn();
    const surface = (width: number) => <ViewportEnvironmentProvider value={{ ...viewport, visibleWidth: width }}>
      <Surface onClose={onClose} />
    </ViewportEnvironmentProvider>;
    const { rerender } = renderWithProviders(surface(390), { withoutRouter: true });
    pointer(screen.getByTestId("grabber"), "pointerdown", 0);
    pointer(window, "pointermove", 50);
    rerender(surface(390));
    expect(screen.getByTestId("surface").style.transform).toContain("50px");
    rerender(surface(844));
    expect(screen.getByTestId("surface").style.transform).toBe("");
    pointer(window, "pointermove", 150);
    pointer(window, "pointerup", 150);
    expect(onClose).not.toHaveBeenCalled();
  });

  it("does not carry an active drag through closing and reopening", () => {
    const onClose = vi.fn();
    const { rerender } = renderWithProviders(<Surface onClose={onClose} />, { withoutRouter: true });
    pointer(screen.getByTestId("grabber"), "pointerdown", 0);
    pointer(window, "pointermove", 50);
    rerender(<Surface onClose={onClose} open={false} />);
    rerender(<Surface onClose={onClose} />);
    pointer(window, "pointermove", 150);
    pointer(window, "pointerup", 150);
    expect(onClose).not.toHaveBeenCalled();
  });

  it("keeps keyboard activation available after a cancelled swipe", () => {
    const onClose = vi.fn();
    renderWithProviders(<Surface onClose={onClose} />, { withoutRouter: true });
    const grabber = screen.getByTestId("grabber");
    pointer(grabber, "pointerdown", 0);
    pointer(window, "pointermove", 40);
    pointer(window, "pointercancel", 40);
    fireEvent.click(grabber, { detail: 0 });
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("keeps a dismissing swipe alive across content updates and releases only at pointerup", () => {
    const onClose = vi.fn();
    const { rerender } = renderWithProviders(<Surface onClose={onClose} />, { withoutRouter: true });
    const grabber = screen.getByTestId("grabber");
    pointer(grabber, "pointerdown", 0);
    pointer(window, "pointermove", 50);
    expect(screen.getByTestId("surface").style.transform).toContain("50px");
    rerender(<Surface onClose={onClose} revision={1} />);
    expect(screen.getByTestId("surface").style.transform).toContain("50px");
    expect(onClose).not.toHaveBeenCalled();
    pointer(window, "pointermove", 130);
    expect(onClose).not.toHaveBeenCalled();
    pointer(window, "pointerup", 130);
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("does not sacrifice the next tap after a cancelled drag that emits no click", () => {
    const onClose = vi.fn();
    renderWithProviders(<Surface onClose={onClose} />, { withoutRouter: true });
    const grabber = screen.getByTestId("grabber");
    pointer(grabber, "pointerdown", 0);
    pointer(window, "pointermove", 40);
    pointer(window, "pointercancel", 40);
    expect(onClose).not.toHaveBeenCalled();
    pointer(grabber, "pointerdown", 40, 2);
    pointer(window, "pointerup", 40, 2);
    fireEvent.click(grabber, { detail: 1 });
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("suppresses only the click from a returned drag and accepts a later tap", () => {
    const onClose = vi.fn();
    renderWithProviders(<Surface onClose={onClose} />, { withoutRouter: true });
    const grabber = screen.getByTestId("grabber");
    pointer(grabber, "pointerdown", 0);
    pointer(window, "pointermove", 30);
    pointer(window, "pointerup", 30);
    fireEvent.click(grabber, { detail: 1 });
    expect(onClose).not.toHaveBeenCalled();
    pointer(grabber, "pointerdown", 0, 2);
    pointer(window, "pointerup", 0, 2);
    fireEvent.click(grabber, { detail: 1 });
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("keeps the nested drawer on top through a parent callback change", () => {
    const onChildClose = vi.fn();
    renderWithProviders(<NestedSurfaces onChildClose={onChildClose} />, { withoutRouter: true });
    fireEvent.click(screen.getByTestId("nested-open-child"));
    expect(screen.getByTestId("nested-parent-surface")).toHaveAttribute("data-state", "open");
    expect(screen.getByTestId("nested-child-surface")).toHaveAttribute("data-state", "open");

    const grabber = screen.getByTestId("nested-child-grabber");
    pointer(grabber, "pointerdown", 0, 11);
    pointer(window, "pointermove", 25, 11);
    pointer(window, "pointerup", 25, 11);
    expect(onChildClose).not.toHaveBeenCalled();
    expect(screen.getByTestId("nested-child-surface")).toHaveAttribute("data-state", "open");

    pointer(grabber, "pointerdown", 0, 12);
    pointer(window, "pointermove", 130, 12);
    pointer(window, "pointerup", 130, 12);
    expect(onChildClose).toHaveBeenCalledOnce();
    expect(screen.getByTestId("nested-child-surface")).toHaveAttribute("data-state", "closed");
    expect(screen.getByTestId("nested-parent-surface")).toHaveAttribute("data-state", "open");
  });
});
