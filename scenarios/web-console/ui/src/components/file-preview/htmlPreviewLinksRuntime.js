// @ts-check
/** Keep fragments local; hand user-activated file links to the native viewer.
 * @param {Window} previewWindow
 * @returns {() => void}
 */
export function installHtmlPreviewLinks(previewWindow) {
  const document = previewWindow.document;
  /** @param {MouseEvent} event */
  const click = (event) => {
    if (event.defaultPrevented || event.button !== 0) return;
    const target = event.target;
    if (!(target instanceof previewWindow.Element)) return;
    const anchor = target.closest("a[href]");
    if (!anchor) return;
    const href = (anchor.getAttribute("href") || "").trim();
    if (!href) return;
    if (href.startsWith("#")) {
      // <base href=runtime/> otherwise turns #section into a document GET.
      event.preventDefault();
      let id = href.slice(1);
      try { id = decodeURIComponent(id); } catch { /* Preserve literal IDs. */ }
      if (!id) previewWindow.scrollTo(0, 0);
      else document.getElementById(id)?.scrollIntoView({ block: "start" });
      return;
    }
    if (href.startsWith("//") || (/^[a-z][a-z0-9+.-]*:/i.test(href) && !/^file:/i.test(href))) return;
    event.preventDefault();
    // Authored scripts must not open host files by synthesizing clicks.
    if (!event.isTrusted) return;
    previewWindow.parent.postMessage({ type: "wc-html-file-link", href }, "*");
  };
  document.addEventListener("click", click);
  return () => { document.removeEventListener("click", click); };
}
