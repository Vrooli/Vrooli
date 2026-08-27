import { BottomSheet } from "./BottomSheet";

const paragraph = "A sheet holds a bounded task without leaving the surface underneath it.";

/** The default rendered anatomy: grabber, header, and a padded scroll region. */
export function Sheet() {
  return (
    <BottomSheet open title="Share" closeLabel="Close sheet">
      <p>{paragraph}</p>
    </BottomSheet>
  );
}

/** A close button may be added beside the header actions where a gesture is not enough. */
export function WithCloseButton() {
  return (
    <BottomSheet open showCloseButton title="Share" closeLabel="Close sheet">
      <p>{paragraph}</p>
    </BottomSheet>
  );
}

/** A full-bleed band above the scroll region, with the caller owning the gutters. */
export function FlushContentWithSubheader() {
  return (
    <BottomSheet
      open
      contentPadding="none"
      title="Share"
      closeLabel="Close sheet"
      subheader={
        <div role="tablist" aria-label="Targets">
          <button type="button" role="tab" aria-selected="true">
            People
          </button>
          <button type="button" role="tab" aria-selected="false">
            Links
          </button>
        </div>
      }
    >
      <p>{paragraph}</p>
    </BottomSheet>
  );
}

/** A footer holds the committing action while the body scrolls. */
export function WithFooter() {
  return (
    <BottomSheet
      open
      title="Share"
      closeLabel="Close sheet"
      footer={<button type="button">Send</button>}
    >
      <div style={{ blockSize: 900 }}>{paragraph}</div>
    </BottomSheet>
  );
}

/** The sheet lifts above the software keyboard when the host reports one. */
export function AvoidingKeyboard() {
  return (
    <BottomSheet avoidKeyboard open title="Share" closeLabel="Close sheet">
      <input aria-label="Recipient" defaultValue="" />
    </BottomSheet>
  );
}
