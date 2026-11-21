import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { DEFAULT_TEMPLATE } from "./library";

const fetchSpy = vi.fn<(input: RequestInfo | URL, init?: RequestInit) => Promise<Response>>();

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  });
}

function methodOf(init: RequestInit | undefined): string {
  return (init?.method ?? "GET").toUpperCase();
}

function requestUrl(input: RequestInfo | URL): string {
  if (typeof input === "string") return input;
  if (input instanceof URL) return input.href;
  return input.url;
}

/** A server that answers every route successfully; individual tests override. */
function healthyApi(overrides: (url: string, init: RequestInit) => Response | undefined) {
  return (input: RequestInfo | URL, init: RequestInit = {}): Promise<Response> => {
    const override = overrides(requestUrl(input), init);
    if (override) return Promise.resolve(override);
    const method = methodOf(init);
    if (method === "GET") return Promise.resolve(jsonResponse({ templates: [] }));
    if (method === "DELETE") return Promise.resolve(new Response(null, { status: 204 }));
    const rawBody = typeof init.body === "string" ? init.body : "{}";
    return Promise.resolve(
      jsonResponse(
        { template: JSON.parse(rawBody) },
        method === "POST" ? 201 : 200,
      ),
    );
  };
}

async function loadModule() {
  // A fresh module instance per test so module-level fallback/queue state and
  // the one-time migration guard behave exactly as on a page load.
  return import("./local");
}

beforeEach(() => {
  vi.resetModules();
  window.localStorage.clear();
  fetchSpy.mockReset();
  fetchSpy.mockImplementation(healthyApi(() => undefined));
  vi.stubGlobal("fetch", fetchSpy);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("browser-local templates", () => {
  it("starts empty and resolves only the built-in shelf", async () => {
    const local = await loadModule();
    expect(local.readLocalTemplates()).toEqual([]);
    expect(local.resolveTemplate(DEFAULT_TEMPLATE.id)?.name).toBe(DEFAULT_TEMPLATE.name);
    expect(local.resolveTemplate("missing")).toBeUndefined();
  });

  it("duplicates a template with fresh ids and keeps the composition", async () => {
    const local = await loadModule();
    const copy = local.cloneTemplate(DEFAULT_TEMPLATE, "Aperture study");
    expect(copy.local).toBe(true);
    expect(copy.name).toBe("Aperture study");
    expect(copy.spec.id).not.toBe(DEFAULT_TEMPLATE.spec.id);
    expect(JSON.stringify(copy.spec)).not.toBe(JSON.stringify(DEFAULT_TEMPLATE.spec));
    expect(copy.palette).toEqual(DEFAULT_TEMPLATE.palette);
  });

  it("creates a blank template with a page, stack and slot", async () => {
    const local = await loadModule();
    const blank = local.blankTemplate("Test page");
    expect(blank.name).toBe("Test page");
    expect(blank.local).toBe(true);
    const types: string[] = [];
    const visit = (current: {
      type: string;
      children?: Array<{ type: string; children?: unknown[] }>;
    }): void => {
      types.push(current.type);
      for (const child of (current.children ?? []) as Array<Parameters<typeof visit>[0]>) visit(child);
    };
    visit(blank.spec);
    expect(types).toContain("page");
    expect(types).toContain("slot");
  });

  it("persists, replaces and removes local templates", async () => {
    const local = await loadModule();
    const copy = local.cloneTemplate(DEFAULT_TEMPLATE, "Stored study");
    local.upsertLocalTemplate(copy);
    expect(local.readLocalTemplates()).toHaveLength(1);

    local.upsertLocalTemplate({ ...copy, name: "Renamed study" });
    const stored = local.readLocalTemplates();
    expect(stored).toHaveLength(1);
    expect(stored[0]?.name).toBe("Renamed study");

    expect(local.templateShelf(stored)).toHaveLength(local.templateShelf().length + 1);
    expect(local.resolveTemplate(copy.id, stored)?.name).toBe("Renamed study");

    local.removeLocalTemplate(copy.id);
    expect(local.readLocalTemplates()).toEqual([]);
    expect(window.localStorage.getItem(local.LOCAL_TEMPLATE_KEY)).toBe("[]");

    await local.flushLocalTemplateWrites();
    expect(local.readLocalTemplates()).toEqual([]);
  });

  it("ignores malformed storage instead of crashing the studio", async () => {
    const local = await loadModule();
    window.localStorage.setItem(local.LOCAL_TEMPLATE_KEY, "{not json");
    expect(local.readLocalTemplates()).toEqual([]);
    window.localStorage.setItem(local.LOCAL_TEMPLATE_KEY, JSON.stringify([{ id: 1 }, null]));
    expect(local.readLocalTemplates()).toEqual([]);
  });

  it("builds a stable slug and applies a palette immutably", async () => {
    const local = await loadModule();
    expect(local.localTemplateId("Aperture Study")).toMatch(/^aperture-study-[a-z0-9]+$/);
    const palette = { ...DEFAULT_TEMPLATE.palette, accent: "#123456" };
    const themed = local.withPalette(DEFAULT_TEMPLATE, palette);
    expect(themed.palette.accent).toBe("#123456");
    expect(DEFAULT_TEMPLATE.palette.accent).not.toBe("#123456");
  });
});

describe("API-backed shelf", () => {
  it("lets the server win for an id present in both", async () => {
    const local = await loadModule();
    const cached = local.cloneTemplate(DEFAULT_TEMPLATE, "Cached name");
    window.localStorage.setItem(local.LOCAL_TEMPLATE_KEY, JSON.stringify([cached]));

    const server = { ...cached, name: "Server name", tagline: "From the API" };
    fetchSpy.mockImplementation(
      healthyApi((_url, init) =>
        methodOf(init) === "GET" ? jsonResponse({ templates: [server] }) : undefined,
      ),
    );

    await local.refreshLocalTemplates();

    const stored = local.readLocalTemplates();
    expect(stored).toHaveLength(1);
    expect(stored[0]?.id).toBe(cached.id);
    expect(stored[0]?.name).toBe("Server name");
    expect(stored[0]?.tagline).toBe("From the API");
    expect(stored[0]?.local).toBe(true);
    expect(local.templatesUsingFallback()).toBe(false);
  });

  it("uploads cache-only templates exactly once", async () => {
    const local = await loadModule();
    const first = local.cloneTemplate(DEFAULT_TEMPLATE, "First draft");
    const second = local.blankTemplate("Second draft");
    window.localStorage.setItem(local.LOCAL_TEMPLATE_KEY, JSON.stringify([first, second]));

    await local.refreshLocalTemplates();
    const postsAfterFirst = fetchSpy.mock.calls.filter((call) => methodOf(call[1]) === "POST");
    expect(postsAfterFirst).toHaveLength(2);
    expect(
      postsAfterFirst
        .map((call) => {
          const body = call[1]?.body;
          return JSON.parse(typeof body === "string" ? body : "{}").id as string;
        })
        .sort(),
    ).toEqual([first.id, second.id].sort());

    await local.refreshLocalTemplates();
    const postsAfterSecond = fetchSpy.mock.calls.filter((call) => methodOf(call[1]) === "POST");
    expect(postsAfterSecond).toHaveLength(2);
    expect(local.templatesUsingFallback()).toBe(false);
  });

  it("creates a new template at the collection route, not under its id", async () => {
    const local = await loadModule();
    const draft = local.blankTemplate("Route check");
    local.upsertLocalTemplate(draft);
    await local.flushLocalTemplateWrites();

    const post = fetchSpy.mock.calls.find((call) => methodOf(call[1]) === "POST");
    expect(post).toBeTruthy();
    expect(requestUrl(post?.[0] ?? "")).toMatch(/\/api\/templates$/);
    expect(fetchSpy.mock.calls.some((call) => methodOf(call[1]) === "PUT")).toBe(false);
  });

  it("updates a template that the server already lists under its id", async () => {
    const local = await loadModule();
    const draft = local.blankTemplate("Known draft");
    fetchSpy.mockImplementation(
      healthyApi((_url, init) =>
        methodOf(init) === "GET"
          ? jsonResponse({ templates: [{ ...draft, name: "Known draft" }] })
          : undefined,
      ),
    );
    await local.refreshLocalTemplates();

    local.upsertLocalTemplate({ ...draft, name: "Renamed draft" });
    await local.flushLocalTemplateWrites();

    const put = fetchSpy.mock.calls.find((call) => methodOf(call[1]) === "PUT");
    expect(put).toBeTruthy();
    expect(requestUrl(put?.[0] ?? "")).toMatch(new RegExp(`/api/templates/${draft.id}$`));
  });

  it("keeps the cache entry and emits an error event when an upload fails", async () => {
    const local = await loadModule();
    const draft = local.blankTemplate("Offline draft");
    window.localStorage.setItem(local.LOCAL_TEMPLATE_KEY, JSON.stringify([draft]));

    fetchSpy.mockImplementation(
      healthyApi((_url, init) =>
        methodOf(init) === "POST"
          ? jsonResponse({ error: { code: "internal", message: "storage down" } }, 500)
          : undefined,
      ),
    );

    const messages: string[] = [];
    const listener = (event: Event): void => {
      const detail = (event as CustomEvent<{ message: string } | undefined>).detail;
      messages.push(detail?.message ?? "");
    };
    window.addEventListener(local.TEMPLATE_ERROR_EVENT, listener);
    try {
      await local.refreshLocalTemplates();
    } finally {
      window.removeEventListener(local.TEMPLATE_ERROR_EVENT, listener);
    }

    expect(local.readLocalTemplates()).toHaveLength(1);
    expect(local.readLocalTemplates()[0]?.name).toBe("Offline draft");
    expect(messages).toHaveLength(1);
    expect(messages[0]).toContain("Could not save template");
    expect(messages[0]?.length).toBeLessThanOrEqual(200);

    // The failed template is pending, and the next successful list retries it.
    fetchSpy.mockImplementation(healthyApi(() => undefined));
    await local.refreshLocalTemplates();
    expect(fetchSpy.mock.calls.filter((call) => methodOf(call[1]) === "POST")).toHaveLength(2);
  });

  it("restores the entry when DELETE fails", async () => {
    const local = await loadModule();
    const draft = local.blankTemplate("Doomed draft");
    window.localStorage.setItem(local.LOCAL_TEMPLATE_KEY, JSON.stringify([draft]));

    fetchSpy.mockImplementation(
      healthyApi((_url, init) => {
        if (methodOf(init) === "GET") return jsonResponse({ templates: [draft] });
        if (methodOf(init) === "DELETE") {
          return jsonResponse({ error: { code: "internal", message: "storage down" } }, 500);
        }
        return undefined;
      }),
    );

    await local.refreshLocalTemplates();
    expect(local.readLocalTemplates()).toHaveLength(1);

    local.removeLocalTemplate(draft.id);
    expect(local.readLocalTemplates()).toEqual([]);

    await local.flushLocalTemplateWrites();
    const restored = local.readLocalTemplates();
    expect(restored).toHaveLength(1);
    expect(restored[0]?.id).toBe(draft.id);
  });

  it("keeps reading the cache and reports fallback when the API is absent", async () => {
    const local = await loadModule();
    const draft = local.blankTemplate("Local only");
    window.localStorage.setItem(local.LOCAL_TEMPLATE_KEY, JSON.stringify([draft]));
    fetchSpy.mockRejectedValue(new Error("offline"));

    await local.refreshLocalTemplates();

    expect(local.templatesUsingFallback()).toBe(true);
    const stored = local.readLocalTemplates();
    expect(stored).toHaveLength(1);
    expect(stored[0]?.name).toBe("Local only");

    // Writes still land in the cache and are honestly marked pending.
    local.upsertLocalTemplate({ ...draft, name: "Edited offline" });
    expect(local.readLocalTemplates()[0]?.name).toBe("Edited offline");
    await local.flushLocalTemplateWrites();
    expect(local.templatesUsingFallback()).toBe(true);
    expect(local.readLocalTemplates()[0]?.name).toBe("Edited offline");
  });
});
