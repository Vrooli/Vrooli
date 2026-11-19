import { beforeEach, describe, expect, it, vi } from "vitest";

import { DEFAULT_TEMPLATE } from "./library";
import {
  LOCAL_TEMPLATE_KEY,
  blankTemplate,
  cloneTemplate,
  localTemplateId,
  readLocalTemplates,
  removeLocalTemplate,
  resolveTemplate,
  templateShelf,
  upsertLocalTemplate,
  withPalette,
} from "./local";

describe("browser-local templates", () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.restoreAllMocks();
  });

  it("starts empty and resolves only the built-in shelf", () => {
    expect(readLocalTemplates()).toEqual([]);
    expect(resolveTemplate(DEFAULT_TEMPLATE.id)?.name).toBe(DEFAULT_TEMPLATE.name);
    expect(resolveTemplate("missing")).toBeUndefined();
  });

  it("duplicates a template with fresh ids and keeps the composition", () => {
    const copy = cloneTemplate(DEFAULT_TEMPLATE, "Aperture study");
    expect(copy.local).toBe(true);
    expect(copy.name).toBe("Aperture study");
    expect(copy.spec.id).not.toBe(DEFAULT_TEMPLATE.spec.id);
    expect(JSON.stringify(copy.spec)).not.toBe(JSON.stringify(DEFAULT_TEMPLATE.spec));
    expect(copy.palette).toEqual(DEFAULT_TEMPLATE.palette);
  });

  it("creates a blank template with a page, stack and slot", () => {
    const blank = blankTemplate("Test page");
    expect(blank.name).toBe("Test page");
    expect(blank.local).toBe(true);
    const types: string[] = [];
    const visit = (current: { type: string; children?: Array<{ type: string; children?: unknown[] }> }): void => {
      types.push(current.type);
      for (const child of (current.children ?? []) as Array<Parameters<typeof visit>[0]>) visit(child);
    };
    visit(blank.spec);
    expect(types).toContain("page");
    expect(types).toContain("slot");
  });

  it("persists, replaces and removes local templates", () => {
    const copy = cloneTemplate(DEFAULT_TEMPLATE, "Stored study");
    upsertLocalTemplate(copy);
    expect(readLocalTemplates()).toHaveLength(1);

    upsertLocalTemplate({ ...copy, name: "Renamed study" });
    const stored = readLocalTemplates();
    expect(stored).toHaveLength(1);
    expect(stored[0]?.name).toBe("Renamed study");

    expect(templateShelf(stored)).toHaveLength(templateShelf().length + 1);
    expect(resolveTemplate(copy.id, stored)?.name).toBe("Renamed study");

    removeLocalTemplate(copy.id);
    expect(readLocalTemplates()).toEqual([]);
    expect(window.localStorage.getItem(LOCAL_TEMPLATE_KEY)).toBe("[]");
  });

  it("ignores malformed storage instead of crashing the studio", () => {
    window.localStorage.setItem(LOCAL_TEMPLATE_KEY, "{not json");
    expect(readLocalTemplates()).toEqual([]);
    window.localStorage.setItem(LOCAL_TEMPLATE_KEY, JSON.stringify([{ id: 1 }, null]));
    expect(readLocalTemplates()).toEqual([]);
  });

  it("builds a stable slug and applies a palette immutably", () => {
    expect(localTemplateId("Aperture Study")).toMatch(/^aperture-study-[a-z0-9]+$/);
    const palette = { ...DEFAULT_TEMPLATE.palette, accent: "#123456" };
    const themed = withPalette(DEFAULT_TEMPLATE, palette);
    expect(themed.palette.accent).toBe("#123456");
    expect(DEFAULT_TEMPLATE.palette.accent).not.toBe("#123456");
  });
});
