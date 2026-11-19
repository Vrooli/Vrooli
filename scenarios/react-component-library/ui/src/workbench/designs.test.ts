import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { API_BASE } from "../api/client";
import type { DesignDocument } from "./designs";
import type { TemplatePalette } from "./templates/spec";

const THEME = {
  bg: "#000",
  surface: "#111",
  surfaceAlt: "#222",
  ink: "#fff",
  muted: "#aaa",
  line: "#333",
  accent: "#f80",
  accentInk: "#000",
  accentSoft: "#421",
  secondary: "#0af",
} as const satisfies TemplatePalette;

const fetchSpy = vi.fn();

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  });
}

function baseDoc(overrides: Partial<DesignDocument> = {}): DesignDocument {
  return {
    id: "design-1",
    name: "Aperture",
    updatedAt: "2026-11-19T12:00:00.000Z",
    revision: 1,
    frames: [],
    theme: THEME,
    scenario: null,
    page: null,
    ...overrides,
  };
}

async function loadModule() {
  // Each test gets a fresh module instance so the module-level fallback flag
  // starts false, exactly as it would on a page load.
  return import("./designs");
}

beforeEach(() => {
  vi.resetModules();
  window.localStorage.clear();
  fetchSpy.mockReset();
  vi.stubGlobal("fetch", fetchSpy);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("saveDesign", () => {
  it("sends the required document fields on PUT", async () => {
    const designs = await loadModule();
    fetchSpy.mockResolvedValueOnce(jsonResponse({ design: baseDoc({ revision: 2 }) }));

    await designs.saveDesign(baseDoc());

    expect(fetchSpy).toHaveBeenCalledTimes(1);
    const [url, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
    expect(url).toContain("/api/designs/design-1");
    expect(init.method).toBe("PUT");
    const body = JSON.parse(init.body as string) as Record<string, unknown>;
    expect(body).toMatchObject({
      name: "Aperture",
      frames: [],
      theme: THEME,
      options: {},
      scenario: null,
      page: null,
      revision: 1,
    });
  });

  it("sends the design's chrome on PUT", async () => {
    const designs = await loadModule();
    fetchSpy.mockResolvedValueOnce(jsonResponse({ design: baseDoc({ revision: 2 }) }));

    await designs.saveDesign(baseDoc({ options: { radius: 12, display: "serif" } }));

    const [, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
    const body = JSON.parse(init.body as string) as Record<string, unknown>;
    expect(body.options).toEqual({ radius: 12, display: "serif" });
  });

  it("surfaces a 400 instead of silently resurrecting a local draft", async () => {
    const designs = await loadModule();
    fetchSpy.mockResolvedValueOnce(
      jsonResponse({ error: { code: "invalid_argument", message: "frames is required" } }, 400),
    );

    await expect(designs.saveDesign(baseDoc())).rejects.toThrow("frames is required");
    expect(designs.designsUsingFallback()).toBe(false);
    expect(window.localStorage.getItem("rcl.designs.v1")).toBeNull();
  });

  it("throws DesignConflictError with the current document on 409", async () => {
    const designs = await loadModule();
    const current = baseDoc({ name: "Aperture (other tab)", revision: 2 });
    fetchSpy.mockResolvedValueOnce(
      jsonResponse(
        { error: { code: "conflict", message: "design was modified elsewhere" }, design: current },
        409,
      ),
    );

    const error = await designs.saveDesign(baseDoc()).catch((caught: unknown) => caught);

    expect(error).toBeInstanceOf(designs.DesignConflictError);
    expect((error as InstanceType<typeof designs.DesignConflictError>).current).toEqual(current);
  });

  it("reports a write that only reached this browser", async () => {
    const designs = await loadModule();
    fetchSpy.mockRejectedValueOnce(new TypeError("Failed to fetch"));

    const result = await designs.putDesign(baseDoc());

    expect(result.local).toBe(true);
    expect(result.design.id).toBe("design-1");
    expect(designs.designsUsingFallback()).toBe(true);
  });

  it("names a 404 as a design that is not on the server", async () => {
    const designs = await loadModule();
    fetchSpy.mockResolvedValueOnce(new Response("gone", { status: 404 }));
    await expect(designs.putDesign(baseDoc())).rejects.toThrow(/not on the server/);
    expect(designs.designsUsingFallback()).toBe(false);
  });

  it("does not pretend a 5xx save is a local draft", async () => {
    const designs = await loadModule();
    fetchSpy.mockResolvedValueOnce(new Response("down", { status: 503 }));

    await expect(designs.putDesign(baseDoc())).rejects.toThrow();
    expect(designs.designsUsingFallback()).toBe(false);
    expect(window.localStorage.getItem("rcl.designs.v1")).toBeNull();
  });
});

describe("createDesign", () => {
  it("sends options on POST", async () => {
    const designs = await loadModule();
    fetchSpy.mockResolvedValueOnce(jsonResponse({ design: baseDoc() }, 201));

    await designs.createDesign({
      name: "Aperture",
      frames: [],
      theme: THEME,
      options: { radius: 20, display: "sans" },
    });

    const [url, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
    expect(url).toContain("/api/designs");
    expect(init.method).toBe("POST");
    const body = JSON.parse(init.body as string) as Record<string, unknown>;
    expect(body.options).toEqual({ radius: 20, display: "sans" });
  });

  it("defaults absent options to an empty object", async () => {
    const designs = await loadModule();
    fetchSpy.mockResolvedValueOnce(jsonResponse({ design: baseDoc() }, 201));

    await designs.createDesign({ name: "Aperture", frames: [], theme: THEME });

    const [, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
    const body = JSON.parse(init.body as string) as Record<string, unknown>;
    expect(body.options).toEqual({});
  });

  it("keeps chrome on a local-only create", async () => {
    const designs = await loadModule();
    fetchSpy.mockRejectedValueOnce(new TypeError("Failed to fetch"));

    const design = await designs.createDesign({
      name: "Aperture",
      frames: [],
      theme: THEME,
      options: { display: "serif" },
    });

    expect(design.options).toEqual({ display: "serif" });
  });
});

describe("fallback honesty", () => {
  it("falls back to localStorage on a network error and reports it", async () => {
    const designs = await loadModule();
    fetchSpy.mockRejectedValueOnce(new TypeError("Failed to fetch"));

    await expect(designs.listDesigns()).resolves.toEqual([]);
    expect(designs.designsUsingFallback()).toBe(true);
  });

  it("falls back when the collection route is missing", async () => {
    const designs = await loadModule();
    fetchSpy.mockResolvedValueOnce(new Response("not found", { status: 404 }));

    await expect(designs.listDesigns()).resolves.toEqual([]);
    expect(designs.designsUsingFallback()).toBe(true);
  });

  it("does not fall back on a 5xx collection response", async () => {
    const designs = await loadModule();
    fetchSpy.mockResolvedValueOnce(new Response("down", { status: 503 }));

    await expect(designs.listDesigns()).rejects.toThrow("503");
    expect(designs.designsUsingFallback()).toBe(false);
  });

  it("does not fall back when a member route returns 404", async () => {
    const designs = await loadModule();
    fetchSpy.mockResolvedValueOnce(new Response("missing", { status: 404 }));

    await expect(designs.getDesign("design-1")).rejects.toThrow("404");
    expect(designs.designsUsingFallback()).toBe(false);
  });
});

describe("designAssetUrl", () => {
  it("builds an absolute URL against the API base", async () => {
    const designs = await loadModule();
    const asset = {
      id: "asset-1",
      designId: "design-1",
      name: "hero.png",
      mime: "image/png",
      bytes: 120,
      width: 10,
      height: 10,
      fidelity: "reference" as const,
      addedAt: "2026-11-19T12:00:00.000Z",
      url: "/api/designs/design-1/assets/asset-1",
    };

    expect(designs.designAssetUrl(asset)).toBe(
      `${API_BASE.replace(/\/$/, "")}/api/designs/design-1/assets/asset-1`,
    );
  });

  it("leaves an already absolute URL untouched", async () => {
    const designs = await loadModule();
    const asset = {
      id: "asset-1",
      designId: "design-1",
      name: "hero.png",
      mime: "image/png",
      bytes: 120,
      width: 10,
      height: 10,
      fidelity: "captured" as const,
      addedAt: "2026-11-19T12:00:00.000Z",
      url: "https://cdn.example.com/hero.png",
    };

    expect(designs.designAssetUrl(asset)).toBe("https://cdn.example.com/hero.png");
  });
});
