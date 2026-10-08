import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { useSecureContextCapabilities } from "./useSecureContextCapabilities";

const secureContext = Object.getOwnPropertyDescriptor(window, "isSecureContext");
const mediaDevices = Object.getOwnPropertyDescriptor(navigator, "mediaDevices");
const wakeLock = Object.getOwnPropertyDescriptor(navigator, "wakeLock");

afterEach(() => {
  if (secureContext) Object.defineProperty(window, "isSecureContext", secureContext);
  if (mediaDevices) Object.defineProperty(navigator, "mediaDevices", mediaDevices);
  else delete (navigator as { mediaDevices?: MediaDevices }).mediaDevices;
  if (wakeLock) Object.defineProperty(navigator, "wakeLock", wakeLock);
  else delete (navigator as { wakeLock?: unknown }).wakeLock;
});

describe("useSecureContextCapabilities", () => {
  it("exposes unavailable capabilities when the console is not secure", async () => {
    Object.defineProperty(window, "isSecureContext", { configurable: true, value: false });
    Object.defineProperty(navigator, "mediaDevices", {
      configurable: true,
      value: { getUserMedia: () => Promise.resolve({}) },
    });
    Object.defineProperty(navigator, "wakeLock", { configurable: true, value: {} });

    const { result } = renderHook(() => useSecureContextCapabilities());

    await waitFor(() => expect(result.current.clipboard).toBe("unsupported"));
    expect(result.current.microphone).toBe("unsupported");
    expect(result.current.wakeLock).toBe("unsupported");
  });
});
