import { useRef } from "react";
import { FullPageDrawer } from "./FullPageDrawer";

const paragraph =
  "A drawer keeps the route and the source context while a page-scale task takes the screen.";

/**
 * The default rendered anatomy: the small-viewport sheet, flush with the
 * bottom edge, dismissed by its grabber.
 */
export function MobileSheet() {
  return (
    <FullPageDrawer
      open
      dismissAffordance="grabber"
      title="Workspace"
      closeLabel="Close drawer"
      headerExtra={<small>Layout and workspace behavior.</small>}
    >
      <p>{paragraph}</p>
    </FullPageDrawer>
  );
}

/** At and above the medium breakpoint the drawer is an inset card with a close button. */
export function DesktopCard() {
  return (
    <FullPageDrawer
      open
      dismissAffordance="close"
      title="Workspace"
      closeLabel="Close drawer"
      headerActions={<button type="button">Save</button>}
    >
      <p>{paragraph}</p>
    </FullPageDrawer>
  );
}

/**
 * A full-bleed band above the scroll region, with the caller owning the
 * content gutters. The band does not scroll, so no second scroller is nested
 * inside the drawer to make it.
 */
export function FlushContentWithSubheader() {
  return (
    <FullPageDrawer
      open
      dismissAffordance="grabber"
      title="Workspace"
      closeLabel="Close drawer"
      subheader={
        <div role="tablist" aria-label="Sections">
          <button type="button" role="tab" aria-selected="true">
            Sessions
          </button>
          <button type="button" role="tab" aria-selected="false">
            Input
          </button>
        </div>
      }
    >
      <p>{paragraph}</p>
    </FullPageDrawer>
  );
}

/** With a footer, the footer clears the bottom safe area and the scroll region does not. */
export function WithFooter() {
  return (
    <FullPageDrawer
      open
      dismissAffordance="grabber"
      title="Workspace"
      closeLabel="Close drawer"
      footer={<button type="button">Continue</button>}
    >
      <p>{paragraph}</p>
    </FullPageDrawer>
  );
}

/** Overflowing content scrolls inside the drawer while the chrome stays put. */
export function LongBody() {
  return (
    <FullPageDrawer
      open
      dismissAffordance="grabber"
      title="Workspace"
      closeLabel="Close drawer"
    >
      <div style={{ blockSize: 1600 }}>{paragraph}</div>
    </FullPageDrawer>
  );
}

/** Initial focus may be directed at a specific control rather than the first one. */
export function DirectedInitialFocus() {
  const target = useRef<HTMLInputElement | null>(null);
  return (
    <FullPageDrawer
      open
      dismissAffordance="grabber"
      title="Workspace"
      closeLabel="Close drawer"
      initialFocusRef={target}
    >
      <input
        ref={target}
        aria-label="Draft value"
        defaultValue="Preserved value"
      />
    </FullPageDrawer>
  );
}

/**
 * `comfortable` opts back in to a uniform gutter for the plain-prose case,
 * where the content carries no insets of its own. It is the non-default
 * branch: leaving the prop off hands the full content box to the caller.
 */
export function ComfortableContent() {
  return (
    <FullPageDrawer
      open
      contentPadding="comfortable"
      title="Share"
      closeLabel="Close sheet"
    >
      <p>{paragraph}</p>
    </FullPageDrawer>
  );
}
