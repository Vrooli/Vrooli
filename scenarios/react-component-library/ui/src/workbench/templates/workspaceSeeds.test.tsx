import { readFileSync } from "node:fs";
import { resolve } from "node:path";

import { cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { renderWithProviders } from "../../test-utils";
import type { Template, TemplateNode, TemplateNodeType } from "./spec";
import { TemplateRenderer } from "./TemplateRenderer";

/**
 * The curated shelf is not a second authoring model. Every entry is an ordinary
 * workspace aggregate, and its composition is the same TemplateNode tree the
 * studio renders and edits. This test reads the reviewed file as data and
 * renders every display through the public renderer, so a template cannot ship
 * with a node type the studio does not support.
 */
// Vitest runs with the UI package as its root, so the reviewed shelf is one
// level up in library/.
const SEED_PATH = resolve(process.cwd(), "../library/workspace-seeds.json");

const KNOWN_TYPES = new Set<TemplateNodeType>([
  "page",
  "stack",
  "row",
  "grid",
  "surface",
  "text",
  "button",
  "input",
  "card",
  "stat",
  "metric",
  "icon",
  "avatar",
  "chip",
  "meter",
  "divider",
  "spacer",
  "slot",
  "swatch",
  "image",
]);

interface SeedState {
  name: string;
  argsJson: string;
}
interface SeedDisplay {
  id: string;
  device: string;
  states: SeedState[];
}
interface SeedPage {
  id: string;
  name: string;
  specJson: string;
  displays: SeedDisplay[];
}
interface SeedWorkspace {
  id: string;
  name: string;
  themeJson: string;
  optionsJson: string;
  provenance: string;
  pages: SeedPage[];
}

function templateFor(workspace: SeedWorkspace, spec: TemplateNode): Template {
  const palette = JSON.parse(workspace.themeJson) as Template["palette"];
  const options = JSON.parse(workspace.optionsJson || "{}") as {
    radius?: number;
    display?: "serif" | "sans";
  };
  return {
    id: workspace.id,
    name: workspace.name,
    tagline: "",
    category: "analytics",
    idea: "",
    slotHint: "",
    palette,
    radius: options.radius ?? 12,
    display: options.display ?? "sans",
    spec,
  };
}

function walk(node: TemplateNode, visit: (current: TemplateNode) => void) {
  visit(node);
  for (const child of node.children ?? []) walk(child, visit);
}

describe("curated workspace seeds", () => {
  afterEach(() => cleanup());

  const seed = JSON.parse(readFileSync(SEED_PATH, "utf8")) as {
    schemaVersion: number;
    seedVersion: number;
    workspaces: SeedWorkspace[];
  };

  it("is a versioned shelf of curated workspace aggregates", () => {
    expect(seed.schemaVersion).toBe(1);
    expect(seed.seedVersion).toBeGreaterThan(0);
    expect(seed.workspaces.length).toBe(16);
    expect(
      seed.workspaces
        .filter((workspace) => workspace.id.startsWith("ws_curated_eight_"))
        .map((workspace) => workspace.pages[0]?.name),
    ).toEqual([
      "Money, night",
      "Money, day",
      "Run, before",
      "Wind-down",
      "Plant, day",
      "Phrase practice",
      "Run, after",
      "Plant, night",
    ]);
    for (const workspace of seed.workspaces) {
      expect(workspace.provenance).toBe("WORKSPACE_PROVENANCE_CURATED");
      expect(workspace.pages.length).toBeGreaterThan(0);
      for (const page of workspace.pages) {
        expect(page.displays.length).toBeGreaterThan(0);
        for (const display of page.displays) {
          expect(display.states.length).toBeGreaterThan(0);
        }
      }
    }
  });

  it("ships Harbor as a four-page responsive onboarding flow", () => {
    const harbor = seed.workspaces.find((workspace) => workspace.id === "ws_curated_harbor");
    expect(harbor).toBeDefined();
    expect(harbor?.pages.map((page) => page.name)).toEqual([
      "Create your workspace",
      "Invite your team",
      "Connect a source",
      "Your workspace is ready",
    ]);
    expect(harbor?.pages).toHaveLength(4);

    const displays = harbor?.pages.flatMap((page) => page.displays) ?? [];
    expect(displays).toHaveLength(12);
    expect(displays.map((display) => display.device)).toEqual(
      Array.from({ length: 4 }, () => [
        "DISPLAY_DEVICE_DESKTOP",
        "DISPLAY_DEVICE_TABLET",
        "DISPLAY_DEVICE_PHONE",
      ]).flat(),
    );
    expect(displays.flatMap((display) => display.states)).toHaveLength(12);
    for (const display of displays) {
      expect(display.states, `${display.id} must have one named state`).toHaveLength(1);
      expect(display.states[0]?.name.trim().length).toBeGreaterThan(0);
    }
  });

  it("renders imported screen actions through the shared RCL Button", () => {
    const studies = seed.workspaces.filter((workspace) =>
      workspace.id.startsWith("ws_curated_eight_"),
    );
    let authoredButtonCount = 0;

    for (const workspace of studies) {
      for (const page of workspace.pages) {
        for (const display of page.displays) {
          for (const state of display.states) {
            const spec = (JSON.parse(state.argsJson) as { spec: TemplateNode }).spec;
            const authoredButtons: TemplateNode[] = [];
            walk(spec, (node) => {
              if (node.type === "button") authoredButtons.push(node);
            });
            if (authoredButtons.length === 0) continue;

            authoredButtonCount += authoredButtons.length;
            const { container } = renderWithProviders(
              <TemplateRenderer
                template={templateFor(workspace, spec)}
                spec={spec}
                mode="preview"
              />,
            );
            const renderedButtons = [
              ...container.querySelectorAll<HTMLElement>('[data-node-type="button"]'),
            ];
            expect(renderedButtons).toHaveLength(authoredButtons.length);
            for (const renderedButton of renderedButtons) {
              expect(renderedButton.querySelector('[data-rcl-button="true"]')).not.toBeNull();
            }
            cleanup();
          }
        }
      }
    }

    expect(studies).toHaveLength(8);
    expect(authoredButtonCount).toBeGreaterThan(0);
  });

  it("keeps the money month interaction as a native editable workspace state", () => {
    const money = seed.workspaces.find(
      (workspace) => workspace.id === "ws_curated_eight_money-night",
    );
    const states = money?.pages[0]?.displays[0]?.states ?? [];
    const month = states.find((state) => state.name === "Month");
    expect(month).toBeDefined();

    const nodes: TemplateNode[] = [];
    walk((JSON.parse(month?.argsJson ?? "{}") as { spec: TemplateNode }).spec, (node) =>
      nodes.push(node),
    );
    const text = nodes
      .map((node) => node.props.text)
      .filter((value): value is string => typeof value === "string");
    expect(text).toContain("Spent this month");
    expect(text).toContain("€1,284.10");
    expect(text).toContain("€1,900");
    expect(text).toContain("412.00");
    expect(text.filter((value) => /^W[1-5]$/.test(value))).toEqual(["W1", "W2", "W3", "W4", "W5"]);

    const bars =
      nodes.find((node) => node.id === "st_kd5bkbonpjay3h-html-71-row-month")?.children ?? [];
    expect(bars).toHaveLength(5);
    expect(bars.map((bar) => JSON.parse(String(bar.props.css))["--h"])).toEqual([
      "62%",
      "46%",
      "90%",
      "72%",
      "34%",
    ]);
  });

  it("exposes the saved Money-night range selection without changing its appearance", () => {
    const money = seed.workspaces.find(
      (workspace) => workspace.id === "ws_curated_eight_money-night",
    );
    expect(money).toBeDefined();
    const states =
      money?.pages.flatMap((page) => page.displays.flatMap((display) => display.states)) ?? [];
    expect(states.length).toBeGreaterThan(0);

    for (const state of states) {
      const spec = (JSON.parse(state.argsJson) as { spec: TemplateNode }).spec;
      const rangeButtons: TemplateNode[] = [];
      walk(spec, (node) => {
        if (node.type === "button" && ["Week", "Month"].includes(String(node.props.text))) {
          rangeButtons.push(node);
        }
      });
      expect(rangeButtons).toHaveLength(2);
      const expectedSelection = rangeButtons.map((node) => {
        const css = JSON.parse(String(node.props.css)) as Record<string, string>;
        return {
          name: String(node.props.text),
          pressed: css.background !== "none",
          background: css.background,
          color: css.color,
        };
      });
      const { container } = renderWithProviders(
        <TemplateRenderer template={templateFor(money!, spec)} spec={spec} mode="preview" />,
      );

      for (const expected of expectedSelection) {
        const button = screen.getByRole("button", { name: expected.name });
        expect(button).toHaveAttribute("aria-pressed", String(expected.pressed));
        expect(button).toHaveStyle({ background: expected.background, color: expected.color });
      }
      expect(container.querySelectorAll('[data-node-type="button"]')).toHaveLength(
        rangeButtons.length + 7,
      );
      cleanup();
    }
  });

  it("preserves the source-matched Plant-day leaf as a selectable SVG layer", () => {
    const plant = seed.workspaces.find(
      (workspace) => workspace.id === "ws_curated_eight_plant-day",
    );
    expect(plant).toBeDefined();
    expect(JSON.parse(plant?.themeJson ?? "{}").bg).toBe("#d7ef66");
    const state = plant?.pages[0]?.displays[0]?.states[0];
    expect(state).toBeDefined();
    const spec = (JSON.parse(state?.argsJson ?? "{}") as { spec: TemplateNode }).spec;
    const layers: TemplateNode[] = [];
    walk(spec, (node) => layers.push(node));
    const leaf = layers.find((node) => node.id.endsWith("html-322-image"));
    const leafStage = layers.find((node) => node.id.endsWith("html-321-stack"));
    expect(leaf?.type).toBe("image");
    expect(leaf?.props.src).toMatch(/^data:image\/svg\+xml;base64,/);
    expect(JSON.parse(String(leafStage?.props.css))).toMatchObject({
      position: "relative",
      height: "300px",
      margin: "-10px -16px 0",
    });
    expect(JSON.parse(String(leaf?.props.css))).toMatchObject({
      position: "absolute",
      width: "272px",
      height: "321px",
    });

    const svg = new DOMParser().parseFromString(
      atob(String(leaf?.props.src).split(",")[1] ?? ""),
      "image/svg+xml",
    );
    expect(svg.querySelector("parsererror")).toBeNull();
    expect(svg.documentElement.getAttribute("viewBox")).toBe("0 0 220 260");
    const silhouette = svg.querySelector("path")?.getAttribute("d") ?? "";
    expect(silhouette).toMatch(/^M110 12C91 20/);
    expect((silhouette.match(/C/g) ?? []).length).toBeGreaterThan(10);
    const vein = svg.querySelector('path[stroke="#d7ef66"]');
    expect(vein?.getAttribute("stroke-width")).toBe("2.5");
    expect(vein?.getAttribute("opacity")).toBe(".55");
    expect(svg.querySelector('g[fill="#d7ef66"]')).not.toBeNull();
    const stem = [...svg.querySelectorAll("path")].find((path) =>
      path.getAttribute("d")?.startsWith("M110 198C111 218"),
    );
    expect(stem?.getAttribute("stroke")).toBe("#14523f");
    expect(stem?.getAttribute("stroke-width")).toBe("6");

    const { container } = renderWithProviders(
      <TemplateRenderer template={templateFor(plant!, spec)} spec={spec} mode="preview" />,
    );
    const renderedLeaf = container.querySelector(`[data-node-id="${leaf?.id}"]`);
    expect(renderedLeaf?.tagName.toLowerCase()).toBe("figure");
    expect(renderedLeaf?.querySelector("img")).toHaveAttribute("src", leaf?.props.src);
    expect(screen.getByRole("button", { name: /Watered/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Skip tonight/i })).toBeInTheDocument();
  });

  it("keeps the reference Plant-day watering facts as three editable data layers", () => {
    const plant = seed.workspaces.find(
      (workspace) => workspace.id === "ws_curated_eight_plant-day",
    );
    expect(plant).toBeDefined();
    const spec = (
      JSON.parse(plant?.pages[0]?.displays[0]?.states[0]?.argsJson ?? "{}") as {
        spec: TemplateNode;
      }
    ).spec;
    const layers: TemplateNode[] = [];
    walk(spec, (node) => layers.push(node));
    const facts = layers.find((node) => node.id.endsWith("html-324-row"));
    expect(facts?.children).toHaveLength(3);
    expect(layers.find((node) => node.id.endsWith("html-356-text"))?.props.text).toBe("650 ml");
    expect(layers.find((node) => node.id.endsWith("html-357-text"))?.props.text).toBe(
      "every 6 days",
    );
    expect(new Set(layers.map((node) => node.id)).size).toBe(layers.length);
    const thumbnails = ["html-333-image", "html-335-image", "html-337-image"].map((suffix) =>
      layers.find((node) => node.id.endsWith(suffix)),
    );
    expect(thumbnails.every((node) => node?.type === "image")).toBe(true);
    for (const thumbnail of thumbnails) {
      expect(thumbnail?.props.src).toMatch(/^data:image\/jpeg;base64,/);
      expect(thumbnail?.props.fit).toBe("cover");
      expect(String(thumbnail?.props.src).length).toBeLessThan(2_500);
    }
    expect(JSON.parse(String(facts?.props.css))).toMatchObject({
      display: "flex",
      "justify-content": "space-between",
      gap: "14px",
      width: "100%",
    });

    const pageSpec = JSON.parse(plant?.pages[0]?.specJson ?? "{}") as TemplateNode;
    const pageLayers: TemplateNode[] = [];
    walk(pageSpec, (node) => pageLayers.push(node));
    expect(pageLayers.find((node) => node.id.endsWith("html-324-row"))?.children).toHaveLength(3);
    expect(pageLayers.find((node) => node.id.endsWith("html-356-text"))?.props.text).toBe("650 ml");

    const rendered = renderWithProviders(
      <TemplateRenderer template={templateFor(plant!, spec)} spec={spec} mode="preview" />,
    );
    expect(rendered.container.textContent).toContain("650 ml");
    expect(rendered.container.textContent).toContain("every 6 days");
  });

  it("matches the Plant-night reference scale, copy hierarchy, and selected Plants tab", () => {
    const plant = seed.workspaces.find(
      (workspace) => workspace.id === "ws_curated_eight_plant-night",
    );
    expect(plant).toBeDefined();
    const spec = (
      JSON.parse(plant?.pages[0]?.displays[0]?.states[0]?.argsJson ?? "{}") as {
        spec: TemplateNode;
      }
    ).spec;
    const layers: TemplateNode[] = [];
    walk(spec, (node) => layers.push(node));
    expect(new Set(layers.map((node) => node.id)).size).toBe(layers.length);

    const stage = layers.find((node) => node.id.endsWith("html-497-stack"));
    expect(JSON.parse(String(stage?.props.css))).toMatchObject({ height: "416px" });
    const art = layers.find((node) => node.id.endsWith("html-498-image"));
    expect(art?.type).toBe("image");
    expect(JSON.parse(String(art?.props.css))).toMatchObject({
      left: "42px",
      top: "46px",
      width: "320px",
      height: "375px",
    });
    const svg = new DOMParser().parseFromString(
      atob(String(art?.props.src).split(",")[1] ?? ""),
      "image/svg+xml",
    );
    expect(svg.querySelector("parsererror")).toBeNull();
    expect(svg.documentElement.getAttribute("viewBox")).toBe("-12 0 252 320");
    expect(svg.querySelector('g[transform="translate(-22,6) scale(1.16,.86)"]')).not.toBeNull();
    expect(svg.querySelector('path[d="M-4 260h82l-10 49H6z"]')).not.toBeNull();

    const title = layers.find((node) => node.id.endsWith("html-499-text"));
    const copy = layers.find((node) => node.id.endsWith("html-500-text"));
    expect(JSON.parse(String(title?.props.css))).toMatchObject({ "font-size": "40px" });
    expect(JSON.parse(String(copy?.props.css))).toMatchObject({
      "font-size": "16px",
      "max-width": "300px",
    });

    const nav = layers.find((node) => node.id.endsWith("html-505-row"));
    expect(nav?.children).toHaveLength(4);
    const navLabels = nav?.children?.map((entry) => entry.children?.[1]?.props.text);
    expect(navLabels).toEqual(["Tonight", "Plants", "Identify", "Search"]);
    expect(JSON.parse(String(nav?.children?.[0]?.props.css))).toMatchObject({
      background: "none",
    });
    expect(JSON.parse(String(nav?.children?.[1]?.props.css))).toMatchObject({
      background: "var(--nav-active)",
      color: "var(--nav-active-ink)",
    });
  });

  it("uses the screenshot's three-item Phrase-practice navigation with an RCL book icon", () => {
    const phrase = seed.workspaces.find(
      (workspace) => workspace.id === "ws_curated_eight_phrase-practice",
    );
    expect(phrase).toBeDefined();
    const spec = (
      JSON.parse(phrase?.pages[0]?.displays[0]?.states[0]?.argsJson ?? "{}") as {
        spec: TemplateNode;
      }
    ).spec;
    const layers: TemplateNode[] = [];
    walk(spec, (node) => layers.push(node));
    expect(new Set(layers.map((node) => node.id)).size).toBe(layers.length);
    const nav = layers.find((node) => node.id.endsWith("html-396-row"));
    expect(nav?.children).toHaveLength(3);
    expect(nav?.children?.map((entry) => entry.children?.[1]?.props.text)).toEqual([
      "Today",
      "Flashcards",
      "Progress",
    ]);
    expect(nav?.children?.[1]?.children?.[0]).toMatchObject({
      type: "icon",
      props: { name: "book", size: 18 },
    });
  });

  it("preserves the source colors and scale of the Wind-down scene and Plant-night art", () => {
    const windDown = seed.workspaces.find(
      (workspace) => workspace.id === "ws_curated_eight_wind-down",
    );
    const plantNight = seed.workspaces.find(
      (workspace) => workspace.id === "ws_curated_eight_plant-night",
    );
    expect(windDown).toBeDefined();
    expect(plantNight).toBeDefined();
    expect(JSON.parse(plantNight?.themeJson ?? "{}").bg).toBe("#0b39be");

    const windSpec = (
      JSON.parse(windDown?.pages[0]?.displays[0]?.states[0]?.argsJson ?? "{}") as {
        spec: TemplateNode;
      }
    ).spec;
    const windNodes: TemplateNode[] = [];
    walk(windSpec, (node) => windNodes.push(node));
    const sceneNodes = ["html-255-image", "html-256-image", "html-257-image"].map((suffix) =>
      windNodes.find((node) => node.id.endsWith(suffix)),
    );
    expect(sceneNodes.every((node) => node?.type === "image")).toBe(true);
    expect(JSON.parse(String(sceneNodes[0]?.props.css))).toMatchObject({
      left: "34px",
      top: "118px",
      width: "62px",
      height: "62px",
    });
    expect(JSON.parse(String(sceneNodes[1]?.props.css))).toMatchObject({
      top: "206px",
      height: "264px",
    });
    expect(JSON.parse(String(sceneNodes[2]?.props.css))).toMatchObject({
      top: "226px",
      height: "252px",
    });
    const housesSvg = atob(String(sceneNodes[2]?.props.src).split(",")[1] ?? "");
    expect(housesSvg).toContain(
      'd="M164 62c-8-8 6-12 0-22s6-14 0-24" stroke="#c9d4d8" stroke-width="2.4"',
    );

    const plantSpec = (
      JSON.parse(plantNight?.pages[0]?.displays[0]?.states[0]?.argsJson ?? "{}") as {
        spec: TemplateNode;
      }
    ).spec;
    const plantNodes: TemplateNode[] = [];
    walk(plantSpec, (node) => plantNodes.push(node));
    const pot = plantNodes.find((node) => node.id.endsWith("html-498-image"));
    expect(pot?.type).toBe("image");
    expect(JSON.parse(String(pot?.props.css))).toMatchObject({
      left: "42px",
      top: "46px",
      width: "320px",
      height: "375px",
    });
    const potSvg = atob(String(pot?.props.src).split(",")[1] ?? "");
    const plantSvg = new DOMParser().parseFromString(potSvg, "image/svg+xml");
    expect(plantSvg.querySelector("parsererror")).toBeNull();
    expect(plantSvg.documentElement.getAttribute("viewBox")).toBe("-12 0 252 320");
    expect(plantSvg.querySelector("g g path")?.getAttribute("d")).toBe(
      "M110 22C172 40 204 86 204 136c0 46-38 86-94 104-56-18-94-58-94-104C16 86 48 40 110 22Z",
    );
    expect(plantSvg.querySelector("g g path")?.getAttribute("fill")).toBe("#d7ef66");
    expect(plantSvg.querySelector("g g g")?.getAttribute("fill")).toBe(
      JSON.parse(plantNight?.themeJson ?? "{}").bg,
    );
    expect(potSvg).toContain('d="M110 32c-5 58-3 136 0 200" stroke="#0b39be" stroke-width="2"');
    expect(potSvg).toContain('d="M110 208c-20 17-48 29-70 40" stroke="#d7ef66" stroke-width="5"');
    expect(potSvg).toContain('d="M-4 260h82l-10 49H6z"');

    const { container } = renderWithProviders(
      <TemplateRenderer
        template={templateFor(plantNight!, plantSpec)}
        spec={plantSpec}
        mode="preview"
      />,
    );
    expect(container.querySelector(`[data-node-id="${pot?.id}"] img`)).toHaveAttribute(
      "src",
      pot?.props.src,
    );
    cleanup();
  });

  it("keeps explanatory study captions outside the eight imported screen canvases", () => {
    const captionStarts = [
      "01 · Money, night.",
      "02 · Money, day.",
      "03 · Run, before.",
      "04 · Wind-down.",
      "05 · Plant, day.",
      "06 · Phrase practice.",
      "07 · Run, after.",
      "08 · Plant, night.",
    ];
    const studies = seed.workspaces.filter((workspace) =>
      workspace.id.startsWith("ws_curated_eight_"),
    );
    for (const workspace of studies) {
      const specs = workspace.pages.flatMap((page) => [
        JSON.parse(page.specJson),
        ...page.displays.flatMap((display) =>
          display.states.map((state) => JSON.parse(state.argsJson || "{}").spec),
        ),
      ]);
      for (const rawSpec of specs) {
        if (!rawSpec) continue;
        const nodes: TemplateNode[] = [];
        walk(rawSpec as TemplateNode, (node) => nodes.push(node));
        expect(
          nodes
            .filter((node) => node.type === "text")
            .some((node) =>
              captionStarts.some((caption) => String(node.props.text ?? "").startsWith(caption)),
            ),
          workspace.name,
        ).toBe(false);
      }
    }
  });

  it("keeps the day jar coins and Run before navigation as editable layers", () => {
    const money = seed.workspaces.find((workspace) => workspace.name === "Money, day");
    const moneySpec = JSON.parse(money?.pages[0]?.displays[0]?.states[0]?.argsJson ?? "{}")
      .spec as TemplateNode;
    const moneyNodes: TemplateNode[] = [];
    walk(moneySpec, (node) => moneyNodes.push(node));
    expect(moneyNodes.filter((node) => node.id.startsWith("seed_money_day_coin_")).length).toBe(10);
    expect(
      moneyNodes.filter(
        (node) => node.type === "surface" && node.id.startsWith("seed_money_day_coin_"),
      ).length,
    ).toBe(9);

    const before = seed.workspaces.find((workspace) => workspace.name === "Run, before");
    const beforeSpec = JSON.parse(before?.pages[0]?.displays[0]?.states[0]?.argsJson ?? "{}")
      .spec as TemplateNode;
    const beforeNodes: TemplateNode[] = [];
    walk(beforeSpec, (node) => beforeNodes.push(node));
    expect(beforeNodes.find((node) => node.id.endsWith("html-228-text"))?.props.text).toBe("5:52");
    const beforePageSpec = JSON.parse(before?.pages[0]?.specJson ?? "{}") as TemplateNode;
    const beforePageNodes: TemplateNode[] = [];
    walk(beforePageSpec, (node) => beforePageNodes.push(node));
    expect(beforePageNodes.find((node) => node.id === "html-228-text")?.props.text).toBe("5:52");
    const expectedRunNav = [
      ["activity", "Run"],
      ["calendar", "Plan"],
      ["chart", "Stats"],
      ["route", "Routes"],
    ];
    const runNav = beforeNodes.find((node) => node.id === "seed_run_before_bottom_nav");
    expect(runNav?.children).toHaveLength(4);
    expect(
      runNav?.children?.map((item) => [
        item.children?.[0]?.props.name,
        item.children?.[1]?.props.text,
      ]),
    ).toEqual(expectedRunNav);
    const beforePageNav = beforePageNodes.find((node) => node.id === "seed_run_before_bottom_nav");
    expect(
      beforePageNav?.children?.map((item) => [
        item.children?.[0]?.props.name,
        item.children?.[1]?.props.text,
      ]),
    ).toEqual(expectedRunNav);

    const runContent = beforeNodes.find((node) => node.id === "st_run-before_phone-html-217-stack");
    const bottomNav = beforeNodes.find((node) => node.id === "seed_run_before_bottom_nav");
    const contentCss = JSON.parse(String(runContent?.props.css)) as Record<string, string>;
    const navCss = JSON.parse(String(bottomNav?.props.css)) as Record<string, string>;
    if (contentCss.padding === undefined) throw new Error("Expected content padding in the seed");
    const contentBottomPadding = Number.parseFloat(contentCss.padding.split(" ")[2] ?? "0");
    const navigationHeight = Number.parseFloat(navCss.height ?? "0");
    const navigationBottomInset = Number.parseFloat(navCss.inset?.split(" ")[2] ?? "0");
    expect(contentBottomPadding).toBeGreaterThanOrEqual(
      navigationHeight + navigationBottomInset + 30,
    );

    const { container } = renderWithProviders(
      <TemplateRenderer
        template={templateFor(before!, beforeSpec)}
        spec={beforeSpec}
        mode="preview"
      />,
    );
    expect(
      container.querySelector('[data-node-id="st_run-before_phone-html-217-stack"]'),
    ).toHaveStyle({ padding: "62px 24px 90px" });
    const renderedNav = container.querySelector('[data-node-id="seed_run_before_bottom_nav"]');
    expect(renderedNav).not.toBeNull();
    expect(renderedNav).toHaveStyle({ height: "46px", inset: "auto 0 14px" });
  });

  it("uses the shared RCL Button for source-authored Money-night category actions", () => {
    const money = seed.workspaces.find(
      (workspace) => workspace.id === "ws_curated_eight_money-night",
    );
    expect(money).toBeDefined();
    const moneyStates =
      money?.pages.flatMap((page) => page.displays.flatMap((display) => display.states)) ?? [];

    for (const state of moneyStates) {
      const spec = (JSON.parse(state.argsJson) as { spec: TemplateNode }).spec;
      const categories: TemplateNode[] = [];
      walk(spec, (node) => {
        if (
          ["html-89-stack", "html-96-stack", "html-103-stack"].some((id) => node.id.includes(id))
        ) {
          categories.push(node);
        }
      });
      expect(categories).toHaveLength(3);
      expect(categories.every((node) => node.type === "button")).toBe(true);
    }

    const spec = (JSON.parse(moneyStates[0]!.argsJson) as { spec: TemplateNode }).spec;
    const { container } = renderWithProviders(
      <TemplateRenderer template={templateFor(money!, spec)} spec={spec} mode="preview" />,
    );
    for (const name of ["Groceries", "Eating out", "Transport"]) {
      const button = screen.getByRole("button", { name: new RegExp(name) });
      expect(button).toHaveAttribute("data-rcl-button", "true");
      expect(button).toHaveStyle({
        background: "#1e2419",
        flex: "0 0 132px",
        borderRadius: "18px",
      });
    }
    expect(container.querySelectorAll('[data-node-type="button"]')).toHaveLength(9);
    cleanup();
  });

  it("retains Run after's source-backed seven-day weekly chart", () => {
    const after = seed.workspaces.find((workspace) => workspace.name === "Run, after");
    const spec = JSON.parse(after?.pages[0]?.displays[0]?.states[0]?.argsJson ?? "{}")
      .spec as TemplateNode;
    const chart = (() => {
      let found: TemplateNode | undefined;
      walk(spec, (node) => {
        if (
          node.type === "row" &&
          node.children?.length === 7 &&
          node.children.every((day) => day.type === "stack")
        )
          found = node;
      });
      return found;
    })();
    expect(chart?.children).toHaveLength(7);
    const labels: string[] = [];
    walk(chart!, (node) => {
      if (typeof node.props.text === "string") labels.push(node.props.text);
    });
    expect(labels).toEqual([
      "8.2",
      "M",
      "10",
      "T",
      "6",
      "W",
      "8.6",
      "T",
      "–",
      "F",
      "16",
      "S",
      "7",
      "S",
    ]);
  });

  it("keeps all 34 study navigation actions backed by the shared Button", () => {
    const study = seed.workspaces.filter((workspace) =>
      workspace.id.startsWith("ws_curated_eight_"),
    );
    let groups = 0;
    let buttons = 0;
    for (const workspace of study) {
      for (const page of workspace.pages) {
        for (const display of page.displays) {
          for (const state of display.states) {
            const spec = (JSON.parse(state.argsJson) as { spec: TemplateNode }).spec;
            walk(spec, (current) => {
              if (
                current.type !== "row" ||
                !String(current.props.css ?? "").includes('"z-index":"8"') ||
                !current.children?.length
              )
                return;
              groups += 1;
              for (const action of current.children) {
                expect(action.type, `${workspace.name}/${display.id}/${state.name}`).toBe("button");
                const nested: TemplateNode[] = [];
                walk(action, (node) => nested.push(node));
                expect(nested.some((node) => node.type === "image" || node.type === "icon")).toBe(
                  true,
                );
                expect(
                  nested.some((node) => typeof node.props.text === "string" && node.props.text),
                ).toBe(true);
                buttons += 1;
              }
            });
          }
        }
      }
    }
    expect(groups).toBe(9);
    expect(buttons).toBe(33);
  });

  it("keeps the four short source toggle actions backed by Button with pressed state", () => {
    const expected = new Map([
      ["Run, before", "Start"],
      ["Run, after", "Start"],
      ["Plant, day", "Watered"],
      ["Plant, night", "Watered"],
    ]);
    for (const [workspaceName, label] of expected) {
      const workspace = seed.workspaces.find((entry) => entry.name === workspaceName);
      expect(workspace).toBeDefined();
      const controls: TemplateNode[] = [];
      for (const page of workspace?.pages ?? []) {
        for (const display of page.displays) {
          for (const state of display.states) {
            walk((JSON.parse(state.argsJson) as { spec: TemplateNode }).spec, (node) => {
              if (node.type === "button" && node.props.pressed === false) controls.push(node);
            });
          }
        }
      }
      expect(controls, workspaceName).toHaveLength(1);
      expect(controls[0]?.props.text).toBe("");
      const labels: string[] = [];
      walk(controls[0]!, (node) => {
        if (typeof node.props.text === "string" && node.props.text) labels.push(node.props.text);
      });
      expect(labels).toContain(label);
    }
  });

  it("keeps the Run after reference fit editable and documents source CSS departures", () => {
    const workspace = seed.workspaces.find((entry) => entry.id === "ws_curated_eight_run-after");
    expect(workspace).toBeDefined();
    const nodes: TemplateNode[] = [];
    for (const page of workspace?.pages ?? []) {
      for (const display of page.displays) {
        for (const state of display.states) {
          walk((JSON.parse(state.argsJson) as { spec: TemplateNode }).spec, (node) =>
            nodes.push(node),
          );
        }
      }
    }
    const byId = (id: string) => nodes.find((node) => node.id === id);
    const css = (id: string) => JSON.parse(String(byId(id)?.props.css)) as Record<string, string>;
    const rulerCap = byId("st_run-after_phone-html-430-surface");
    expect(rulerCap?.type).toBe("surface");
    expect(rulerCap?.props.label).toBe("Ruler cap [reference 6x109px; source CSS 3x26px]");
    expect(css("st_run-after_phone-html-423-stack").padding).toBe("30px 24px 0");
    expect(css("st_run-after_phone-html-432-row")).toMatchObject({
      "font-size": "300px",
      transform: "translateX(-5px) scaleX(.66)",
      "transform-origin": "left top",
    });
    expect(css("st_run-after_phone-html-433-text")["flex-shrink"]).toBe("0");
    expect(byId("st_run-after_phone-html-434-text")?.props.label).toBe(
      "KM [reference 74x49px; source CSS 30px]",
    );
    expect(css("st_run-after_phone-html-434-text")).toMatchObject({
      "font-size": "70px",
      "margin-left": "30px",
      transform: "translateY(2px) scaleX(1.17)",
    });
    expect(byId("st_run-after_phone-html-435-text")?.props.text).toBe(
      "Easy at 5:50 to 6:05.\nLast 3 km at race pace, 5:05.",
    );
    expect(css("st_run-after_phone-html-435-text")).toMatchObject({
      "white-space": "pre-line",
      "font-size": "22px",
      "margin-top": "27px",
      "margin-left": "21px",
      "font-weight": "500",
      transform: "translateY(-1px) scaleX(1.08)",
      "line-height": "1.36",
    });
    expect(css("st_run-after_phone-html-436-row")).toMatchObject({
      "margin-top": "79px",
      "padding-top": "0",
      "padding-left": "24px",
      "border-top": "0",
    });
    for (const index of [0, 1, 2, 3, 4, 5, 6]) {
      const marker = css(`st_run-after_phone-html-${439 + index * 4}-surface`);
      expect(marker.height).toBe(index === 1 || index === 5 ? "6px" : "5px");
      expect(marker.width).toBe(index === 4 ? "12px" : "40px");
      expect(marker["--h"]).toBeUndefined();
      expect(css(`st_run-after_phone-html-${437 + index * 4}-stack`)).toMatchObject({
        "justify-content": "flex-start",
        gap: "13px",
      });
      expect(css(`st_run-after_phone-html-${438 + index * 4}-text`)).toMatchObject({
        transform: "translateY(-2px) scaleY(1.7)",
        "transform-origin": "center top",
      });
    }
    expect(css("st_run-after_phone-html-439-surface").height).toBe("5px");
    expect(css("st_run-after_phone-html-443-surface").height).toBe("6px");
    expect(css("st_run-after_phone-html-443-surface").background).toBe("#e8402c");
    expect(css("st_run-after_phone-html-459-surface")).toMatchObject({
      background: "transparent",
      border: "1px solid rgba(255, 255, 255, .95)",
    });
    expect(css("st_run-after_phone-html-438-text")).toMatchObject({
      "font-size": "16px",
      "line-height": "1",
    });
    expect(css("st_run-after_phone-html-440-text")["font-size"]).toBe("12px");
    expect(byId("st_run-after_phone-html-465-row")?.props.label).toBe(
      "Start CTA [reference bottom 46px; source CSS 12px]",
    );
    expect(css("st_run-after_phone-html-465-row")["margin-bottom"]).toBe("46px");
    expect(byId("st_run-after_phone-html-467-row")?.props.label).toBe(
      "Nav [reference 3 items; source HTML 4]",
    );
    expect(byId("st_run-after_phone-html-467-row")?.children?.map((node) => node.id)).toEqual([
      "st_run-after_phone-html-468-row",
      "st_run-after_phone-html-471-stack",
      "st_run-after_phone-html-477-stack",
    ]);
    expect(css("st_run-after_phone-html-430-surface")).toMatchObject({
      left: "-4px",
      top: "-18px",
      width: "6px",
      height: "109px",
      background: "#e8402c",
    });
  });

  it("uses unique layer ids across every display in a workspace", () => {
    for (const workspace of seed.workspaces) {
      const ids = new Set<string>();
      for (const page of workspace.pages) {
        for (const display of page.displays) {
          for (const state of display.states) {
            const parsed = JSON.parse(state.argsJson) as { spec: TemplateNode };
            walk(parsed.spec, (node) => {
              expect(ids.has(node.id), `${workspace.id}: layer id ${node.id} is reused`).toBe(
                false,
              );
              ids.add(node.id);
            });
          }
        }
      }
    }
  });

  it("uses only composition vocabulary the studio already supports", () => {
    for (const workspace of seed.workspaces) {
      for (const page of workspace.pages) {
        for (const display of page.displays) {
          for (const state of display.states) {
            const parsed = JSON.parse(state.argsJson) as { spec: TemplateNode };
            expect(parsed.spec.type).toBe("page");
            walk(parsed.spec, (node) => {
              expect(
                KNOWN_TYPES.has(node.type),
                `${workspace.id}/${display.id}: unsupported node type ${node.type}`,
              ).toBe(true);
              expect(node.id.length).toBeGreaterThan(0);
            });
          }
        }
      }
    }
  });

  it("keeps every Money-night category reachable in its horizontal rail", () => {
    const money = seed.workspaces.find(
      (workspace) => workspace.id === "ws_curated_eight_money-night",
    );
    const state = money?.pages[0]?.displays[0]?.states[0];
    expect(state).toBeDefined();
    const spec = (JSON.parse(state?.argsJson ?? "{}") as { spec: TemplateNode }).spec;
    const { container } = renderWithProviders(
      <TemplateRenderer template={templateFor(money!, spec)} spec={spec} mode="preview" />,
    );

    const rail = container.querySelector('[data-node-id$="html-88-row"]');
    expect(rail).toHaveAttribute("role", "region");
    expect(rail).toHaveAttribute(
      "aria-label",
      "Categories. Scroll horizontally to see all categories.",
    );
    expect(rail).toHaveAttribute("tabindex", "0");
    expect(rail).toHaveStyle({
      overflowX: "auto",
      scrollSnapType: "x mandatory",
      scrollPaddingInline: "22px",
    });
    for (const name of ["Groceries", "Eating out", "Transport"]) {
      expect(container).toHaveTextContent(name);
    }
    expect(container.querySelector('[data-node-id$="html-103-stack"]')).toHaveStyle({
      scrollSnapAlign: "start",
    });
  });

  it("renders every display through the public renderer with no unknown layer", () => {
    const expected = seed.workspaces.reduce(
      (total, workspace) =>
        total +
        workspace.pages.reduce(
          (pages, page) =>
            pages +
            page.displays.reduce((displays, display) => displays + display.states.length, 0),
          0,
        ),
      0,
    );
    let rendered = 0;
    for (const workspace of seed.workspaces) {
      for (const page of workspace.pages) {
        for (const display of page.displays) {
          for (const state of display.states) {
            const parsed = JSON.parse(state.argsJson) as { spec: TemplateNode };
            const template = templateFor(workspace, parsed.spec);
            const { container } = renderWithProviders(
              <TemplateRenderer template={template} spec={parsed.spec} mode="preview" />,
            );
            expect(
              container.querySelector(".tpl-unknown"),
              `${workspace.id}/${display.id}: renderer could not draw a layer`,
            ).toBeNull();
            expect(container.querySelector(".tpl-page")).not.toBeNull();
            expect(screen.queryByText(/Unsupported layer type/i)).toBeNull();
            cleanup();
            rendered += 1;
          }
        }
      }
    }
    expect(rendered).toBe(expected);
  });
});
