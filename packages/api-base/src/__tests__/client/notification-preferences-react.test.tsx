import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor, act } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { createNotificationPreferencesClient } from "../../client/notification-preferences.js";
import { notificationPreferencesQueryKey, useNotificationPreferences } from "../../client/notification-preferences-react.js";

const initial = {
  sources: [{ source_scenario: "web-console", display_name: "Web Console", icon: "", deep_link_base: "", declared: true, categories: [] }],
  preferences: [{ source_scenario: "web-console", category: "", scope_key: "", enabled: true, min_urgency: "normal", channel_override: "" }],
};

describe("shared notification preferences hook", () => {
  it("shares the scoped query key and invalidates to the saved server state", async () => {
    let stored = initial;
    const request = vi.fn(async (_path: string, init?: RequestInit) => {
      if (init?.method === "PUT") {
        stored = { ...stored, preferences: [JSON.parse(String(init.body)) as typeof stored.preferences[number]] };
        return Response.json(stored.preferences[0]);
      }
      return Response.json(stored);
    });
    const client = createNotificationPreferencesClient(request);
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
    const { result } = renderHook(() => useNotificationPreferences(client, ["web-console"]), { wrapper });

    await waitFor(() => expect(result.current.catalog.data?.preferences).toEqual(initial.preferences));
    await act(async () => {
      result.current.save.mutate({ ...initial.preferences[0]!, enabled: false });
    });
    await waitFor(() => expect(result.current.catalog.data?.preferences[0]?.enabled).toBe(false));

    expect(notificationPreferencesQueryKey(["web-console"])).toEqual(["notification-catalog", "web-console"]);
    expect(request.mock.calls[0]?.[0]).toBe("/notifications/catalog?source_scenario=web-console");
    expect(request.mock.calls[0]?.[1]?.signal).toBeInstanceOf(AbortSignal);
    queryClient.clear();
  });
});
