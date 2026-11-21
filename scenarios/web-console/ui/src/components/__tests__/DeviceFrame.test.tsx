import { renderWithProviders as render } from "../../test-utils";
import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { setLocale } from "../../i18n";
import { strings } from "../../consts/strings";
import { DeviceFrame } from "../terminal/DeviceFrame";
import { DEVICE_ARCHETYPES, type DeviceArchetype } from "../../lib/deviceArchetype";
import { FRAME_ASPECT, type ChromeTier } from "../../lib/followerViewport";
import { FOLLOWER_ENCLOSURE_Z } from "../../hooks/terminal/useFollowerViewportLayout";

const PANE = { width: 900, height: 700 };
const PANE_THEME = { background: "#0f172a", foreground: "#e2e8f0" };

/**
 * Build a rect whose proportions match the archetype under test. The previous
 * suite passed one 300×200 landscape rect and called it a phone for every case.
 */
function rectFor(archetype: DeviceArchetype) {
  const aspect = FRAME_ASPECT[archetype];
  let width = Math.min(PANE.width, PANE.height * aspect);
  const height = width / aspect;
  if (height > PANE.height) width = PANE.height * aspect;
  return { x: 0, y: 0, width, height: width / aspect, fontSize: 12, scale: 1 };
}

function renderFrame(overrides: Partial<Parameters<typeof DeviceFrame>[0]> = {}) {
  const archetype = overrides.archetype ?? "phone";
  const onTakeOver = overrides.onTakeOver ?? vi.fn();
  render(<DeviceFrame
    archetype={archetype}
    chromeTier={overrides.chromeTier ?? "full"}
    rect={overrides.rect ?? rectFor(archetype)}
    keyboardShare={overrides.keyboardShare ?? 0}
    captionOffset={overrides.captionOffset ?? 8}
    leaderDevice={overrides.leaderDevice ?? ""}
    gridCols={overrides.gridCols ?? 80}
    gridRows={overrides.gridRows ?? 24}
    kbOpen={overrides.kbOpen ?? false}
    paneTheme={overrides.paneTheme ?? PANE_THEME}
    onTakeOver={onTakeOver}
  />);
  return { onTakeOver };
}

describe("DeviceFrame", () => {
  it.each(["full", "hairline", "strip"] as const)("renders a takeover control at %s", (chromeTier: ChromeTier) => {
    const { onTakeOver } = renderFrame({ chromeTier });
    fireEvent.click(screen.getByRole("button"));
    expect(onTakeOver).toHaveBeenCalledOnce();
  });

  it.each(DEVICE_ARCHETYPES)("draws a distinct silhouette for %s", (archetype) => {
    const { onTakeOver } = renderFrame({ archetype });
    const frame = screen.getByTestId("device-frame-full");
    expect(frame.dataset.archetype).toBe(archetype);
    // The viewBox aspect must equal the frame aspect, or the silhouette is
    // stretched — the defect the unit-square viewBox used to guarantee.
    const svg = frame.querySelector("svg");
    const [, , boxWidth, boxHeight] = (svg?.getAttribute("viewBox") ?? "").split(" ").map(Number);
    expect(Number(boxWidth) / Number(boxHeight)).toBeCloseTo(FRAME_ASPECT[archetype], 5);
    expect(svg?.getAttribute("preserveAspectRatio")).toBeNull();
    fireEvent.click(screen.getByRole("button"));
    expect(onTakeOver).toHaveBeenCalledOnce();
  });

  it("gives every archetype its own geometry rather than sharing one drawing", () => {
    const drawn = new Set<string>();
    for (const archetype of DEVICE_ARCHETYPES) {
      const { unmount } = renderWithFrame(archetype);
      drawn.add(screen.getByTestId("device-frame-full").querySelector("svg")?.innerHTML ?? "");
      unmount();
    }
    expect(drawn.size).toBe(DEVICE_ARCHETYPES.length);
  });

  it("paints the opaque enclosure below the terminal and the caption above it", () => {
    renderFrame();
    const enclosure = screen.getByTestId("device-frame-full");
    const caption = screen.getByTestId("device-caption-full");
    // The enclosure carries no text: it is decoration behind the session.
    expect(enclosure.textContent).toBe("");
    expect(Number(enclosure.style.zIndex)).toBe(FOLLOWER_ENCLOSURE_Z);
    expect(caption.className).toContain("z-wc-chrome-raised");
  });

  it("places the caption below the stand it was given clearance for", () => {
    // A monitor panel 400px tall carries a stand that scales with it, so the
    // caption offset is a measured distance, never a fixed class.
    renderFrame({ archetype: "monitor", captionOffset: 84 });
    const caption = screen.getByTestId("device-caption-full");
    const chip = caption.firstElementChild as HTMLElement;
    const frameHeight = rectFor("monitor").height;
    expect(chip.style.top).toBe(`${String(frameHeight + 84)}px`);
  });

  it("pins the caption to the top of the pane in the strip tier", () => {
    renderFrame({ chromeTier: "strip", captionOffset: 84 });
    const chip = screen.getByTestId("device-caption-strip").firstElementChild as HTMLElement;
    expect(chip.style.top).toBe("");
    expect(chip.className).toContain("top-0");
  });

  it("announces who is driving, since the silhouette is decorative", () => {
    renderFrame({ leaderDevice: "iPhone", gridCols: 46, gridRows: 26 });
    expect(screen.getByRole("status")).toHaveTextContent(strings.deviceFrame.following);
    expect(screen.getByTestId("device-frame-full").querySelector("svg")).toHaveAttribute("aria-hidden", "true");
  });

  it("reports the leader's keyboard instead of letting the frame deform", () => {
    renderFrame({ leaderDevice: "iPhone", gridCols: 46, gridRows: 13, kbOpen: true, keyboardShare: 0.3 });
    const frame = screen.getByTestId("device-frame-full");
    expect(frame.dataset.keyboard).toBe("open");
    expect(screen.getByText(strings.deviceFrame.keyboardOpen)).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent(strings.deviceFrame.followingKeyboard);
  });

  it("draws no key plate when the leader reports no keyboard", () => {
    renderFrame({ kbOpen: false, keyboardShare: 0.3 });
    const frame = screen.getByTestId("device-frame-full");
    expect(frame.dataset.keyboard).toBe("closed");
    expect(screen.queryByText(strings.deviceFrame.keyboardOpen)).toBeNull();
  });
});

// These assert on real rendered copy rather than key paths, so they opt out of
// the process-wide cimode default. The caption is translated and interpolated,
// which is exactly what a key-path assertion cannot check.
describe("DeviceFrame copy", () => {
  beforeEach(async () => { await setLocale("en"); });

  it("names the leader's device and its grid", () => {
    renderFrame({ leaderDevice: "iPhone", gridCols: 46, gridRows: 26 });
    expect(screen.getByText("iPhone · 46×26")).toBeInTheDocument();
  });

  it("never leaks the raw archetype enum when the leader has no label", () => {
    renderFrame({ archetype: "ultrawide", leaderDevice: "" });
    const caption = screen.getByTestId("device-caption-full");
    // The enum reaches the DOM as a data attribute for tests and styling, but
    // must never appear in copy a reader sees.
    expect(caption.textContent).not.toMatch(/\bultrawide\b/);
    expect(caption.textContent).toContain("Ultrawide display");
  });

  it("explains a shrunken grid rather than leaving it unexplained", () => {
    renderFrame({ leaderDevice: "iPhone", gridCols: 46, gridRows: 13, kbOpen: true, keyboardShare: 0.3 });
    expect(screen.getByRole("status")).toHaveTextContent("Following iPhone, 46 by 13 characters. Its keyboard is open.");
    expect(screen.getByText("keyboard open")).toBeInTheDocument();
  });
});

function renderWithFrame(archetype: DeviceArchetype) {
  return render(<DeviceFrame
    archetype={archetype}
    chromeTier="full"
    rect={rectFor(archetype)}
    keyboardShare={0}
    captionOffset={8}
    leaderDevice="Device"
    gridCols={80}
    gridRows={24}
    kbOpen={false}
    paneTheme={PANE_THEME}
    onTakeOver={vi.fn()}
  />);
}
