import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  NotificationsUnavailableError,
  fetchNotificationCatalog,
  updateNotificationPreference,
  type NotificationPreference,
} from "./notifications";

vi.mock("../lib/auth", () => ({
  getAccessToken: vi.fn().mockResolvedValue("test-token"),
}));

vi.mock("../lib/deviceIdentity", () => ({
  deviceIdentity: vi.fn(() => ({ id: "device-123" })),
}));

describe("api/notifications", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  describe("fetchNotificationCatalog", () => {
    it("GETs the catalog with the device header and bearer token and normalizes the body", async () => {
      fetchSpy.mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            sources: [
              {
                source_scenario: "web-console",
                display_name: "Web Console",
                icon: "",
                deep_link_base: "",
                declared: true,
                categories: [],
              },
            ],
            preferences: [
              {
                source_scenario: "web-console",
                category: "session.input_needed",
                enabled: true,
                min_urgency: "normal",
                channel_override: "",
              },
            ],
          }),
          { status: 200 },
        ),
      );

      const catalog = await fetchNotificationCatalog();

      expect(catalog.sources).toHaveLength(1);
      expect(catalog.preferences).toHaveLength(1);

      expect(fetchSpy).toHaveBeenCalledTimes(1);
      const [url, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
      expect(url).toMatch(/\/api\/v1\/notifications\/catalog\?source_scenario=web-console$/);
      expect(init).toMatchObject({ method: "GET", cache: "no-store" });
      const headers = new Headers(init.headers);
      expect(headers.get("X-Vrooli-Device-Id")).toBe("device-123");
      expect(headers.get("Authorization")).toBe("Bearer test-token");
    });

    it("scopes the request to the given source and drops other scenarios a hub may still return", async () => {
      const source = (slug: string) => ({
        source_scenario: slug,
        display_name: slug,
        icon: "",
        deep_link_base: "",
        declared: true,
        categories: [],
      });
      const pref = (slug: string) => ({
        source_scenario: slug,
        category: "",
        enabled: true,
        min_urgency: "normal",
        channel_override: "",
      });
      // An older hub ignores ?source_scenario= and returns every source.
      fetchSpy.mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            sources: [source("git-control-tower"), source("my-app"), source("web-console")],
            preferences: [pref("git-control-tower"), pref("my-app")],
          }),
          { status: 200 },
        ),
      );

      const catalog = await fetchNotificationCatalog("my-app");

      const [url] = fetchSpy.mock.calls[0] as [string, RequestInit];
      expect(url).toMatch(/\/api\/v1\/notifications\/catalog\?source_scenario=my-app$/);
      expect(catalog.sources.map((s) => s.source_scenario)).toEqual(["my-app"]);
      expect(catalog.preferences.map((p) => p.source_scenario)).toEqual(["my-app"]);
    });

    it("defaults missing or non-array sources/preferences to empty arrays", async () => {
      fetchSpy.mockResolvedValueOnce(
        new Response(JSON.stringify({ sources: "nope" }), { status: 200 }),
      );

      const catalog = await fetchNotificationCatalog();

      expect(catalog).toEqual({ sources: [], preferences: [] });
    });

    it("raises NotificationsUnavailableError when the facade returns 503", async () => {
      fetchSpy.mockResolvedValueOnce(new Response("", { status: 503 }));

      await expect(fetchNotificationCatalog()).rejects.toBeInstanceOf(
        NotificationsUnavailableError,
      );
    });

    it("raises a general API error for other non-ok statuses", async () => {
      fetchSpy.mockResolvedValueOnce(new Response("", { status: 500 }));

      await expect(fetchNotificationCatalog()).rejects.toThrow(
        /notifications request failed \(500\)/,
      );
    });
  });

  describe("updateNotificationPreference", () => {
    const preference: NotificationPreference = {
      source_scenario: "git-control-tower",
      category: "review",
      enabled: false,
      min_urgency: "high",
      channel_override: "",
    };

    it("PUTs the preference as JSON with auth + content-type headers", async () => {
      fetchSpy.mockResolvedValueOnce(new Response("", { status: 200 }));

      await updateNotificationPreference(preference);

      expect(fetchSpy).toHaveBeenCalledTimes(1);
      const [url, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
      expect(url).toMatch(/\/api\/v1\/notifications\/preferences$/);
      expect(init.method).toBe("PUT");
      const headers = new Headers(init.headers);
      expect(headers.get("Content-Type")).toBe("application/json");
      expect(headers.get("X-Vrooli-Device-Id")).toBe("device-123");
      expect(headers.get("Authorization")).toBe("Bearer test-token");
      expect(init.body).toBe(JSON.stringify(preference));
    });

    it("propagates NotificationsUnavailableError from a 503", async () => {
      fetchSpy.mockResolvedValueOnce(new Response("", { status: 503 }));

      await expect(updateNotificationPreference(preference)).rejects.toBeInstanceOf(
        NotificationsUnavailableError,
      );
    });
  });
});
