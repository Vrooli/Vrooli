// Tests for classifyMicError — the honest getUserMedia failure classifier.
//
// The historical bug this locks down: every getUserMedia failure was reported
// as "Microphone access denied", which is wrong (and unactionable) for the two
// dominant real-world cases — an insecure context on a phone, and a device held
// by another capture. See classifyMicError's doc comment.

import { afterEach, describe, expect, it, vi } from "vitest";
import { classifyMicError } from "./types";

/** Swap window.isSecureContext / navigator.mediaDevices for one test, restoring after. */
function withEnv(
  opts: { secure?: boolean; hasMediaDevices?: boolean },
  run: () => void,
): void {
  const origSecure = Object.getOwnPropertyDescriptor(window, "isSecureContext");
  const origMedia = Object.getOwnPropertyDescriptor(navigator, "mediaDevices");
  try {
    Object.defineProperty(window, "isSecureContext", {
      configurable: true,
      value: opts.secure ?? true,
    });
    Object.defineProperty(navigator, "mediaDevices", {
      configurable: true,
      value: (opts.hasMediaDevices ?? true)
        ? { getUserMedia: () => Promise.resolve({}) }
        : undefined,
    });
    run();
  } finally {
    if (origSecure) Object.defineProperty(window, "isSecureContext", origSecure);
    if (origMedia) Object.defineProperty(navigator, "mediaDevices", origMedia);
    else Reflect.deleteProperty(navigator as object, "mediaDevices");
  }
}

describe("classifyMicError", () => {
  afterEach(() => vi.restoreAllMocks());

  it("flags an insecure context (HTTP on a phone) as an HTTPS problem, not a denial", () => {
    withEnv({ secure: false, hasMediaDevices: true }, () => {
      const msg = classifyMicError(new TypeError("mediaDevices undefined"));
      expect(msg).toMatch(/HTTPS/i);
      expect(msg).not.toMatch(/access denied/i);
    });
  });

  it("flags a missing navigator.mediaDevices (iOS non-secure origin) as an HTTPS problem", () => {
    withEnv({ secure: true, hasMediaDevices: false }, () => {
      const msg = classifyMicError(new TypeError("cannot read getUserMedia of undefined"));
      expect(msg).toMatch(/HTTPS/i);
    });
  });

  it("reports a real permission denial as access denied (secure context)", () => {
    withEnv({ secure: true, hasMediaDevices: true }, () => {
      const err = new DOMException("denied", "NotAllowedError");
      expect(classifyMicError(err)).toBe("Microphone access denied");
    });
  });

  it("reports a busy device (NotReadableError) as busy/held, not denied", () => {
    withEnv({ secure: true, hasMediaDevices: true }, () => {
      const err = new DOMException("busy", "NotReadableError");
      const msg = classifyMicError(err);
      expect(msg).toMatch(/busy|held/i);
      expect(msg).not.toMatch(/access denied/i);
    });
  });

  it("reports a missing device (NotFoundError) distinctly", () => {
    withEnv({ secure: true, hasMediaDevices: true }, () => {
      const err = new DOMException("none", "NotFoundError");
      expect(classifyMicError(err)).toMatch(/no usable microphone/i);
    });
  });

  it("falls back to a generic retry message for unknown DOMExceptions", () => {
    withEnv({ secure: true, hasMediaDevices: true }, () => {
      const err = new DOMException("weird", "SomeNovelError");
      expect(classifyMicError(err)).toMatch(/try again/i);
    });
  });
});
