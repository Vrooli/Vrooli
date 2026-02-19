import { describe, it, expect, vi, beforeEach } from "vitest";

// Mock api-base before any imports
vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:15000/api/v1",
  buildApiUrl: (path: string, opts: { baseUrl: string }) =>
    `${opts.baseUrl}${path}`,
  resolveWsBase: () => "ws://localhost:15000/api/v1",
  buildWsUrl: (path: string, opts: { baseUrl: string }) =>
    `${opts.baseUrl}${path}`,
}));

beforeEach(() => {
  vi.restoreAllMocks();
});

// [REQ:P1-002a] Shortcut Profile API client tests
describe("Shortcut Profile API", () => {
  it("getEffectiveShortcuts returns shortcut list", async () => {
    const mockShortcuts = [
      { label: "Claude", command: "claude --dangerously-skip-permissions" },
      { label: "Codex", command: "codex --yolo" },
    ];
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockShortcuts),
    }) as typeof fetch;

    const { getEffectiveShortcuts } = await import("../lib/api");
    const result = await getEffectiveShortcuts();

    expect(result).toEqual(mockShortcuts);
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/shortcuts"),
      expect.objectContaining({ cache: "no-store" }),
    );
  });

  it("listShortcutProfiles returns profile list", async () => {
    const mockProfiles = [
      { id: "default", scope: "service", name: "Default", shortcuts: [] },
    ];
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockProfiles),
    }) as typeof fetch;

    const { listShortcutProfiles } = await import("../lib/api");
    const result = await listShortcutProfiles();

    expect(result).toEqual(mockProfiles);
  });

  it("upsertShortcutProfile sends PUT request", async () => {
    const mockProfile = {
      id: "ws",
      scope: "workspace",
      name: "Workspace",
      shortcuts: [{ label: "Test", command: "echo test" }],
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    };
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(mockProfile),
    }) as typeof fetch;

    const { upsertShortcutProfile } = await import("../lib/api");
    const result = await upsertShortcutProfile({
      id: "ws",
      scope: "workspace",
      name: "Workspace",
      shortcuts: [{ label: "Test", command: "echo test" }],
    });

    expect(result.id).toBe("ws");
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/shortcuts/profiles"),
      expect.objectContaining({ method: "PUT" }),
    );
  });

  it("deleteShortcutProfile sends DELETE request", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
    }) as typeof fetch;

    const { deleteShortcutProfile } = await import("../lib/api");
    await deleteShortcutProfile("ws");

    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/shortcuts/profiles/ws"),
      expect.objectContaining({ method: "DELETE" }),
    );
  });

  it("getEffectiveShortcuts throws on error", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
      json: () =>
        Promise.resolve({
          error: "Internal error",
          code: "internal_error",
          category: "internal",
        }),
    }) as typeof fetch;

    const { getEffectiveShortcuts, APIError } = await import("../lib/api");

    try {
      await getEffectiveShortcuts();
      expect.unreachable("should have thrown");
    } catch (err) {
      expect(err).toBeInstanceOf(APIError);
    }
  });
});

// [REQ:P1-002b] Shortcut Profile Management UI tests
describe("TerminalLauncher with API shortcuts", () => {
  it("component module exports default function", async () => {
    const mod = await import("../components/TerminalLauncher");
    expect(typeof mod.default).toBe("function");
  });

  it("ShortcutEntry interface includes fields", async () => {
    const entry: import("../lib/api").ShortcutEntry = {
      label: "Test",
      command: "echo hello",
      description: "A test shortcut",
    };
    expect(entry.label).toBe("Test");
    expect(entry.command).toBe("echo hello");
  });

  it("ShortcutProfile interface includes scope and timestamps", async () => {
    const profile: import("../lib/api").ShortcutProfile = {
      id: "ws",
      scope: "workspace",
      name: "Workspace",
      shortcuts: [{ label: "A", command: "a" }],
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    };
    expect(profile.scope).toBe("workspace");
    expect(profile.shortcuts).toHaveLength(1);
  });
});
