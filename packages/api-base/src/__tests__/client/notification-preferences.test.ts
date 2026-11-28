import { describe, expect, it, vi } from "vitest";
import {
  createNotificationPreferencesClient,
  notificationPreferenceControlGroups,
  NotificationPreferencesError,
  NotificationPreferencesUnavailableError,
} from "../../client/notification-preferences.js";

const catalog = {
  sources: [
    { source_scenario: "web-console", display_name: "Web Console", icon: "", deep_link_base: "", declared: true, categories: [] },
    { source_scenario: "other", display_name: "Other", icon: "", deep_link_base: "", declared: true, categories: [] },
  ],
  preferences: [
    { source_scenario: "web-console", category: "", scope_key: "", enabled: true, min_urgency: "normal", channel_override: "" },
    { source_scenario: "other", category: "", scope_key: "", enabled: true, min_urgency: "normal", channel_override: "" },
  ],
};

describe("shared notification preference client", () => {
  it("asks for each requested source and filters an older unscoped response", async () => {
    const request = vi.fn(async () => Response.json(catalog));
    const client = createNotificationPreferencesClient(request);
    const result = await client.getCatalog(["web-console"]);

    expect(request.mock.calls[0]?.[0]).toBe("/notifications/catalog?source_scenario=web-console");
    expect(result.sources.map((source) => source.source_scenario)).toEqual(["web-console"]);
    expect(result.preferences.map((pref) => pref.source_scenario)).toEqual(["web-console"]);
  });

  it("keeps auth ownership with the injected request and uses typed update bodies", async () => {
    const request = vi.fn(async (_path: string, init?: RequestInit) => {
      const headers = new Headers(init?.headers);
      headers.set("Authorization", "Bearer owner");
      expect(headers.get("Authorization")).toBe("Bearer owner");
      return Response.json({ ...catalog.preferences[0], enabled: false });
    });
    const client = createNotificationPreferencesClient(request);
    const result = await client.updatePreference(catalog.preferences[0]!);
    const init = request.mock.calls[0]?.[1];

    expect(init?.method).toBe("PUT");
    expect(JSON.parse(String(init?.body))).toEqual(catalog.preferences[0]);
    expect(result.enabled).toBe(false);
  });

  it("allows a server-pinned facade to supply the preference source", async () => {
    const request = vi.fn(async (_path: string, init?: RequestInit) => {
      expect(JSON.parse(String(init?.body))).not.toHaveProperty("source_scenario");
      return Response.json(catalog.preferences[0]);
    });
    const client = createNotificationPreferencesClient(request);

    await expect(client.updatePreference({
      category: "",
      enabled: true,
      min_urgency: "normal",
      channel_override: "",
    })).resolves.toMatchObject({ source_scenario: "web-console", scope_key: "" });
  });

  it("keeps unavailable and malformed-response states distinct from an empty catalog", async () => {
    const unavailable = createNotificationPreferencesClient(async () => new Response("down", { status: 503 }));
    await expect(unavailable.getCatalog()).rejects.toBeInstanceOf(NotificationPreferencesUnavailableError);

    const malformed = createNotificationPreferencesClient(async () => Response.json({ sources: "not-an-array", preferences: [] }));
    await expect(malformed.getCatalog()).rejects.toThrow("invalid catalog");

    const unauthorized = createNotificationPreferencesClient(async () => new Response("denied", { status: 403 }));
    await expect(unauthorized.getCatalog()).rejects.toMatchObject<Partial<NotificationPreferencesError>>({ status: 403 });
  });

  it("bounds response bytes and keeps injected auth headers and request signals intact", async () => {
    const request = vi.fn(async (_path: string, init?: RequestInit) => {
      expect(init?.signal).toBeInstanceOf(AbortSignal);
      return new Response(JSON.stringify(catalog), { headers: { "Content-Length": String(JSON.stringify(catalog).length) } });
    });
    const client = createNotificationPreferencesClient(request, { maxResponseBytes: 32 });

    await expect(client.getCatalog()).rejects.toMatchObject<Partial<NotificationPreferencesError>>({
      status: 502,
      message: "Notification response is too large",
    });
  });

  it("surfaces a bounded request timeout as an unavailable state", async () => {
    const request = vi.fn((_path: string, init?: RequestInit) => new Promise<Response>((_resolve, reject) => {
      init?.signal?.addEventListener("abort", () => reject(new DOMException("aborted", "AbortError")), { once: true });
    }));
    const client = createNotificationPreferencesClient(request, { timeoutMs: 5 });

    await expect(client.getCatalog()).rejects.toBeInstanceOf(NotificationPreferencesUnavailableError);
  });

  it("caps configured request timeouts at sixty seconds", async () => {
    vi.useFakeTimers();
    try {
      let signal: AbortSignal | undefined;
      const request = vi.fn((_path: string, init?: RequestInit) => new Promise<Response>((_resolve, reject) => {
        signal = init?.signal ?? undefined;
        signal?.addEventListener("abort", () => reject(new DOMException("aborted", "AbortError")), { once: true });
      }));
      const client = createNotificationPreferencesClient(request, { timeoutMs: 120_000 });
      const pending = client.getCatalog().catch((error: unknown) => error);
      await vi.advanceTimersByTimeAsync(59_999);
      expect(signal?.aborted).toBe(false);
      await vi.advanceTimersByTimeAsync(1);
      expect(signal?.aborted).toBe(true);
      expect(await pending).toBeInstanceOf(NotificationPreferencesUnavailableError);
    } finally {
      vi.useRealTimers();
    }
  });

  it("rejects invalid timeout values instead of removing the bound", () => {
    expect(() => createNotificationPreferencesClient(vi.fn(), { timeoutMs: Number.NaN })).toThrow("positive");
    expect(() => createNotificationPreferencesClient(vi.fn(), { timeoutMs: -1 })).toThrow("positive");
  });

  it("also bounds a response body that stalls after headers arrive", async () => {
    const request = vi.fn(async () => new Response(new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(new TextEncoder().encode("{"));
      },
    })));
    const client = createNotificationPreferencesClient(request, { timeoutMs: 5 });

    await expect(client.getCatalog()).rejects.toBeInstanceOf(NotificationPreferencesUnavailableError);
  });

  it("builds global category controls from defaults and ignores scoped rows", () => {
    const source = catalog.sources[0]!;
    const updates: unknown[] = [];
    const groups = notificationPreferenceControlGroups(
      [source],
      [
        { ...catalog.preferences[0]!, enabled: false },
        { ...catalog.preferences[0]!, scope_key: "session-1", enabled: true },
      ],
      "All notifications",
      (preference) => updates.push(preference),
    );

    expect(groups[0]?.rows).toHaveLength(1);
    expect(groups[0]?.rows[0]).toMatchObject({ enabled: false, minUrgency: "normal" });
    groups[0]?.rows[0]?.onEnabledChange(true);
    expect(updates).toEqual([expect.objectContaining({ source_scenario: "web-console", scope_key: "", enabled: true })]);
  });
});
