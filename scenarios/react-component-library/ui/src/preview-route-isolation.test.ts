// @vitest-environment jsdom
import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import { createRequire } from "node:module";
const { JSDOM } = createRequire(import.meta.url)("jsdom");

const source = readFileSync("../api/handlers/preview/assets/harness.js", "utf8");
const isolation = source.slice(
  source.indexOf("(() => {", source.indexOf("// Route isolation:")),
  source.indexOf("// Inspector (req"),
);

function preview(base?: string) {
  const dom = new JSDOM(`${base ? `<base href="${base}">` : ""}<a>Skip</a>`, {
    url: "http://preview.test/preview/AppShell/harness.html?version=draft&story=skip",
    runScripts: "outside-only",
  });
  dom.window.eval(isolation);
  return dom;
}

describe("preview route isolation", () => {
  it("isolates an unhandled fragment whose base URI points outside the harness", () => {
    const dom = preview("/");
    try {
      const link = dom.window.document.querySelector("a");
      link.setAttribute("href", "#main");
      expect(link.href).toBe("http://preview.test/#main");
      let received = false;
      link.addEventListener("click", (event: Event) => {
        received = true;
        expect(event.defaultPrevented).toBe(false);
      });
      const event = new dom.window.MouseEvent("click", { bubbles: true, cancelable: true });
      link.dispatchEvent(event);
      expect(received).toBe(true);
      expect(event.defaultPrevented).toBe(true);
      expect(dom.window.location.pathname).toBe("/preview/AppShell/harness.html");
      expect(dom.window.location.hash).toBe("#/#main");
    } finally {
      dom.window.close();
    }
  });
  it.each([undefined, "/"])(
    "delivers document fragments to component handlers with base %s",
    (base) => {
      const dom = preview(base);
      try {
        const link = dom.window.document.querySelector("a");
        for (const href of [
          "#main",
          `${dom.window.location.pathname}${dom.window.location.search}#main`,
        ]) {
          link.setAttribute("href", href);
          let received = false;
          link.addEventListener(
            "click",
            (event: Event) => {
              received = true;
              expect(event.defaultPrevented).toBe(false);
              event.preventDefault();
            },
            { once: true },
          );
          const event = new dom.window.MouseEvent("click", { bubbles: true, cancelable: true });
          link.dispatchEvent(event);
          expect(received).toBe(true);
          expect(event.defaultPrevented).toBe(true);
          expect(dom.window.location.hash).toBe("");
        }
      } finally {
        dom.window.close();
      }
    },
  );
  it.each([
    "#/settings",
    "/settings#account",
    "/settings?panel=setup#account",
    "/preview/AppShell/harness.html?other=1#main",
  ])("isolates virtual/cross-page navigation %s", (href) => {
    const dom = preview("/");
    try {
      const link = dom.window.document.querySelector("a");
      link.setAttribute("href", href);
      let received = false;
      link.addEventListener("click", () => {
        received = true;
      });
      const event = new dom.window.MouseEvent("click", { bubbles: true, cancelable: true });
      link.dispatchEvent(event);
      expect(received).toBe(false);
      expect(event.defaultPrevented).toBe(true);
      expect(dom.window.location.pathname).toBe("/preview/AppShell/harness.html");
      expect(dom.window.location.hash).not.toBe("");
    } finally {
      dom.window.close();
    }
  });
  it("preserves modified-click and external destination defaults", () => {
    for (const external of [false, true]) {
      const dom = preview();
      try {
        const link = dom.window.document.querySelector("a");
        link.href = external ? "https://elsewhere.test/settings" : "/settings";
        let received = false;
        link.addEventListener("click", (e: Event) => {
          received = true;
          expect(e.defaultPrevented).toBe(false);
          e.preventDefault();
        });
        link.dispatchEvent(
          new dom.window.MouseEvent("click", {
            bubbles: true,
            cancelable: true,
            ctrlKey: !external,
          }),
        );
        expect(received).toBe(true);
      } finally {
        dom.window.close();
      }
    }
  });
});
