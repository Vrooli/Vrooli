import DOMPurify from "dompurify";

// Word contributes content, never app styles, controls, remote images or code.
export function sanitizeDocument(html: string, tableLabel: string): string {
  const fragment = DOMPurify.sanitize(html, {
    ALLOWED_TAGS: ["p", "h1", "h2", "h3", "h4", "h5", "h6", "strong", "em", "u", "s", "sup", "sub", "br", "hr", "ul", "ol", "li", "blockquote", "pre", "code", "table", "thead", "tbody", "tfoot", "tr", "th", "td", "a", "img", "span"],
    ALLOWED_ATTR: ["href", "id", "src", "alt", "colspan", "rowspan", "start"],
    ALLOW_DATA_ATTR: false, ALLOW_ARIA_ATTR: false,
    RETURN_DOM_FRAGMENT: true,
  });
  for (const image of fragment.querySelectorAll("img")) {
    if (!/^data:image\/(png|jpeg|gif|webp);base64,[a-z0-9+/=]+$/i.test(image.getAttribute("src") ?? "")) { image.remove(); continue; }
    image.setAttribute("loading", "lazy"); image.setAttribute("decoding", "async");
    if (!image.hasAttribute("alt")) image.setAttribute("alt", "");
  }
  for (const link of fragment.querySelectorAll("a")) {
    const href = link.getAttribute("href") ?? "";
    if (!/^(https?:\/\/|mailto:|#)/i.test(href)) link.removeAttribute("href");
    else if (!href.startsWith("#")) { link.setAttribute("target", "_blank"); link.setAttribute("rel", "noopener noreferrer"); }
  }
  // A table scrolls locally; prose always reflows within the reading measure.
  for (const table of fragment.querySelectorAll("table")) {
    const wrapper = document.createElement("div"); wrapper.className = "wc-document-table";
    wrapper.tabIndex = 0; wrapper.setAttribute("role", "region"); wrapper.setAttribute("aria-label", tableLabel);
    table.replaceWith(wrapper); wrapper.append(table);
  }
  const container = document.createElement("div"); container.append(fragment);
  return container.innerHTML;
}
