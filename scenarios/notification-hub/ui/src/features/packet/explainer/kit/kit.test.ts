import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import kitSource from "./kit.js?raw";

// Vitest stubs CSS imports, so the stylesheet is read from disk.
const kitCss = readFileSync(join(process.cwd(), "src/features/packet/explainer/kit/kit.css"), "utf8");

// kit.js is a classic script the Hub injects into the sealed page, so the tests
// run its source the same way: as the text of a script element.
function runScript(source: string): void {
  const script = document.createElement("script");
  script.textContent = source;
  document.head.appendChild(script);
  script.remove();
}

function runKit(): void {
  runScript(kitSource);
}

// What the page itself would see on `window`, read through a page script.
function pageHas(name: string): boolean {
  document.body.removeAttribute("data-probe");
  runScript(`document.body.setAttribute("data-probe", String(typeof window.${name} !== "undefined"))`);
  return document.body.getAttribute("data-probe") === "true";
}

function mount(html: string): HTMLElement {
  document.body.innerHTML = html;
  runKit();
  return document.body;
}

const COMPARE = `
  <div data-hub-compare data-before-label="Today" data-after-label="Proposed" data-position="30">
    <div id="before">Old layout</div>
    <div id="after">New layout</div>
  </div>`;

function pointer(type: string, clientX: number): Event {
  return new MouseEvent(type, { bubbles: true, clientX, button: 0 });
}

function stubWidth(el: Element, left: number, width: number): void {
  el.getBoundingClientRect = () => new DOMRect(left, 0, width, 100);
}

describe("explainer kit: compare", () => {
  beforeEach(() => {
    runScript("delete window.hubKit");
  });
  afterEach(() => {
    document.body.innerHTML = "";
  });

  it("builds an accessible slider from the first two children", () => {
    const body = mount(COMPARE);
    const root = body.querySelector<HTMLElement>("[data-hub-compare]");
    const range = root?.querySelector<HTMLInputElement>("input[type=range]");

    expect(root).toHaveAttribute("data-hub-ready");
    expect(root).toHaveClass("hub-compare");
    expect(body.querySelector("#before")).toHaveClass("hub-compare__before");
    expect(body.querySelector("#after")).toHaveClass("hub-compare__after");
    expect(range).toHaveAttribute("aria-label", "Today and Proposed");
    expect(range?.value).toBe("30");
    expect(range).toHaveAttribute("aria-valuetext", "70% Proposed");
    expect(root?.style.getPropertyValue("--hub-pos")).toBe("30%");
    expect(root?.textContent).toContain("Today");
    expect(root?.textContent).toContain("Proposed");
  });

  it("starts in the middle with default labels", () => {
    const body = mount(`<div data-hub-compare><p>a</p><p>b</p></div>`);
    const range = body.querySelector<HTMLInputElement>("input[type=range]");

    expect(range?.value).toBe("50");
    expect(range).toHaveAttribute("aria-label", "Before and After");
  });

  it("moves the divider with the keyboard range input", () => {
    const body = mount(COMPARE);
    const root = body.querySelector<HTMLElement>("[data-hub-compare]");
    const range = body.querySelector<HTMLInputElement>("input[type=range]");
    if (!root || !range) throw new Error("slider missing");

    range.value = "80";
    range.dispatchEvent(new Event("input", { bubbles: true }));

    expect(root.style.getPropertyValue("--hub-pos")).toBe("80%");
    expect(range).toHaveAttribute("aria-valuetext", "20% Proposed");
  });

  it("follows a pointer drag and stops on release", () => {
    const body = mount(COMPARE);
    const root = body.querySelector<HTMLElement>("[data-hub-compare]");
    const range = body.querySelector<HTMLInputElement>("input[type=range]");
    if (!root || !range) throw new Error("slider missing");
    stubWidth(root, 100, 200);

    root.dispatchEvent(pointer("pointerdown", 150));
    expect(range.value).toBe("25");
    root.dispatchEvent(pointer("pointermove", 250));
    expect(range.value).toBe("75");
    root.dispatchEvent(pointer("pointerup", 250));
    root.dispatchEvent(pointer("pointermove", 120));
    expect(range.value).toBe("75");
  });

  it("clamps a drag that leaves the block", () => {
    const body = mount(COMPARE);
    const root = body.querySelector<HTMLElement>("[data-hub-compare]");
    const range = body.querySelector<HTMLInputElement>("input[type=range]");
    if (!root || !range) throw new Error("slider missing");
    stubWidth(root, 0, 100);

    root.dispatchEvent(pointer("pointerdown", 40));
    root.dispatchEvent(pointer("pointermove", 900));
    expect(range.value).toBe("100");
    root.dispatchEvent(pointer("pointermove", -50));
    expect(range.value).toBe("0");
  });

  it("is idempotent: a second run adds no controls and no listeners", () => {
    const body = mount(COMPARE);
    runKit();
    runKit();
    const root = body.querySelector<HTMLElement>("[data-hub-compare]");
    if (!root) throw new Error("slider missing");

    expect(body.querySelectorAll("input[type=range]")).toHaveLength(1);
    expect(body.querySelectorAll(".hub-compare__handle")).toHaveLength(1);
    expect(body.querySelectorAll(".hub-compare__tag")).toHaveLength(2);

    stubWidth(root, 0, 100);
    root.dispatchEvent(pointer("pointerdown", 10));
    expect(body.querySelector<HTMLInputElement>("input[type=range]")?.value).toBe("10");
  });

  it("leaves a block with fewer than two children alone", () => {
    const body = mount(`<div data-hub-compare><p>only one</p></div>`);

    expect(body.querySelector("input")).toBeNull();
    expect(body.querySelector("[data-hub-ready]")).toBeNull();
  });

  it("picks up blocks added after load when the kit runs again or hubKit.init is called", () => {
    mount(`<p>start</p>`);
    expect(pageHas("hubKit")).toBe(true);
    document.body.insertAdjacentHTML("beforeend", COMPARE);
    expect(document.querySelector("input")).toBeNull();

    runScript("window.hubKit.init(document.body)");

    expect(document.querySelectorAll("input[type=range]")).toHaveLength(1);
  });

  it("does not implement the Hub bridge", () => {
    mount(COMPARE);

    expect(pageHas("hub")).toBe(false);
  });
});

describe("explainer kit: groups and copy", () => {
  beforeEach(() => {
    runScript("delete window.hubKit; window.__prefills = []; window.hub = { prefill: function (q, v) { window.__prefills.push([q, v]); } };");
  });
  afterEach(() => {
    document.body.innerHTML = "";
    runScript("delete window.hub; delete window.__prefills");
  });

  const GROUPS = `
    <div class="hub-groups" data-hub-groups="add">
      <section class="hub-group" data-key="keys"><button class="hub-group__toggle" aria-pressed="true">Keys</button></section>
      <section class="hub-group" data-key="logs"><button class="hub-group__toggle" aria-pressed="false">Logs</button></section>
    </div>`;

  function prefills(): unknown {
    document.body.removeAttribute("data-probe");
    runScript(`document.body.setAttribute("data-probe", JSON.stringify(window.__prefills))`);
    return JSON.parse(document.body.getAttribute("data-probe") ?? "null");
  }

  it("toggles a group and fills the form's multi-select with every selected key", () => {
    const body = mount(GROUPS);
    body.querySelectorAll<HTMLButtonElement>(".hub-group__toggle")[1]?.click();
    expect(body.querySelectorAll(".hub-group__toggle")[1]?.getAttribute("aria-pressed")).toBe("true");
    body.querySelectorAll<HTMLButtonElement>(".hub-group__toggle")[0]?.click();
    expect(prefills()).toEqual([
      ["add", { selected: ["keys", "logs"] }],
      ["add", { selected: ["logs"] }],
    ]);
  });

  it("selects the command text when the clipboard is not available", () => {
    const body = mount(`<section class="hub-command"><div class="hub-command__line"><code>vrooli setup --rebaseline</code><button data-hub-copy>Copy</button></div></section>`);
    const button = body.querySelector<HTMLButtonElement>("[data-hub-copy]");
    button?.click();
    expect(window.getSelection()?.toString()).toBe("vrooli setup --rebaseline");
    expect(button?.textContent).toMatch(/Copied|Selected: hold to copy/);
  });
});

describe("explainer kit: stylesheet contract", () => {
  it("keeps every rule inside the hub-kit layer", () => {
    const stripped = kitCss.replace(/\/\*[\s\S]*?\*\//g, "").trim();

    expect(stripped.startsWith("@layer hub-kit {")).toBe(true);
    expect(stripped.endsWith("}")).toBe(true);
    // Exactly one top-level block: the layer. Depth never returns to zero early.
    let depth = 0;
    let closedAt = -1;
    for (let i = 0; i < stripped.length; i++) {
      if (stripped[i] === "{") depth++;
      if (stripped[i] === "}" && --depth === 0) {
        closedAt = i;
        break;
      }
    }
    expect(closedAt).toBe(stripped.length - 1);
  });

  // A page that sets its own root colours replaces the kit's, because the kit
  // colours html and body inherits; a kit colour on body would beat the page's
  // root colour and leave light text on the page's own white cards.
  it("colours html, not body, so a page's root colours reach its text", () => {
    const stripped = kitCss.replace(/\/\*[\s\S]*?\*\//g, "");
    const html = /\n\s*html \{([^}]*)\}/.exec(stripped)?.[1] ?? "";
    const body = /\n\s*body \{([^}]*)\}/.exec(stripped)?.[1] ?? "";
    expect(html).toContain("color: var(--hub-fg)");
    expect(html).toContain("background: var(--hub-bg)");
    expect(body).not.toMatch(/(^|[\s;])(color|background)\s*:/);
  });

  it("uses plain colour values, so an older browser never loses text or backgrounds", () => {
    expect(kitCss).not.toContain("light-dark(");
  });

  it("lets the Hub theme attribute win over the device preference", () => {
    const stripped = kitCss.replace(/\/\*[\s\S]*?\*\//g, "");
    const tokens = (block: string): string[] =>
      [...block.matchAll(/(--hub-[a-z-]+):\s*(#[0-9a-f]{3,8})\s*;/g)].map((m) => `${m[1]}:${m[2]}`);
    const light = /:root \{([^}]*)\}/.exec(stripped)?.[1] ?? "";
    const device = /prefers-color-scheme: dark\) \{\s*:root:not\(\[data-theme="light"\]\) \{([^}]*)\}/.exec(stripped)?.[1] ?? "";
    const forced = /:root\[data-theme="dark"\] \{([^}]*)\}/.exec(stripped)?.[1] ?? "";

    // The device rule steps aside for an explicit light theme.
    expect(device).toContain("color-scheme: dark");
    expect(forced).toContain("color-scheme: dark");
    expect(light).toContain("color-scheme: light");
    // Both dark paths carry the same palette, and it covers every colour token.
    expect(tokens(device).length).toBeGreaterThanOrEqual(12);
    expect(tokens(device)).toEqual(tokens(forced));
    expect(tokens(device).map((t) => t.split(":")[0])).toEqual(tokens(light).map((t) => t.split(":")[0]));
    expect(tokens(device)).not.toEqual(tokens(light));
    // The device rule comes before the forced one, so equal specificity never lets it win.
    expect(stripped.indexOf("prefers-color-scheme")).toBeLessThan(stripped.indexOf(':root[data-theme="dark"]'));
  });

  it("defines every token the stylesheet uses", () => {
    // --hub-pos is the slider position that kit.js sets at runtime.
    const defined = new Set([...kitCss.matchAll(/(--hub-[a-z0-9-]+):/g)].map((m) => m[1]).concat("--hub-pos"));
    const used = [...kitCss.matchAll(/var\((--hub-[a-z0-9-]+)/g)].map((m) => m[1]);

    expect(used.filter((name) => !defined.has(name))).toEqual([]);
  });

  // Every page carries the kit. The budget was 10 KiB before the decision
  // blocks (proportion, groups, command, recurrence, matrix) took it to about
  // 18 KiB; a new block has to earn its bytes against this line.
  it("stays small", () => {
    const stripped = kitCss.replace(/\/\*[\s\S]*?\*\//g, "");

    expect(stripped.length).toBeLessThan(20 * 1024);
  });

  it("names every documented building block", () => {
    for (const name of [
      ".hub-pair",
      ".hub-compare",
      ".hub-timeline",
      ".hub-tradeoffs",
      ".hub-callout",
      ".hub-figure",
      ".hub-chip",
      ".hub-proportion",
      ".hub-groups",
      ".hub-command",
      ".hub-recurrence",
      ".hub-matrix",
    ]) {
      expect(kitCss).toContain(name);
    }
  });
});
