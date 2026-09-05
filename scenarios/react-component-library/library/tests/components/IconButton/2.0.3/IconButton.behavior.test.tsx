import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { createRef, useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, screen } from "@testing-library/react";

import { renderWithProviders } from "../../../../../ui/src/test-utils";
import { IconButton } from "../../../../components/IconButton/versions/2.0.3/IconButton.tsx";
import { clearIconMorphCache } from "@vrooli/react-component-library/useIconMorph/1";

/**
 * Two lucide-shaped icons. They are declared as distinct module-scope
 * components on purpose: the swap detection keys off component identity, which
 * is exactly how every icon library is built.
 */
function BubbleIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
      <path d="M13 8H7" />
      <path d="M17 12H7" />
    </svg>
  );
}

function TerminalIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="m7 11 2-2-2-2" />
      <path d="M11 13h4" />
      <rect width="18" height="18" x="3" y="3" rx="2" ry="2" />
    </svg>
  );
}

function SunIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2" />
      <path d="M12 20v2" />
      <path d="M2 12h2" />
      <path d="M20 12h2" />
      <path d="m4.93 4.93 1.41 1.41" />
      <path d="m17.66 17.66 1.41 1.41" />
      <path d="m6.34 17.66-1.41 1.41" />
      <path d="m19.07 4.93-1.41 1.41" />
    </svg>
  );
}

const __dirname = dirname(fileURLToPath(import.meta.url));

const button = () => screen.getByTestId("controls.icon-button");
const glyph = () => document.querySelector("[data-rcl-morphing-icon]")!;

beforeEach(() => {
  clearIconMorphCache();
});

describe("surface and shape defaults", () => {
  it("is circular and ghost with no props but a label and an icon", () => {
    renderWithProviders(
      <IconButton aria-label="Close">
        <BubbleIcon />
      </IconButton>,
    );
    expect(button()).toHaveAttribute("data-rcl-shape", "circle");
    expect(button()).toHaveAttribute("data-rcl-surface", "ghost");
  });

  it("reaches the standing surface with one prop", () => {
    renderWithProviders(
      <IconButton aria-label="Toggle" surface="soft">
        <BubbleIcon />
      </IconButton>,
    );
    expect(button()).toHaveAttribute("data-rcl-surface", "soft");
  });

  it("exposes the other two shapes", () => {
    const { rerender } = renderWithProviders(
      <IconButton aria-label="A" shape="rounded">
        <BubbleIcon />
      </IconButton>,
    );
    expect(button()).toHaveAttribute("data-rcl-shape", "rounded");
    rerender(
      <IconButton aria-label="A" shape="square">
        <BubbleIcon />
      </IconButton>,
    );
    expect(button()).toHaveAttribute("data-rcl-shape", "square");
  });

  it("keeps the control box square so a circle is not a stadium", () => {
    renderWithProviders(
      <IconButton aria-label="Close">
        <BubbleIcon />
      </IconButton>,
    );
    const { inlineSize, blockSize } = button().style;
    expect(inlineSize).toBe(blockSize);
    expect(inlineSize).not.toBe("");
  });
});

describe("legacy variant migration", () => {
  // 2.x call sites pass `variant`. They must keep working, and they must land
  // on the surface that matches what they were compensating for: both
  // web-console adopters passed `secondary` to escape the broken ghost hover.
  it.each([
    ["secondary", "soft"],
    ["outline", "soft"],
    ["primary", "solid"],
    ["default", "solid"],
    ["danger", "danger"],
    ["destructive", "danger"],
    ["error", "danger"],
    ["ghost", "ghost"],
  ] as const)("maps variant=%s onto surface=%s", (variant, surface) => {
    renderWithProviders(
      <IconButton aria-label="A" variant={variant}>
        <BubbleIcon />
      </IconButton>,
    );
    expect(button()).toHaveAttribute("data-rcl-surface", surface);
  });

  it("lets an explicit surface win over a legacy variant", () => {
    renderWithProviders(
      <IconButton aria-label="A" variant="primary" surface="ghost">
        <BubbleIcon />
      </IconButton>,
    );
    expect(button()).toHaveAttribute("data-rcl-surface", "ghost");
  });
});

describe("toggle state", () => {
  it("announces a pressed toggle rather than only colouring it", () => {
    renderWithProviders(
      <IconButton aria-label="Grid" selected>
        <BubbleIcon />
      </IconButton>,
    );
    expect(button()).toHaveAttribute("aria-pressed", "true");
  });

  it("announces an unpressed toggle when selected is false", () => {
    renderWithProviders(
      <IconButton aria-label="Grid" selected={false}>
        <BubbleIcon />
      </IconButton>,
    );
    expect(button()).toHaveAttribute("aria-pressed", "false");
  });

  it("stays a plain action when selected is omitted", () => {
    // A plain button must not announce itself as an unpressed toggle.
    renderWithProviders(
      <IconButton aria-label="Send">
        <BubbleIcon />
      </IconButton>,
    );
    expect(button()).not.toHaveAttribute("aria-pressed");
  });
});

describe("naming and tooltip", () => {
  it("requires no separate title to get one", () => {
    renderWithProviders(
      <IconButton aria-label="Close panel">
        <BubbleIcon />
      </IconButton>,
    );
    expect(button()).toHaveAttribute("title", "Close panel");
    expect(button()).toHaveAccessibleName("Close panel");
  });

  it("keeps an explicit title distinct from the label", () => {
    renderWithProviders(
      <IconButton aria-label="Close panel" title="Esc">
        <BubbleIcon />
      </IconButton>,
    );
    expect(button()).toHaveAttribute("title", "Esc");
    expect(button()).toHaveAccessibleName("Close panel");
  });

  it("suppresses the native tooltip on request", () => {
    renderWithProviders(
      <IconButton aria-label="Close" disableTooltip>
        <BubbleIcon />
      </IconButton>,
    );
    expect(button()).not.toHaveAttribute("title");
  });
});

describe("pending", () => {
  it("disables, marks busy, and hides the glyph behind a spinner", () => {
    renderWithProviders(
      <IconButton aria-label="Save" pending pendingLabel="Saving…">
        <BubbleIcon />
      </IconButton>,
    );
    expect(button()).toBeDisabled();
    expect(button()).toHaveAttribute("aria-busy", "true");
    expect(button()).toHaveAttribute("data-rcl-pending", "true");
    expect(document.querySelector("[data-rcl-icon-button-spinner]")).not.toBeNull();
    expect(screen.getByText("Saving…")).toBeInTheDocument();
  });

  /**
   * A caller whose icon changes as the pending window opens — "switch the view,
   * then load it" — must still be seen to change. 3.0.1 hid the glyph outright
   * and swallowed the whole animation.
   */
  it("keeps the glyph rendered while pending so a coincident swap is visible", () => {
    renderWithProviders(
      <IconButton aria-label="Save" pending>
        <BubbleIcon />
      </IconButton>,
    );
    const glyph = document.querySelector("[data-rcl-icon-button-glyph]");
    expect(glyph).not.toBeNull();
    expect(glyph!.querySelector("svg")).not.toBeNull();
    expect(document.querySelector("[data-rcl-morphing-icon]")).not.toBeNull();
  });

  it("does not fire while pending", () => {
    const onClick = vi.fn();
    renderWithProviders(
      <IconButton aria-label="Save" pending onClick={onClick}>
        <BubbleIcon />
      </IconButton>,
    );
    fireEvent.click(button());
    expect(onClick).not.toHaveBeenCalled();
  });
});

describe("size scale", () => {
  it.each(["xs", "sm", "md", "lg"] as const)("exposes the %s rung", (size) => {
    renderWithProviders(
      <IconButton aria-label="A" size={size}>
        <BubbleIcon />
      </IconButton>,
    );
    expect(button()).toHaveAttribute("data-rcl-size", size);
    expect(button().style.inlineSize).toBe(button().style.blockSize);
  });

  it("keeps xs on the 32px control token dense toolbars are built on", () => {
    renderWithProviders(
      <IconButton aria-label="A" size="xs">
        <BubbleIcon />
      </IconButton>,
    );
    expect(button().style.inlineSize).toContain("--control-size-xs");
  });
});

describe("tap target", () => {
  it("asks for the comfortable target by default", () => {
    renderWithProviders(
      <IconButton aria-label="A">
        <BubbleIcon />
      </IconButton>,
    );
    expect(button()).toHaveAttribute("data-rcl-tap-target", "comfortable");
  });

  it("opts out only when the caller says so", () => {
    renderWithProviders(
      <IconButton aria-label="A" denseTapTarget>
        <BubbleIcon />
      </IconButton>,
    );
    expect(button()).toHaveAttribute("data-rcl-tap-target", "dense");
  });
});

describe("pass-through", () => {
  it("forwards the ref to the native button", () => {
    const ref = createRef<HTMLButtonElement>();
    renderWithProviders(
      <IconButton ref={ref} aria-label="A">
        <BubbleIcon />
      </IconButton>,
    );
    expect(ref.current).toBe(button());
  });

  it("forwards arbitrary props and data attributes", () => {
    renderWithProviders(
      <IconButton aria-label="A" data-active="true" tabIndex={-1}>
        <BubbleIcon />
      </IconButton>,
    );
    expect(button()).toHaveAttribute("data-active", "true");
    expect(button()).toHaveAttribute("tabindex", "-1");
  });

  it("fires onClick", () => {
    const onClick = vi.fn();
    renderWithProviders(
      <IconButton aria-label="A" onClick={onClick}>
        <BubbleIcon />
      </IconButton>,
    );
    fireEvent.click(button());
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("defaults to type=button so it never submits a surrounding form", () => {
    renderWithProviders(
      <IconButton aria-label="A">
        <BubbleIcon />
      </IconButton>,
    );
    expect(button()).toHaveAttribute("type", "button");
  });
});

/**
 * The swap behaviour is driven by `requestAnimationFrame`, so these drive the
 * clock explicitly rather than waiting on wall time.
 */
describe("icon swapping", () => {
  let now = 0;
  let frames: FrameRequestCallback[] = [];

  beforeEach(() => {
    now = 0;
    frames = [];
    vi.spyOn(performance, "now").mockImplementation(() => now);
    vi.stubGlobal("requestAnimationFrame", (cb: FrameRequestCallback) => {
      frames.push(cb);
      return frames.length;
    });
    vi.stubGlobal("cancelAnimationFrame", () => {});
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  const advance = (ms: number) => {
    now += ms;
    const pending = frames;
    frames = [];
    act(() => {
      for (const frame of pending) frame(now);
    });
  };

  function Toggle({ morph }: { morph?: "auto" | "morph" | "crossfade" | "none" }) {
    const [on, setOn] = useState(false);
    return (
      <IconButton
        aria-label="Toggle view"
        morph={morph}
        onClick={() => {
          setOn((v) => !v);
        }}
      >
        {on ? <TerminalIcon /> : <BubbleIcon />}
      </IconButton>
    );
  }

  /**
   * Reported from a frame-by-frame screen recording: pressing the toggle showed
   * the *incoming* icon for a single frame, snapped back to the outgoing one,
   * and only then morphed forward.
   *
   * React commits new children and lets the browser paint before *passive*
   * effects run, so starting the transition in `useEffect` leaves exactly one
   * painted frame with the incoming icon unhidden and no morph frame over it.
   * A layout effect runs synchronously after mutation, and the re-render it
   * schedules is flushed before paint, so that window does not exist.
   *
   * This is pinned at the source rather than behaviourally, and deliberately:
   * jsdom does not paint, and the distinction between the two effect phases is
   * not observable from a sibling probe — a sibling does not re-render when the
   * hook sets state, so its effects never run again to see it. The assertion
   * below is therefore a contract on the implementation, which is the thing
   * that actually regressed. The surrounding tests cover what the transition
   * does; this one covers *when* it starts.
   */
  it("starts the transition in the layout phase, before the browser can paint", () => {
    // Vitest serves modules over an http URL, so resolve from the repo path.
    const hookSource = readFileSync(
      resolve(__dirname, "@vrooli/react-component-library/useIconMorph/1.ts"),
      "utf8",
    );
    // The transition and the measurement that feeds it both run before paint.
    expect(hookSource).toContain("const useBeforePaint =");
    expect(hookSource).toContain("useLayoutEffect");
    expect(hookSource.match(/useBeforePaint\(\(\) => \{/g) ?? []).toHaveLength(2);
    // Nothing but the unmount cleanup may remain a passive effect.
    const passive = hookSource.match(/useEffect\(/g) ?? [];
    expect(passive).toHaveLength(1);
    expect(hookSource).toContain("useEffect(() => stop, [stop]);");
  });

  it("is idle until the icon actually changes", () => {
    renderWithProviders(<Toggle />);
    expect(glyph()).toHaveAttribute("data-rcl-technique", "idle");
  });

  it("morphs a compatible pair without the call site asking", () => {
    // BubbleIcon -> TerminalIcon is the web-console view toggle. It scores 0.862.
    renderWithProviders(<Toggle />);
    act(() => {
      fireEvent.click(button());
    });
    expect(glyph()).toHaveAttribute("data-rcl-technique", "morph");
    // A morph paints synthesised paths over the live icon.
    expect(document.querySelector("[data-rcl-morphing-icon-frame]")).not.toBeNull();
  });

  it("crossfades a pair that would morph badly", () => {
    function SunToggle() {
      const [on, setOn] = useState(false);
      return (
        <IconButton
          aria-label="Theme"
          onClick={() => {
            setOn((v) => !v);
          }}
        >
          {on ? <BubbleIcon /> : <SunIcon />}
        </IconButton>
      );
    }
    renderWithProviders(<SunToggle />);
    act(() => {
      fireEvent.click(button());
    });
    expect(glyph()).toHaveAttribute("data-rcl-technique", "crossfade");
    expect(document.querySelector("[data-rcl-morphing-icon-previous]")).not.toBeNull();
  });

  it("returns to idle once the transition completes", () => {
    renderWithProviders(<Toggle />);
    act(() => {
      fireEvent.click(button());
    });
    expect(glyph()).toHaveAttribute("data-rcl-technique", "morph");
    advance(400);
    expect(glyph()).toHaveAttribute("data-rcl-technique", "idle");
    expect(document.querySelector("[data-rcl-morphing-icon-frame]")).toBeNull();
  });

  it("swaps instantly when asked to", () => {
    renderWithProviders(<Toggle morph="none" />);
    act(() => {
      fireEvent.click(button());
    });
    expect(glyph()).toHaveAttribute("data-rcl-technique", "idle");
  });

  it("forces a morph on a pair it would otherwise reject", () => {
    function SunToggle() {
      const [on, setOn] = useState(false);
      return (
        <IconButton
          aria-label="Theme"
          morph="morph"
          onClick={() => {
            setOn((v) => !v);
          }}
        >
          {on ? <BubbleIcon /> : <SunIcon />}
        </IconButton>
      );
    }
    renderWithProviders(<SunToggle />);
    act(() => {
      fireEvent.click(button());
    });
    expect(glyph()).toHaveAttribute("data-rcl-technique", "morph");
  });

  it("never morphs when told to crossfade", () => {
    renderWithProviders(<Toggle morph="crossfade" />);
    act(() => {
      fireEvent.click(button());
    });
    expect(glyph()).toHaveAttribute("data-rcl-technique", "crossfade");
  });

  it("keeps the live icon mounted through a morph so the next swap can measure it", () => {
    renderWithProviders(<Toggle />);
    act(() => {
      fireEvent.click(button());
    });
    const live = document.querySelector("[data-rcl-morphing-icon-current]");
    expect(live).not.toBeNull();
    expect(live!.querySelector("svg")).not.toBeNull();
    // Hidden, not unmounted — unmounting would destroy the geometry source.
    expect(live).toHaveAttribute("data-rcl-hidden", "true");
  });

  /**
   * Driven by state rather than by the harness's `rerender`, which tears the
   * subtree down and remounts it — that would reset the hook's record of what
   * was previously showing and prove nothing about update behaviour.
   */
  it("treats an explicit iconKey as the identity", () => {
    function Keyed() {
      const [variant, setVariant] = useState("one");
      return (
        <IconButton
          aria-label="A"
          iconKey={variant}
          onClick={() => {
            setVariant("two");
          }}
        >
          <BubbleIcon />
        </IconButton>
      );
    }
    renderWithProviders(<Keyed />);
    expect(glyph()).toHaveAttribute("data-rcl-technique", "idle");
    act(() => {
      fireEvent.click(button());
    });
    // Same component, different declared identity, so it still transitions.
    expect(glyph()).not.toHaveAttribute("data-rcl-technique", "idle");
  });

  /**
   * The defect this prop exists for. web-console renders one logical view
   * toggle from two different parents — floating over the terminal in one mode,
   * inline in the messages toolbar in the other — so switching views destroys
   * the instance and builds a new one. Without a stable identity the new
   * instance believes it has always shown its current icon, and the swap that
   * is the entire point of the control is skipped every time.
   */
  it("still animates when the control remounts under a different parent", () => {
    function MovingToggle() {
      const [terminal, setTerminal] = useState(false);
      const control = (
        <IconButton
          aria-label="Toggle view"
          swapIdentity="test-view-toggle"
          onClick={() => {
            setTerminal((v) => !v);
          }}
        >
          {terminal ? <TerminalIcon /> : <BubbleIcon />}
        </IconButton>
      );
      // Two mutually exclusive parents: React cannot preserve the instance.
      return terminal ? <section>{control}</section> : <article>{control}</article>;
    }
    renderWithProviders(<MovingToggle />);
    expect(glyph()).toHaveAttribute("data-rcl-technique", "idle");
    act(() => {
      fireEvent.click(button());
    });
    expect(glyph()).toHaveAttribute("data-rcl-technique", "morph");
  });

  it("does not animate a control that appears for the first time", () => {
    // A fresh identity must not animate: there is no previous icon to come from.
    function LateArrival() {
      const [shown, setShown] = useState(false);
      return (
        <>
          <button
            type="button"
            onClick={() => {
              setShown(true);
            }}
          >
            show
          </button>
          {shown ? (
            <IconButton aria-label="New" swapIdentity="test-late-arrival">
              <BubbleIcon />
            </IconButton>
          ) : null}
        </>
      );
    }
    renderWithProviders(<LateArrival />);
    act(() => {
      fireEvent.click(screen.getByText("show"));
    });
    expect(glyph()).toHaveAttribute("data-rcl-technique", "idle");
  });

  it("keeps remount-skipping behaviour when no identity is given", () => {
    // Without an identity a remount is a genuinely new control and must not
    // animate — otherwise every mount would flash a transition.
    function MovingToggle() {
      const [terminal, setTerminal] = useState(false);
      const control = (
        <IconButton
          aria-label="Toggle view"
          onClick={() => {
            setTerminal((v) => !v);
          }}
        >
          {terminal ? <TerminalIcon /> : <BubbleIcon />}
        </IconButton>
      );
      return terminal ? <section>{control}</section> : <article>{control}</article>;
    }
    renderWithProviders(<MovingToggle />);
    act(() => {
      fireEvent.click(button());
    });
    expect(glyph()).toHaveAttribute("data-rcl-technique", "idle");
  });

  it("does not transition when an unrelated prop changes", () => {
    function Relabelled() {
      const [label, setLabel] = useState("A");
      return (
        <IconButton
          aria-label={label}
          onClick={() => {
            setLabel("B");
          }}
        >
          <BubbleIcon />
        </IconButton>
      );
    }
    renderWithProviders(<Relabelled />);
    act(() => {
      fireEvent.click(button());
    });
    expect(button()).toHaveAttribute("aria-label", "B");
    expect(glyph()).toHaveAttribute("data-rcl-technique", "idle");
  });
});
