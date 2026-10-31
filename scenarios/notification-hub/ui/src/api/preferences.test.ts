import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  MIN_URGENCY_CHOICES,
  fetchNotificationCatalog,
  updateNotificationPreference,
  type NotificationPreference,
} from "./preferences";

describe("api/preferences", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("exposes the urgency ladder in ascending order", () => {
    expect(MIN_URGENCY_CHOICES).toEqual(["low", "normal", "high", "critical"]);
  });

  describe("fetchNotificationCatalog", () => {
    it("requests the catalog with the browser-session header and same-origin cookie", async () => {
      fetchSpy.mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            sources: [
              {
                source_scenario: "git-control-tower",
                display_name: "Git Control Tower",
                icon: "",
                deep_link_base: "",
                declared: true,
                categories: [],
              },
            ],
            preferences: [
              {
                source_scenario: "git-control-tower",
                category: "review",
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
      expect(catalog.sources[0]?.source_scenario).toBe("git-control-tower");
      expect(catalog.preferences).toHaveLength(1);

      expect(fetchSpy).toHaveBeenCalledTimes(1);
      const [url, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
      expect(url).toMatch(/\/notifications\/catalog$/);
      expect(init).toMatchObject({ credentials: "same-origin", cache: "no-store" });
      const headers = new Headers(init.headers);
      expect(headers.get("X-Vrooli-Browser-Session")).toBe("1");
    });

    it("tolerates a response missing sources/preferences by defaulting to empty arrays", async () => {
      fetchSpy.mockResolvedValueOnce(new Response("{}", { status: 200 }));

      const catalog = await fetchNotificationCatalog();

      expect(catalog).toEqual({ sources: [], preferences: [] });
    });

    it("raises an error with the status code when the catalog is unavailable", async () => {
      fetchSpy.mockResolvedValueOnce(new Response("", { status: 503 }));

      await expect(fetchNotificationCatalog()).rejects.toThrow(
        "Notification catalog request failed (503)",
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

    it("PUTs the preference as JSON and returns the server's echo", async () => {
      fetchSpy.mockResolvedValueOnce(
        new Response(JSON.stringify(preference), { status: 200 }),
      );

      const result = await updateNotificationPreference(preference);

      expect(result).toEqual(preference);
      expect(fetchSpy).toHaveBeenCalledTimes(1);
      const [url, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
      expect(url).toMatch(/\/notifications\/preferences$/);
      expect(init.method).toBe("PUT");
      expect(init).toMatchObject({ credentials: "same-origin", cache: "no-store" });
      const headers = new Headers(init.headers);
      expect(headers.get("Content-Type")).toBe("application/json");
      expect(headers.get("X-Vrooli-Browser-Session")).toBe("1");
      expect(init.body).toBe(JSON.stringify(preference));
    });

    it("raises an error with the status code when the update fails", async () => {
      fetchSpy.mockResolvedValueOnce(new Response("", { status: 500 }));

      await expect(updateNotificationPreference(preference)).rejects.toThrow(
        "Notification preference update failed (500)",
      );
    });
  });
});
