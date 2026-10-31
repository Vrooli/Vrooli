/**
 * iOS raises the keyboard only for a field focused inside the tap itself. A
 * field that mounts a commit later (a sheet's portal waits one effect) is
 * focused too late and the keyboard stays down. Focusing this stand-in inside
 * the tap raises the keyboard; iOS keeps it up when focus then moves to the
 * real field, and the stand-in removes itself on blur.
 */
export function holdKeyboardForNextField(): void {
  const standIn = document.createElement("input");
  standIn.setAttribute("aria-hidden", "true");
  standIn.tabIndex = -1;
  // 16px keeps iOS from zooming the page; opacity 0 keeps it unseen.
  standIn.style.cssText = "position:fixed;top:0;left:0;width:1px;height:1px;opacity:0;font-size:16px;pointer-events:none;";
  standIn.addEventListener("blur", () => { standIn.remove(); }, { once: true });
  document.body.appendChild(standIn);
  standIn.focus({ preventScroll: true });
  // The field never took focus: lower the keyboard rather than leave it up.
  window.setTimeout(() => { if (document.activeElement === standIn) standIn.blur(); }, 1000);
}
