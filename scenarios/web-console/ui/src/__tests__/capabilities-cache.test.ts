import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { apiBaseMock } from "../test-utils";

vi.mock("@vrooli/api-base", () => apiBaseMock());

// We need to mock fetch at the global level to control responses
const mockFetch = vi.fn();
vi.stubGlobal("fetch", mockFetch);

// Import after mocks
const { fetchCapabilitiesLivenessCached, _resetCapabilitiesCache } = await import("../lib/api");

const fakeResponse = (caps: { id: string; status: string }[]) => ({
  ok: true,
  json: () => Promise.resolve({ capabilities: caps, timestamp: new Date().toISOString() }),
});

describe("fetchCapabilitiesLivenessCached", () => {
  beforeEach(() => {
    _resetCapabilitiesCache();
    mockFetch.mockReset();
    mockFetch.mockResolvedValue(fakeResponse([{ id: "kokoro-tts", status: "available" }]));
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("deduplicates concurrent calls into a single fetch", async () => {
    const [r1, r2, r3] = await Promise.all([
      fetchCapabilitiesLivenessCached(),
      fetchCapabilitiesLivenessCached(),
      fetchCapabilitiesLivenessCached(),
    ]);

    // All three should return the same result
    expect(r1).toBe(r2);
    expect(r2).toBe(r3);
    // But only one fetch was made
    expect(mockFetch).toHaveBeenCalledTimes(1);
  });

  it("returns cached result within TTL", async () => {
    const first = await fetchCapabilitiesLivenessCached();
    const second = await fetchCapabilitiesLivenessCached();

    expect(first).toBe(second);
    expect(mockFetch).toHaveBeenCalledTimes(1);
  });

  it("refreshes after TTL expires", async () => {
    vi.useFakeTimers();

    await fetchCapabilitiesLivenessCached();
    expect(mockFetch).toHaveBeenCalledTimes(1);

    // Advance past the 30s TTL
    vi.advanceTimersByTime(31_000);

    await fetchCapabilitiesLivenessCached();
    expect(mockFetch).toHaveBeenCalledTimes(2);

    vi.useRealTimers();
  });

  it("clears cache on rejection so next caller retries", async () => {
    mockFetch.mockRejectedValueOnce(new Error("network down"));

    await expect(fetchCapabilitiesLivenessCached()).rejects.toThrow("network down");

    // Next call should retry (cache was cleared)
    mockFetch.mockResolvedValueOnce(fakeResponse([{ id: "kokoro-tts", status: "available" }]));
    const result = await fetchCapabilitiesLivenessCached();

    expect(result.capabilities[0]?.status).toBe("available");
    expect(mockFetch).toHaveBeenCalledTimes(2);
  });
});
