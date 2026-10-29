import { waitFor } from "@testing-library/react";
import type { UserEvent } from "@testing-library/user-event";

/**
 * The shared error presentation renders an alert that names what failed and
 * keeps the raw transport text as detail (a tooltip in compact mode, a
 * disclosure in full mode). Assert on that contract, not on raw strings.
 */
export function getErrorNotice(subject: string | RegExp, detail?: string): HTMLElement {
  const notices = Array.from(document.body.querySelectorAll<HTMLElement>("[data-error-kind]"));
  const matches = (text: string | null | undefined) =>
    typeof subject === "string" ? (text ?? "").toLowerCase().includes(subject.toLowerCase()) : subject.test(text ?? "");
  const notice = notices.find((element) => matches(element.textContent));
  if (!notice) throw new Error(`No error notice for ${String(subject)}; alerts: ${notices.map((n) => n.textContent).join(" | ")}`);
  if (detail !== undefined) {
    const carried = (notice.querySelector("[title]")?.getAttribute("title") ?? "") + (notice.textContent ?? "");
    if (!carried.includes(detail)) throw new Error(`Error notice for ${String(subject)} does not carry detail "${detail}"`);
  }
  return notice;
}

/** Confirms (or cancels) the shared destructive-action dialog. */
export async function answerConfirmDialog(user: UserEvent, answer: "confirm" | "cancel" = "confirm"): Promise<HTMLElement> {
  // Query by attribute: role queries recompute styles across the library's
  // injected stylesheets and make overlay tests needlessly slow.
  const dialog = await waitFor(() => {
    const found = document.querySelector<HTMLElement>('[role="alertdialog"]');
    if (!found) throw new Error("No confirmation dialog is open");
    return found;
  });
  const button = dialog.querySelector<HTMLButtonElement>(answer === "cancel" ? "[data-rcl-alert-dialog-cancel]" : "[data-rcl-alert-dialog-confirm]");
  if (!button) throw new Error(`No ${answer} button in confirmation dialog`);
  await user.click(button);
  return dialog;
}

/** Resolves once the confirmation dialog has finished closing. */
export async function waitForConfirmDialogClosed(): Promise<void> {
  await waitFor(() => {
    if (document.querySelector('[role="alertdialog"]')) throw new Error("Confirmation dialog is still open");
  });
}
