import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils";
import { describe, expect, it } from "vitest";
import { strings } from "../../consts/strings";

import { TemplateRenderer } from "./TemplateRenderer";
import { DEFAULT_TEMPLATE } from "./library";
import { node, type TemplateNode } from "./spec";

const WORKSPACE_SEEDS_PATH = resolve(process.cwd(), "../library/workspace-seeds.json");

function findNode(root: TemplateNode, id: string): TemplateNode | undefined {
  if (root.id === id) return root;
  for (const child of root.children ?? []) {
    const found = findNode(child, id);
    if (found) return found;
  }
  return undefined;
}

describe("TemplateNodeRenderer authored content", () => {
  it("renders the source-backed Plant-day Monstera as an accessible editable image layer", () => {
    const seeds = JSON.parse(readFileSync(WORKSPACE_SEEDS_PATH, "utf8")) as {
      workspaces: Array<{
        id: string;
        pages: Array<{ displays: Array<{ states: Array<{ argsJson: string }> }> }>;
      }>;
    };
    const plantDay = seeds.workspaces.find(
      (workspace) => workspace.id === "ws_curated_eight_plant-day",
    );
    const spec = JSON.parse(plantDay?.pages[0]?.displays[0]?.states[0]?.argsJson ?? "{}")
      .spec as TemplateNode;
    const leaf = findNode(spec, "st_plant-day_phone-html-322-image");
    expect(leaf?.type).toBe("image");
    expect(leaf?.props.alt).toBe("");
    expect(leaf?.props.label).toBe("Monstera leaf · lobed silhouette");
    expect(String(leaf?.props.src)).toContain("data:image/svg+xml;base64,");

    const svg = Buffer.from(String(leaf?.props.src).split(",")[1] ?? "", "base64").toString();
    expect(svg).toContain('viewBox="0 0 220 260"');
    expect(svg).toContain('class="leaf"');
    expect(svg).toContain("C");
    expect(svg).toContain('fill="#d7ef66"');

    const { container } = renderWithProviders(
      <TemplateRenderer template={DEFAULT_TEMPLATE} spec={spec} mode="preview" />,
    );
    const rendered = container.querySelector<HTMLElement>(
      '[data-node-id="st_plant-day_phone-html-322-image"]',
    );
    expect(rendered).toHaveAttribute("data-node-type", "image");
    expect(rendered?.querySelector("img")).toHaveAttribute("alt", "");
    expect(rendered?.querySelector("img")).toHaveAttribute("src", leaf?.props.src);
    expect(rendered).toHaveStyle({ width: "272px", height: "321px" });
  });

  it("preserves the Run-before track illustration height inside its flex stack", () => {
    const seeds = JSON.parse(readFileSync(WORKSPACE_SEEDS_PATH, "utf8")) as {
      workspaces: Array<{
        id: string;
        pages: Array<{
          displays: Array<{ states: Array<{ argsJson: string }> }>;
        }>;
      }>;
    };
    const runBefore = seeds.workspaces.find(
      (workspace) => workspace.id === "ws_curated_eight_run-before",
    );
    const spec = JSON.parse(runBefore?.pages[0]?.displays[0]?.states[0]?.argsJson ?? "{}")
      .spec as TemplateNode;
    const track = findNode(spec, "st_run-before_phone-html-224-image");
    expect(track?.type).toBe("image");
    expect(JSON.parse(String(track?.props.css))).toMatchObject({ height: "252px" });

    const { container } = renderWithProviders(
      <TemplateRenderer template={DEFAULT_TEMPLATE} spec={spec} mode="preview" />,
    );

    const trackFigure = container.querySelector<HTMLElement>(
      '[data-node-id="st_run-before_phone-html-224-image"]',
    );
    expect(trackFigure).toHaveStyle({ height: "252px", flexShrink: "0" });
    expect(trackFigure?.querySelector("img")).toHaveAttribute("src", track?.props.src);
  });

  it("preserves authored roles and accessible names on custom layers", () => {
    const spec = node("page", "page", {}, [
      node("import-status", "text", {
        text: "Import complete",
        variant: "caption",
        tone: "muted",
        role: "status",
        ariaLabel: "HTML import status",
        tabIndex: 0,
      }),
      node("continue", "button", {
        text: "Continue",
        variant: "primary",
        size: "sm",
        pressed: true,
        role: "button",
        ariaLabel: "Continue to review",
        tabIndex: 0,
      }),
      node("availability", "meter", {
        value: 142,
        compact: true,
        label: "Storage available",
        tone: "red",
        role: "meter",
        tabIndex: 0,
      }),
    ]);

    const { container } = renderWithProviders(
      <TemplateRenderer template={DEFAULT_TEMPLATE} spec={spec} mode="preview" />,
    );

    const status = screen.getByRole("status", { name: "HTML import status" });
    expect(status).toHaveTextContent("Import complete");
    expect(status).toHaveAttribute("tabindex", "0");
    expect(status).toHaveClass("is-caption", "is-muted");

    const action = container.querySelector<HTMLElement>('[data-node-id="continue"]');
    expect(action).toHaveAttribute("role", "button");
    expect(action).toHaveAttribute("aria-label", "Continue to review");
    expect(action).toHaveAttribute("tabindex", "0");
    expect(container.querySelector(".tpl-control button")).toHaveAttribute("aria-pressed", "true");

    const meter = screen.getByRole("meter", { name: "Storage available" });
    expect(meter).toHaveAttribute("aria-valuemin", "0");
    expect(meter).toHaveAttribute("aria-valuemax", "100");
    expect(meter).toHaveAttribute("aria-valuenow", "100");
    expect(meter).toHaveAttribute("tabindex", "0");
    expect(meter.firstElementChild).toHaveStyle({ width: "100%" });
  });

  it("renders a legacy button placement fallback with the shared Button defaults", () => {
    const spec = node("page", "page", {}, [
      node("start-action", "slot", { minH: 44 }, [
        node("start-action-button", "button", {
          text: "Start a conversation",
          variant: "primary",
          shape: "pill",
          ariaLabel: "Start a conversation",
        }),
      ]),
    ]);

    const { container } = renderWithProviders(
      <TemplateRenderer template={DEFAULT_TEMPLATE} spec={spec} mode="preview" />,
    );

    const action = screen.getByRole("button", { name: "Start a conversation" });
    expect(action).toHaveAttribute("data-control-shape", "pill");
    expect(action).toHaveAttribute("data-control-variant", "primary");
    expect(container.querySelector('[data-node-id="start-action"]')).toHaveClass("is-fallback");
  });

  it("renders authored template content variants with their semantic and visual states", () => {
    const spec = node("page", "page", {}, [
      node("revenue-stat", "stat", {
        label: "Monthly revenue",
        value: "$1,240",
        detail: "8% above last month",
        tone: "red",
      }),
      node("revenue-metric", "metric", {
        label: "Revenue",
        prefix: "$",
        value: "1,240",
        frac: ".50",
        suffix: "/mo",
        detail: "Recurring",
        size: "lg",
        tone: "green",
      }),
      node("icon", "icon", { name: "check", size: 20, box: true, tone: "accent" }),
      node("avatar", "avatar", { name: "Ada Lovelace", size: "lg" }),
      node("chip", "chip", { text: "Verified", tone: "positive" }),
      node("full-meter", "meter", { label: "Battery", value: 63, tone: "green" }),
      node("empty-art", "image", {}),
      node("circle-swatch", "swatch", {
        tone: "accent",
        shape: "circle",
        h: 28,
        w: 80,
        grow: true,
        radius: 14,
      }),
    ]);

    const { container } = renderWithProviders(
      <TemplateRenderer template={DEFAULT_TEMPLATE} spec={spec} mode="preview" />,
    );

    const stat = container.querySelector<HTMLElement>(
      '[data-node-id="revenue-stat"] [data-rcl-stat]',
    );
    expect(stat).toHaveAttribute("data-provenance", "sample");
    expect(
      container.querySelector('[data-node-id="revenue-stat"] [data-rcl-stat-trend]'),
    ).toHaveAttribute("data-tone", "negative");

    const metric = container.querySelector<HTMLElement>('[data-node-id="revenue-metric"]');
    expect(metric).toHaveClass("is-lg", "is-green");
    expect(metric).toHaveTextContent("Revenue$1,240.50/moRecurring");

    expect(container.querySelector('[data-node-id="icon"]')).toHaveClass("is-boxed", "is-accent");
    expect(container.querySelector('[data-node-id="icon"] svg')).toHaveAttribute(
      "aria-hidden",
      "true",
    );
    expect(container.querySelector('[data-node-id="avatar"] [data-rcl-avatar]')).toHaveAttribute(
      "data-size",
      "lg",
    );
    expect(container.querySelector('[data-node-id="chip"]')).toHaveAttribute(
      "data-tone",
      "positive",
    );

    const meter = screen.getByRole("meter", { name: "Battery value" });
    expect(meter).toHaveAttribute("value", "63");
    expect(
      container.querySelector('[data-node-id="full-meter"] [data-rcl-bounded-meter]'),
    ).toHaveAttribute("data-tone", "success");

    expect(container.querySelector('[data-node-id="empty-art"]')).toHaveTextContent(
      strings.workbench.template.imageEmpty,
    );
    const swatch = container.querySelector<HTMLElement>('[data-node-id="circle-swatch"]');
    expect(swatch).toHaveAttribute("data-shape", "circle");
    expect(swatch).toHaveStyle({
      height: "28px",
      width: "80px",
      flexGrow: "1",
      borderRadius: "14px",
    });
  });

  it("uses explicit defaults for sparse layers and non-default layout choices", () => {
    const spec = node("page", "page", {}, [
      node("stack-defaults", "stack", {}, [node("stack-copy", "text", { text: "Stacked" })]),
      node(
        "row-options",
        "row",
        {
          wrap: true,
          border: true,
          gap: 18,
          pad: 6,
          align: "start",
          justify: "space-between",
          h: 52,
        },
        [node("row-copy", "text", { text: "Row" })],
      ),
      node("grid-defaults", "grid", {}, [node("grid-item", "text", { text: "Grid" })]),
      node("accent-surface", "surface", { bg: "accent", radius: 20, h: 72 }),
      node("raised-card", "card", { bg: "surfaceAlt", elevated: true, radius: 16 }),
      node("default-stat", "stat", {}),
      node("partial-metric", "metric", { value: "7" }),
      node("default-icon", "icon", {}),
      node("small-avatar", "avatar", { name: "Grace Hopper", size: "xs" }),
      node("plain-chip", "chip", { text: "Draft" }),
      node("warning-meter", "meter", { label: "Quota", value: 40, tone: "amber" }),
      node("neutral-meter", "meter", { label: "Capacity", value: 12 }),
      node("spaced-divider", "divider", { space: 12 }),
      node("default-spacer", "spacer", {}),
      node("line-swatch", "swatch", { tone: "line" }),
      node("secondary-swatch", "swatch", { tone: "secondary", shape: "square" }),
      node("remote-art", "image", { src: "/assets/reference.svg", fit: "contain", h: 96 }),
    ]);

    const { container } = renderWithProviders(
      <TemplateRenderer template={DEFAULT_TEMPLATE} spec={spec} mode="preview" />,
    );

    expect(container.querySelector('[data-node-id="stack-defaults"]')).toHaveStyle({ gap: "12px" });
    expect(container.querySelector('[data-node-id="stack-defaults"]')).not.toHaveStyle({
      padding: "0px",
    });
    expect(container.querySelector('[data-node-id="row-options"]')).toHaveClass(
      "is-wrap",
      "is-bordered",
    );
    expect(container.querySelector('[data-node-id="row-options"]')).toHaveStyle({
      gap: "18px",
      minHeight: "52px",
      justifyContent: "space-between",
    });
    expect(container.querySelector('[data-node-id="grid-defaults"]')).toHaveStyle({
      "--tpl-cols": "2",
      gap: "12px",
    });
    expect(container.querySelector('[data-node-id="accent-surface"]')).toHaveStyle({
      background: "var(--t-accent)",
      borderRadius: "20px",
      minHeight: "72px",
    });
    expect(container.querySelector('[data-node-id="raised-card"]')).toHaveStyle({
      background: "var(--t-surface-alt)",
      borderRadius: "16px",
    });
    expect(
      container.querySelector('[data-node-id="raised-card"]')?.getAttribute("style"),
    ).toContain("box-shadow");

    expect(
      container.querySelector('[data-node-id="default-stat"] [data-rcl-stat-value]'),
    ).toHaveTextContent("—");
    expect(
      container.querySelector('[data-node-id="default-stat"] [data-rcl-stat]'),
    ).toHaveAttribute("data-provenance", "sample");
    expect(container.querySelector('[data-node-id="partial-metric"]')).toHaveClass("is-md");
    expect(container.querySelector('[data-node-id="partial-metric"]')).toHaveTextContent("7");
    expect(container.querySelector('[data-node-id="default-icon"]')).toHaveStyle({
      width: "32px",
      height: "32px",
    });
    expect(
      container.querySelector('[data-node-id="small-avatar"] [data-rcl-avatar]'),
    ).toHaveAttribute("data-size", "xs");
    expect(container.querySelector('[data-node-id="plain-chip"]')).not.toHaveAttribute("data-tone");

    const quota = screen.getByRole("meter", { name: "Quota value" });
    const capacity = screen.getByRole("meter", { name: "Capacity value" });
    expect(quota).toHaveAttribute("value", "40");
    expect(
      container.querySelector('[data-node-id="warning-meter"] [data-rcl-bounded-meter]'),
    ).toHaveAttribute("data-tone", "warning");
    expect(
      container.querySelector('[data-node-id="neutral-meter"] [data-rcl-bounded-meter]'),
    ).toHaveAttribute("data-tone", "neutral");
    expect(capacity).toHaveAttribute("value", "12");

    expect(container.querySelector('[data-node-id="spaced-divider"]')).toHaveStyle({
      marginBlock: "12px",
    });
    expect(container.querySelector('[data-node-id="default-spacer"]')).toHaveAttribute(
      "aria-hidden",
      "true",
    );
    expect(container.querySelector('[data-node-id="default-spacer"]')).toHaveStyle({
      height: "16px",
    });
    expect(container.querySelector('[data-node-id="line-swatch"]')).toHaveStyle({
      backgroundColor: "var(--t-line)",
      height: "48px",
    });
    expect(
      container.querySelector('[data-node-id="secondary-swatch"] [class*="tpl-art-shape"]'),
    ).toHaveClass("is-square");
    const remoteImage = container.querySelector<HTMLImageElement>(
      '[data-node-id="remote-art"] img',
    );
    expect(remoteImage?.getAttribute("src")).toContain("/assets/reference.svg");
    expect(remoteImage).toHaveStyle({ objectFit: "contain" });
  });

  it("keeps authored art direction when a layer uses the shared RCL Card", () => {
    const spec = node("page", "page", {}, [
      node("art-directed-card", "card", {
        css: JSON.stringify({
          padding: "32px",
          background: "#f3e8d8",
          borderRadius: "28px",
          boxShadow: "0 6px 12px rgba(30, 20, 10, 0.18)",
        }),
      }),
    ]);

    const { container } = renderWithProviders(
      <TemplateRenderer template={DEFAULT_TEMPLATE} spec={spec} mode="preview" />,
    );

    const card = container.querySelector<HTMLElement>('[data-node-id="art-directed-card"]');
    expect(card).toHaveClass("tpl-card");
    expect(card).toHaveStyle({ padding: "32px", background: "#f3e8d8" });
    expect(card?.style.borderRadius).toBe("28px");
    expect(card?.style.boxShadow).toContain("0 6px 12px");
  });

  it("filters imported styles before applying them to controls and layout", () => {
    const spec = node("page", "page", {}, [
      node("styled-button", "button", {
        text: "Save",
        css: JSON.stringify({
          "background-color": "#123456",
          height: "36px",
          "font-size": 14,
          "--imported-accent": "violet",
          position: "fixed",
          "background-image": "url(https://example.test/pixel)",
          "border-color": "red; color: blue",
          "<bad-name>": "ignored",
          "too-long": "x".repeat(161),
        }),
      }),
      node("styled-input", "input", {
        placeholder: "Search records",
        css: JSON.stringify({ display: "inline-flex", gap: 8, color: "#222", zIndex: 9 }),
      }),
      node("malformed-style", "text", { text: "Safe", css: "{not-json" }),
      node("array-style", "text", { text: "Still safe", css: "[1,2,3]" }),
      node("large-style", "text", { text: "Bounded", css: "x".repeat(12_001) }),
    ]);

    const { container } = renderWithProviders(
      <TemplateRenderer template={DEFAULT_TEMPLATE} spec={spec} mode="preview" />,
    );

    const button = container.querySelector<HTMLElement>('[data-node-id="styled-button"]');
    expect(button).toHaveStyle({ backgroundColor: "rgb(18, 52, 86)" });
    expect(button).toHaveStyle({ position: "fixed" });
    expect(button?.style.backgroundImage).toBe("");
    const nativeButton = button?.querySelector("button");
    expect(nativeButton?.style.position).toBe("");
    expect(nativeButton).toHaveStyle({ minHeight: "36px", backgroundColor: "rgb(18, 52, 86)" });

    const input = screen.getByRole("textbox", { name: "Search records" });
    expect(input).toHaveStyle({ display: "inline-flex", gap: "8px" });
    expect(input).not.toHaveStyle({ zIndex: "9" });
    for (const id of ["malformed-style", "array-style", "large-style"]) {
      expect(container.querySelector(`[data-node-id="${id}"]`)).not.toHaveStyle({
        color: "red",
        background: "blue",
      });
    }
  });

  it("keeps image and unknown-node fallbacks visible for incomplete imported data", () => {
    const importedUnknown = node("legacy-layer", "future-chart" as never, {
      text: "Imported chart details",
    });
    const spec = node("page", "page", {}, [
      node("remote-image", "image", { src: "https://example.test/chart.png", alt: "Chart" }),
      node("empty-image", "image", { src: "", fit: "stretch" }),
      importedUnknown,
    ]);

    const { container } = renderWithProviders(
      <TemplateRenderer template={DEFAULT_TEMPLATE} spec={spec} mode="preview" />,
    );

    const image = container.querySelector<HTMLImageElement>('[data-node-id="remote-image"] img');
    expect(image).toHaveAttribute("src", "https://example.test/chart.png");
    expect(image).toHaveAttribute("alt", "Chart");
    expect(image).toHaveAttribute("loading", "lazy");
    expect(image).toHaveStyle({ objectFit: "cover" });
    expect(container.querySelector('[data-node-id="empty-image"]')).toHaveTextContent(
      strings.workbench.template.imageEmpty,
    );
    expect(screen.getByRole("note")).toHaveTextContent(strings.workbench.template.unknownLayer);
    expect(screen.getByRole("note")).toHaveTextContent("Imported chart details");
  });
});
