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
    expect(seed.workspaces.length).toBe(8);
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

  it("uses unique layer ids across every display in a workspace", () => {
    for (const workspace of seed.workspaces) {
      const ids = new Set<string>();
      for (const page of workspace.pages) {
        for (const display of page.displays) {
          for (const state of display.states) {
            const parsed = JSON.parse(state.argsJson) as { spec: TemplateNode };
            walk(parsed.spec, (node) => {
              expect(
                ids.has(node.id),
                `${workspace.id}: layer id ${node.id} is reused`,
              ).toBe(false);
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
