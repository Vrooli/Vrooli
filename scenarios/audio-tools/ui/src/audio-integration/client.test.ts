/**
 * Tests for audio-integration/client.tsx
 *
 * Covers:
 *   - createAudioToolsClient with and without custom transport
 *   - AudioToolsProvider + useAudioToolsClient hook (happy path + missing provider throw)
 *   - useAudioToolsUnavailableReason
 *   - getActiveAudioToolsClient sentinel fallback
 *   - setActiveAudioToolsClientForTesting
 */
import { describe, it, expect, vi, afterEach } from "vitest";
import { renderHook, cleanup } from "@testing-library/react";
import type { ReactNode } from "react";
import { createElement } from "react";

// Mock @connectrpc/connect so we don't need a real transport
const createClientMock = vi.fn((service: unknown, _transport: unknown) => ({ _service: service }));
vi.mock("@connectrpc/connect", () => ({
  createClient: (service: unknown, transport: unknown) => createClientMock(service, transport),
}));

// createConnectTransport is called when no transport is provided
const createConnectTransportMock = vi.fn((_opts: unknown) => ({ type: "connect-transport" }));
vi.mock("@connectrpc/connect-web", () => ({
  createConnectTransport: (opts: unknown) => createConnectTransportMock(opts),
}));

// Proto service types — just empty objects for structural purposes
vi.mock("@vrooli/proto-types/audio-tools/v1/stt/stt_pb", () => ({
  STTService: { typeName: "STTService" },
}));
vi.mock("@vrooli/proto-types/audio-tools/v1/stt/stt_admin_pb", () => ({
  STTAdminService: { typeName: "STTAdminService" },
}));
vi.mock("@vrooli/proto-types/audio-tools/v1/summarize/summarize_pb", () => ({
  SummarizeService: { typeName: "SummarizeService" },
}));
vi.mock("@vrooli/proto-types/audio-tools/v1/tts/tts_pb", () => ({
  TTSService: { typeName: "TTSService" },
}));

import {
  createAudioToolsClient,
  AudioToolsProvider,
  useAudioToolsClient,
  useAudioToolsUnavailableReason,
  getActiveAudioToolsClient,
  setActiveAudioToolsClientForTesting,
} from "./client";

afterEach(() => {
  cleanup();
  // Reset the active client between tests
  setActiveAudioToolsClientForTesting(null);
  vi.clearAllMocks();
});

describe("createAudioToolsClient", () => {
  it("creates a client with four service fields and exposes baseUrl", () => {
    const client = createAudioToolsClient({ baseUrl: "http://localhost:9000" });
    expect(client.baseUrl).toBe("http://localhost:9000");
    expect(client.stt).toBeDefined();
    expect(client.sttAdmin).toBeDefined();
    expect(client.tts).toBeDefined();
    expect(client.summarize).toBeDefined();
  });

  it("creates a connect transport internally when none is supplied", () => {
    createAudioToolsClient({ baseUrl: "http://localhost:9000" });
    expect(createConnectTransportMock).toHaveBeenCalledWith({ baseUrl: "http://localhost:9000" });
  });

  it("uses the injected transport instead of creating one", () => {
    const fakeTransport = { type: "fake" } as unknown as Parameters<typeof createAudioToolsClient>[0]["transport"];
    createAudioToolsClient({ baseUrl: "http://localhost:9000", transport: fakeTransport });
    // createConnectTransport should NOT have been called
    expect(createConnectTransportMock).not.toHaveBeenCalled();
  });
});

describe("AudioToolsProvider + useAudioToolsClient", () => {
  function makeClient() {
    return createAudioToolsClient({ baseUrl: "http://test:1" });
  }

  it("provides the client to useAudioToolsClient", () => {
    const client = makeClient();
    const wrapper = ({ children }: { children: ReactNode }) =>
      createElement(AudioToolsProvider, { client, children });
    const { result } = renderHook(() => useAudioToolsClient(), { wrapper });
    expect(result.current).toBe(client);
  });

  it("throws when useAudioToolsClient is called outside the provider", () => {
    expect(() => {
      const { result } = renderHook(() => useAudioToolsClient());
      // Access to force evaluation
      return result.current;
    }).toThrow(/useAudioToolsClient/);
  });

  it("returns undefined unavailableReason by default", () => {
    const client = makeClient();
    const wrapper = ({ children }: { children: ReactNode }) =>
      createElement(AudioToolsProvider, { client, children });
    const { result } = renderHook(() => useAudioToolsUnavailableReason(), { wrapper });
    expect(result.current).toBeUndefined();
  });

  it("exposes the unavailableReason prop via useAudioToolsUnavailableReason", () => {
    const client = makeClient();
    const wrapper = ({ children }: { children: ReactNode }) =>
      createElement(AudioToolsProvider, { client, unavailableReason: "offline", children });
    const { result } = renderHook(() => useAudioToolsUnavailableReason(), { wrapper });
    expect(result.current).toBe("offline");
  });

  it("returns undefined from useAudioToolsUnavailableReason when called outside provider", () => {
    const { result } = renderHook(() => useAudioToolsUnavailableReason());
    expect(result.current).toBeUndefined();
  });
});

describe("getActiveAudioToolsClient", () => {
  it("creates a sentinel-URL client when no provider has registered one", () => {
    setActiveAudioToolsClientForTesting(null);
    const client = getActiveAudioToolsClient();
    expect(client.baseUrl).toBe("http://localhost:3000");
  });

  it("returns the same sentinel client on subsequent calls (no re-create)", () => {
    setActiveAudioToolsClientForTesting(null);
    const a = getActiveAudioToolsClient();
    const b = getActiveAudioToolsClient();
    expect(a).toBe(b);
  });

  it("returns the explicitly injected test client", () => {
    const testClient = createAudioToolsClient({ baseUrl: "http://test-injected:99" });
    setActiveAudioToolsClientForTesting(testClient);
    expect(getActiveAudioToolsClient()).toBe(testClient);
  });

  it("can be reset to null via setActiveAudioToolsClientForTesting", () => {
    const testClient = createAudioToolsClient({ baseUrl: "http://test-injected:99" });
    setActiveAudioToolsClientForTesting(testClient);
    setActiveAudioToolsClientForTesting(null);
    expect(getActiveAudioToolsClient().baseUrl).toBe("http://localhost:3000");
  });
});
