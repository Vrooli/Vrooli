import { ResponsiveDialog } from "./ResponsiveDialog";

const paragraph = "A bounded decision keeps the surface underneath it visible and intact.";

/** The default rendered anatomy: the sheet presentation with its grabber. */
export function Sheet() {
  return (
    <ResponsiveDialog open title="Rename pane" closeLabel="Close dialog">
      <p>{paragraph}</p>
    </ResponsiveDialog>
  );
}

/** The large centered card at and above the medium breakpoint. */
export function LargeCard() {
  return (
    <ResponsiveDialog open size="lg" title="Rename pane" closeLabel="Close dialog">
      <p>{paragraph}</p>
    </ResponsiveDialog>
  );
}

/** A full-bleed band above the scroll region, with the caller owning the gutters. */
export function FlushContentWithSubheader() {
  return (
    <ResponsiveDialog
      open
      title="Rename pane"
      closeLabel="Close dialog"
      subheader={
        <div role="tablist" aria-label="Scope">
          <button type="button" role="tab" aria-selected="true">
            This pane
          </button>
          <button type="button" role="tab" aria-selected="false">
            All panes
          </button>
        </div>
      }
    >
      <p>{paragraph}</p>
    </ResponsiveDialog>
  );
}

/** A footer holds the committing action while the body scrolls. */
export function WithFooter() {
  return (
    <ResponsiveDialog
      open
      title="Rename pane"
      closeLabel="Close dialog"
      footer={<button type="button">Rename</button>}
    >
      <div style={{ blockSize: 900 }}>{paragraph}</div>
    </ResponsiveDialog>
  );
}

/** Uncontrolled input state survives the presentation change at the breakpoint. */
export function PreservedInput() {
  return (
    <ResponsiveDialog avoidKeyboard open size="sm" title="Rename pane" closeLabel="Close dialog">
      <input aria-label="Pane name" defaultValue="Preserved value" />
    </ResponsiveDialog>
  );
}

/**
 * `comfortable` opts back in to a uniform gutter for the plain-prose case,
 * where the content carries no insets of its own. It is the non-default
 * branch: leaving the prop off hands the full content box to the caller.
 */
export function ComfortableContent() {
  return (
    <ResponsiveDialog open contentPadding="comfortable" title="Share" closeLabel="Close sheet">
      <p>{paragraph}</p>
    </ResponsiveDialog>
  );
}
