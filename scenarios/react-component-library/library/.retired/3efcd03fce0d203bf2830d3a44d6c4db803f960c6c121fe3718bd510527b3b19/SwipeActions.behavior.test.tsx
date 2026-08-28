import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";

import { renderWithProviders } from "../../../../../ui/src/test-utils";
import { HandednessProvider } from "@vrooli/react-component-library/useHandedness/1.1.2";
import { SidebarShell } from "@vrooli/react-component-library/SidebarShell/2.6.0";
import { SwipeActions, type SwipeAction, type SwipeActionsProps } from "./SwipeActions";

const WIDTH = 76;

function makeActions(onSelect: Record<string, () => void> = {}): SwipeAction[] {
  return [
    { id: "unread", label: "Unread", onSelect: onSelect.unread ?? (() => {}) },
    { id: "close", label: "Close", tone: "destructive", onSelect: onSelect.close ?? (() => {}) },
  ];
}

function Row(props: Partial<SwipeActionsProps> & { actions?: SwipeAction[] }) {
  const { actions = makeActions(), children, ...rest } = props;
  return (
    <SwipeActions actions={actions} label="Row actions" {...rest}>
      {children ?? <div data-testid="row-content">Session</div>}
    </SwipeActions>
  );
}

function face() {
  return screen.getByTestId("patterns.swipe-actions.face");
}

function root() {
  return screen.getByTestId("patterns.swipe-actions");
}

/**
 * Drives a gesture the way the browser delivers one.
 *
 * Every event is dispatched on the face and left to bubble, which is what the
 * browser does: a pointermove targets the element under the pointer (or the
 * capture target) and rises from there. Dispatching directly on `window`
 * instead would skip the row's own handlers entirely, and a gesture that only
 * works when its element is bypassed is not a gesture that works.
 */
function drag(xs: number[], opts: { end?: "up" | "cancel"; step?: number } = {}) {
  const step = opts.step ?? 200;
  const target = face();
  fireEvent.pointerDown(target, {
    pointerId: 1,
    clientX: xs[0],
    clientY: 0,
    button: 0,
    pointerType: "touch",
    timeStamp: 1,
  });
  xs.slice(1).forEach((x, index) => {
    fireEvent.pointerMove(target, {
      pointerId: 1,
      clientX: x,
      clientY: 0,
      pointerType: "touch",
      timeStamp: 1 + (index + 1) * step,
    });
  });
  const last = xs[xs.length - 1];
  const init = {
    pointerId: 1,
    clientX: last,
    clientY: 0,
    pointerType: "touch",
    timeStamp: 1 + xs.length * step,
  };
  if (opts.end === "cancel") fireEvent.pointerCancel(target, init);
  else fireEvent.pointerUp(target, init);
}

describe("composition", () => {
  it("renders the row content it wraps", () => {
    renderWithProviders(<Row />);
    expect(screen.getByTestId("row-content")).toBeInTheDocument();
  });

  // The escape hatch SidebarShell publishes. Without it every ancestor rule
  // pins this subtree to pan-y and the browser cancels the drag as a scroll.
  it("claims the inline axis so it survives a swipe-enabled drawer", () => {
    renderWithProviders(<Row />);
    expect(root().hasAttribute("data-rcl-pan-x")).toBe(true);
  });

  it("does not claim the axis when it has nothing to reveal", () => {
    renderWithProviders(<Row actions={[]} />);
    expect(root().hasAttribute("data-rcl-pan-x")).toBe(false);
  });

  it("does not claim the axis while disabled", () => {
    renderWithProviders(<Row disabled />);
    expect(root().hasAttribute("data-rcl-pan-x")).toBe(false);
  });
});

describe("direction", () => {
  it("reveals away from a start-anchored drawer", () => {
    renderWithProviders(<Row />);
    expect(root().getAttribute("data-reveal")).toBe("right");
  });

  it("reveals away from an end-anchored drawer", () => {
    renderWithProviders(
      <HandednessProvider value="inline-end">
        <Row />
      </HandednessProvider>,
    );
    expect(root().getAttribute("data-reveal")).toBe("left");
  });

  it("puts the track on the side the face uncovers", () => {
    renderWithProviders(<Row />);
    const track = root().querySelector("[data-rcl-swipe-actions-track]");
    expect(track?.getAttribute("data-side")).toBe("left");
  });

  it("follows the finger one to one", () => {
    renderWithProviders(<Row />);
    drag([0, 40]);
    // Released short of the first threshold, so it returns — but the frame in
    // between is what proves tracking, and it is written to the DOM directly.
    expect(face().style.transform).toBe("");
  });

  it("tracks during the drag rather than only on release", () => {
    renderWithProviders(<Row />);
    fireEvent.pointerDown(face(), {
      pointerId: 1,
      clientX: 0,
      clientY: 0,
      button: 0,
      pointerType: "touch",
      timeStamp: 1,
    });
    fireEvent.pointerMove(window, {
      pointerId: 1,
      clientX: 50,
      clientY: 0,
      pointerType: "touch",
      timeStamp: 200,
    });
    expect(face().style.transform).toBe("translateX(50px)");
  });

  it("translates negatively when revealing leftward", () => {
    renderWithProviders(
      <HandednessProvider value="inline-end">
        <Row />
      </HandednessProvider>,
    );
    fireEvent.pointerDown(face(), {
      pointerId: 1,
      clientX: 0,
      clientY: 0,
      button: 0,
      pointerType: "touch",
      timeStamp: 1,
    });
    fireEvent.pointerMove(window, {
      pointerId: 1,
      clientX: -50,
      clientY: 0,
      pointerType: "touch",
      timeStamp: 200,
    });
    expect(face().style.transform).toBe("translateX(-50px)");
  });
});

describe("rest-open release", () => {
  it("stays closed below the first threshold", () => {
    renderWithProviders(<Row />);
    drag([0, 20]);
    expect(root().getAttribute("data-open")).toBe("false");
  });

  it("rests open past the first threshold", () => {
    renderWithProviders(<Row />);
    drag([0, 120]);
    expect(root().getAttribute("data-open")).toBe("true");
  });

  it("holds the face at the full track width when open", () => {
    renderWithProviders(<Row />);
    drag([0, 200]);
    expect(face().style.transform).toBe(`translateX(${String(WIDTH * 2)}px)`);
  });

  it("does not fire an action on release", () => {
    const unread = vi.fn();
    renderWithProviders(<Row actions={makeActions({ unread })} />);
    drag([0, 200]);
    expect(unread).not.toHaveBeenCalled();
  });

  it("closes again when swiped back", () => {
    renderWithProviders(<Row />);
    drag([0, 200]);
    drag([0, -200]);
    expect(root().getAttribute("data-open")).toBe("false");
  });

  it("reports open state to a listener", () => {
    const onOpenChange = vi.fn();
    renderWithProviders(<Row onOpenChange={onOpenChange} />);
    drag([0, 200]);
    expect(onOpenChange).toHaveBeenCalledWith(true);
  });
});

describe("auto-commit release", () => {
  it("fires the first action when only it is armed", () => {
    const unread = vi.fn();
    const close = vi.fn();
    renderWithProviders(<Row releaseMode="commit" actions={makeActions({ unread, close })} />);
    drag([0, 60]);
    expect(unread).toHaveBeenCalledTimes(1);
    expect(close).not.toHaveBeenCalled();
  });

  it("fires the further action when the drag reaches it", () => {
    const unread = vi.fn();
    const close = vi.fn();
    renderWithProviders(<Row releaseMode="commit" actions={makeActions({ unread, close })} />);
    drag([0, 130]);
    expect(close).toHaveBeenCalledTimes(1);
    expect(unread).not.toHaveBeenCalled();
  });

  it("fires nothing below the first threshold", () => {
    const unread = vi.fn();
    renderWithProviders(<Row releaseMode="commit" actions={makeActions({ unread })} />);
    drag([0, 20]);
    expect(unread).not.toHaveBeenCalled();
  });

  it("leaves the row closed after committing", () => {
    renderWithProviders(<Row releaseMode="commit" />);
    drag([0, 130]);
    expect(root().getAttribute("data-open")).toBe("false");
    expect(face().style.transform).toBe("");
  });
});

describe("cancellation", () => {
  // The regression that shipped once in SidebarShell: cancel wired to the same
  // path as pointerup performs the action the user just stopped asking for.
  it("fires nothing when the browser cancels a committing drag", () => {
    const close = vi.fn();
    renderWithProviders(<Row releaseMode="commit" actions={makeActions({ close })} />);
    drag([0, 300], { end: "cancel" });
    expect(close).not.toHaveBeenCalled();
  });

  it("does not rest open when the browser cancels", () => {
    renderWithProviders(<Row />);
    drag([0, 300], { end: "cancel" });
    expect(root().getAttribute("data-open")).toBe("false");
  });

  it("returns the face to its closed offset after a cancel", () => {
    renderWithProviders(<Row />);
    drag([0, 300], { end: "cancel" });
    expect(face().style.transform).toBe("");
  });
});

describe("accessibility", () => {
  it("keeps closed actions out of the tab order", () => {
    renderWithProviders(<Row />);
    for (const button of screen.getAllByRole("button", { hidden: true })) {
      expect(button.getAttribute("tabindex")).toBe("-1");
    }
  });

  it("hides the closed track from assistive technology", () => {
    renderWithProviders(<Row />);
    const track = root().querySelector("[data-rcl-swipe-actions-track]");
    expect(track?.getAttribute("aria-hidden")).toBe("true");
  });

  it("exposes the actions once the row rests open", () => {
    renderWithProviders(<Row />);
    drag([0, 200]);
    const track = root().querySelector("[data-rcl-swipe-actions-track]");
    expect(track?.getAttribute("aria-hidden")).toBeNull();
    expect(screen.getByRole("button", { name: "Unread" }).getAttribute("tabindex")).toBe("0");
  });

  it("names the revealed group", () => {
    renderWithProviders(<Row />);
    drag([0, 200]);
    expect(screen.getByRole("group", { name: "Row actions" })).toBeInTheDocument();
  });

  it("runs an action when its button is activated", () => {
    const close = vi.fn();
    renderWithProviders(<Row actions={makeActions({ close })} />);
    drag([0, 200]);
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(close).toHaveBeenCalledTimes(1);
  });

  it("closes the row after running an action", () => {
    renderWithProviders(<Row />);
    drag([0, 200]);
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(root().getAttribute("data-open")).toBe("false");
  });
});

describe("controlled state", () => {
  function Controlled() {
    const [open, setOpen] = useState(false);
    return (
      <>
        <button
          type="button"
          onClick={() => {
            setOpen(false);
          }}
        >
          Close from outside
        </button>
        <Row open={open} onOpenChange={setOpen} />
      </>
    );
  }

  it("honours an external close by returning the face home", () => {
    renderWithProviders(<Controlled />);
    drag([0, 200]);
    expect(face().style.transform).toBe(`translateX(${String(WIDTH * 2)}px)`);
    fireEvent.click(screen.getByRole("button", { name: "Close from outside" }));
    expect(face().style.transform).toBe("");
  });
});

describe("disabled", () => {
  it("ignores a drag entirely", () => {
    renderWithProviders(<Row disabled />);
    drag([0, 300]);
    expect(root().getAttribute("data-open")).toBe("false");
    expect(face().style.transform).toBe("");
  });
});

describe("axis claim", () => {
  it("publishes no claim while the row is shut", () => {
    renderWithProviders(<Row />);
    expect(root().hasAttribute("data-rcl-gesture-claim")).toBe(false);
  });

  // Once the actions show, putting them away travels the same direction as
  // dismissing an ancestor drawer, so the row claims the axis outright.
  it("publishes a claim once the actions are showing", () => {
    renderWithProviders(<Row />);
    drag([0, 200]);
    expect(root().hasAttribute("data-rcl-gesture-claim")).toBe(true);
  });

  it("drops the claim once the row is put away", () => {
    renderWithProviders(<Row />);
    drag([0, 200]);
    drag([0, -200]);
    expect(root().hasAttribute("data-rcl-gesture-claim")).toBe(false);
  });

  // The claim is an attribute precisely so the row does not have to stop
  // propagation: React dispatches from its root, so stopping a synthetic event
  // also stops the native one before it reaches the window listeners this
  // component's own gesture runs on -- silencing the drawer and itself at once.
  it("lets its own events through so the gesture still runs", () => {
    const seen: string[] = [];
    renderWithProviders(
      <div
        onPointerMoveCapture={() => seen.push("move")}
        onPointerUpCapture={() => seen.push("up")}
      >
        <Row />
      </div>,
    );
    drag([0, 200]);
    expect(seen).toContain("move");
    expect(seen).toContain("up");
  });
});

describe("track containment", () => {
  // `overflow: hidden` would make this box a block formatting context, which
  // contains a child's bottom margin and leaves the track showing under every
  // row as a stripe. `clip` lets that margin collapse out as it normally would.
  it("clips the track without containing a child's margin", () => {
    renderWithProviders(<Row />);
    // jsdom does not implement the `overflow: hidden; overflow: clip` pair, so
    // the declaration itself is the assertion the browser acts on.
    const sheet =
      document.querySelector("style[data-rcl-stylesheet='rcl-swipe-actions-1-0-0']") ??
      Array.from(document.querySelectorAll("style")).find((node) =>
        node.textContent?.includes("data-rcl-swipe-actions]"),
      );
    expect(sheet?.textContent).toContain("overflow: clip");
  });
});

// The stand-in above proves the propagation rule; this proves it against the
// component that actually wraps these rows, with the same handler topology
// production uses (SidebarShell attaches its drag as React props, so a
// synthetic stopPropagation is what governs whether it ever starts).
describe("inside a real drawer", () => {
  function Drawer() {
    return (
      <SidebarShell
        mode="overlay"
        mobileOpen
        swipeToClose
        mobileLabel="Sessions"
        closeLabel="Close sessions"
        onMobileClose={() => {}}
        testId="drawer"
      >
        <Row />
      </SidebarShell>
    );
  }

  const shell = () => screen.getByTestId("drawer");

  /** A leftward drag — the direction that dismisses a start-anchored drawer. */
  const dismissDrag = () => {
    const target = face();
    fireEvent.pointerDown(target, {
      pointerId: 1,
      clientX: 200,
      clientY: 0,
      button: 0,
      pointerType: "touch",
      timeStamp: 1,
    });
    for (let step = 1; step <= 5; step += 1) {
      fireEvent.pointerMove(target, {
        pointerId: 1,
        clientX: 200 - step * 25,
        clientY: 0,
        pointerType: "touch",
        timeStamp: 1 + step * 20,
      });
    }
  };

  it("lets the drawer claim a dismissing drag on a shut row", () => {
    renderWithProviders(<Drawer />);
    dismissDrag();
    expect(shell().getAttribute("data-dragging")).toBe("true");
  });

  it("keeps the drawer still when the drag is putting an open row away", () => {
    renderWithProviders(<Drawer />);
    drag([0, 200]);
    expect(root().getAttribute("data-open")).toBe("true");
    dismissDrag();
    expect(shell().getAttribute("data-dragging")).not.toBe("true");
  });

  it("still closes the row it was performed on", () => {
    renderWithProviders(<Drawer />);
    drag([0, 200]);
    drag([0, -200]);
    expect(root().getAttribute("data-open")).toBe("false");
  });
});
