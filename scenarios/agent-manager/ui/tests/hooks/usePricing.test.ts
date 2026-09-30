import assert from "node:assert/strict";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, test, vi } from "vitest";
import {
  useCreateAlias,
  useDeleteOverride,
  useModelAliases,
  useModelOverrides,
  useModelPricing,
  usePricingCacheStatus,
  usePricingSettings,
  useRecalculateModelPricing,
  useRefreshAllPricing,
  useSetOverride,
  useUpdatePricingSettings,
} from "../../src/hooks/usePricing.js";

afterEach(() => vi.unstubAllGlobals());

function ok(payload: unknown = { status: "ok" }) {
  return new Response(JSON.stringify(payload), { status: 200, headers: { "Content-Type": "application/json" } });
}

test("pricing read hooks fetch their resources, including optional aliases and empty model overrides", async () => {
  const fetch = vi.fn(async (url: string) => {
    if (url.includes("/models") && !url.includes("overrides")) return ok({ models: [{ model: "gpt-5" }] });
    if (url.includes("overrides")) return ok({ overrides: [{ component: "input" }] });
    if (url.includes("aliases")) return ok({ aliases: [{ alias: "fast" }] });
    if (url.includes("settings")) return ok({ currency: "USD" });
    return ok({ fresh: true });
  });
  vi.stubGlobal("fetch", fetch);

  const models = renderHook(() => useModelPricing());
  await waitFor(() => assert.equal(models.result.current.data?.[0]?.model, "gpt-5"));
  const overrides = renderHook(() => useModelOverrides("gpt/5"));
  await waitFor(() => assert.equal(overrides.result.current.data?.[0]?.component, "input"));
  const aliases = renderHook(() => useModelAliases("codex"));
  await waitFor(() => assert.equal(aliases.result.current.data?.[0]?.alias, "fast"));
  const settings = renderHook(() => usePricingSettings());
  await waitFor(() => assert.equal(settings.result.current.data?.currency, "USD"));
  const cache = renderHook(() => usePricingCacheStatus());
  await waitFor(() => assert.equal(cache.result.current.data?.fresh, true));
  const missing = renderHook(() => useModelOverrides(null));
  await waitFor(() => assert.deepEqual(missing.result.current.data, []));

  const urls = fetch.mock.calls.map(([url]) => String(url));
  assert.ok(urls.some((url) => url.endsWith("/pricing/models")));
  assert.ok(urls.some((url) => url.endsWith("/pricing/models/gpt%2F5/overrides")));
  assert.ok(urls.some((url) => url.endsWith("/pricing/aliases?runner_type=codex")));
  assert.ok(urls.some((url) => url.endsWith("/pricing/settings")));
  assert.ok(urls.some((url) => url.endsWith("/pricing/cache")));
});

test("pricing mutations use their intended methods, encoded paths, and request bodies", async () => {
  const fetch = vi.fn(async () => ok({ currency: "USD" }));
  vi.stubGlobal("fetch", fetch);
  const recalculation = renderHook(() => useRecalculateModelPricing());
  const setter = renderHook(() => useSetOverride());
  const deleter = renderHook(() => useDeleteOverride());
  const alias = renderHook(() => useCreateAlias());
  const updater = renderHook(() => useUpdatePricingSettings());
  const refresher = renderHook(() => useRefreshAllPricing());

  await act(async () => {
    await recalculation.result.current.recalculate("gpt/5");
    await setter.result.current.setOverride("gpt/5", { component: "input", priceUsd: 1 } as never);
    await deleter.result.current.deleteOverride("gpt/5", "input" as never);
    await alias.result.current.createAlias({ alias: "fast", model: "gpt-5" } as never);
    assert.equal((await updater.result.current.updateSettings({ currency: "USD" } as never)).currency, "USD");
    await refresher.result.current.refreshAll();
  });

  assert.equal(fetch.mock.calls[0]?.[0], "http://localhost:3000/api/v1/pricing/models/gpt%2F5/recalculate");
  assert.equal(fetch.mock.calls[0]?.[1]?.method, "POST");
  assert.equal(fetch.mock.calls[1]?.[1]?.method, "PUT");
  assert.match(String(fetch.mock.calls[1]?.[1]?.body), /priceUsd/);
  assert.equal(fetch.mock.calls[2]?.[1]?.method, "DELETE");
  assert.equal(fetch.mock.calls[3]?.[1]?.method, "POST");
  assert.equal(fetch.mock.calls[4]?.[1]?.method, "PUT");
  assert.equal(fetch.mock.calls[5]?.[0], "http://localhost:3000/api/v1/pricing/refresh");
});

test("pricing hooks retain server failures for operator feedback", async () => {
  vi.stubGlobal("fetch", vi.fn(async () => new Response(JSON.stringify({ message: "pricing unavailable" }), { status: 503 })));
  const models = renderHook(() => useModelPricing());
  await waitFor(() => assert.equal(models.result.current.error, "pricing unavailable"));
  const refresh = renderHook(() => useRefreshAllPricing());
  await act(async () => {
    await assert.rejects(refresh.result.current.refreshAll(), /pricing unavailable/);
  });
  await waitFor(() => assert.equal(refresh.result.current.error, "pricing unavailable"));
});

test("pricing read hooks preserve a useful fallback when payloads are partial or invalid", async () => {
  const fetch = vi.fn(async (url: string) => {
    if (url.includes("/models/")) return new Response("not-json", { status: 502 });
    if (url.endsWith("/models")) return ok({});
    if (url.endsWith("/aliases")) return ok({});
    if (url.endsWith("/cache")) return new Response(JSON.stringify({ error: "cache unavailable" }), { status: 503 });
    return ok({});
  });
  vi.stubGlobal("fetch", fetch);
  const models = renderHook(() => useModelPricing());
  const aliases = renderHook(() => useModelAliases());
  const cache = renderHook(() => usePricingCacheStatus());
  const overrides = renderHook(() => useModelOverrides("gpt-5"));

  await waitFor(() => assert.deepEqual(models.result.current.data, []));
  await waitFor(() => assert.deepEqual(aliases.result.current.data, []));
  await waitFor(() => assert.equal(cache.result.current.error, "cache unavailable"));
  await waitFor(() => assert.equal(overrides.result.current.error, "Request failed: 502"));
});

test("pricing mutation hooks surface failures and accept no-content success responses", async () => {
  const fetch = vi.fn()
    .mockResolvedValueOnce(new Response(null, { status: 204 }))
    .mockResolvedValueOnce(new Response(JSON.stringify({ error: "settings locked" }), { status: 409 }));
  vi.stubGlobal("fetch", fetch);
  const refresh = renderHook(() => useRefreshAllPricing());
  const update = renderHook(() => useUpdatePricingSettings());

  await act(async () => {
    await refresh.result.current.refreshAll();
    await assert.rejects(update.result.current.updateSettings({ currency: "EUR" } as never), /settings locked/);
  });
  assert.equal(refresh.result.current.error, null);
  assert.equal(update.result.current.error, "settings locked");
  assert.equal(update.result.current.loading, false);
});

test("each pricing mutation keeps its own API failure available for recovery UI", async () => {
  vi.stubGlobal("fetch", vi.fn(async () => new Response(JSON.stringify({ error: "pricing write denied" }), { status: 403 })));
  const recalculate = renderHook(() => useRecalculateModelPricing());
  const setOverride = renderHook(() => useSetOverride());
  const deleteOverride = renderHook(() => useDeleteOverride());
  const createAlias = renderHook(() => useCreateAlias());

  await act(async () => {
    await assert.rejects(recalculate.result.current.recalculate("gpt-5"), /pricing write denied/);
    await assert.rejects(setOverride.result.current.setOverride("gpt-5", { component: "input" } as never), /pricing write denied/);
    await assert.rejects(deleteOverride.result.current.deleteOverride("gpt-5", "input" as never), /pricing write denied/);
    await assert.rejects(createAlias.result.current.createAlias({ alias: "fast", model: "gpt-5" } as never), /pricing write denied/);
  });
  for (const hook of [recalculate, setOverride, deleteOverride, createAlias]) {
    assert.equal(hook.result.current.error, "pricing write denied");
    assert.equal(hook.result.current.loading, false);
  }
});
