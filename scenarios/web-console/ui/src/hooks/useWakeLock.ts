import { useCallback, useEffect, useRef, useState } from "react";

/** Describes why the wake lock is not active. */
export type WakeLockStatus =
  | "active"        // Lock successfully held
  | "off"           // Feature disabled by user
  | "unsupported"   // Browser/context doesn't support Wake Lock API
  | "denied"        // Request was denied (low battery, permissions, etc.)
  | "released";     // Was active but got released; will retry on next visibility change

/** Max consecutive re-acquisition attempts before giving up. */
const MAX_RETRIES = 5;
/** Base delay (ms) for exponential backoff between retries. */
const BASE_DELAY_MS = 1_000;
/** How often (ms) the heartbeat checks sentinel + video health. */
const HEARTBEAT_MS = 30_000;

/**
 * Manages the Screen Wake Lock API to prevent the device screen from
 * dimming or locking. Designed for hands-free voice interaction where
 * the user may not touch the screen for extended periods.
 *
 * - Requests a `"screen"` wake lock when `enabled` is `true`.
 * - Re-acquires automatically when the page becomes visible again
 *   (browsers release wake locks when a tab is hidden).
 * - Re-acquires when the sentinel fires a `release` event (e.g. OS
 *   power-management policy released it while the tab was still visible).
 * - Retries with exponential backoff if re-acquisition fails.
 * - Falls back to a hidden silent-video loop only when the Wake Lock API
 *   isn't available. The video hack is avoided when the native API works
 *   because iOS treats even muted video as an active media session, which
 *   causes the Dynamic Island audio indicator to flash on every keystroke
 *   (keyboard sounds interrupt the audio session, pausing the video, which
 *   our handler immediately resumes — re-triggering the indicator).
 * - Returns the current {@link WakeLockStatus} so the UI can inform
 *   the user when the lock isn't working.
 */
export function useWakeLock(enabled: boolean): WakeLockStatus {
  const [status, setStatus] = useState<WakeLockStatus>(() => {
    if (!enabled) return "off";
    if (!("wakeLock" in navigator)) return "unsupported";
    return "off"; // will become "active" once effect runs
  });

  const sentinelRef = useRef<WakeLockSentinel | null>(null);
  const enabledRef = useRef(enabled);
  enabledRef.current = enabled;

  // ── Video-fallback refs ────────────────────────────────────────────
  const videoRef = useRef<HTMLVideoElement | null>(null);

  const startVideoFallback = useCallback(() => {
    if (videoRef.current) return; // already running
    try {
      const video = document.createElement("video");
      video.setAttribute("playsinline", "");
      video.setAttribute("muted", "");
      video.muted = true;
      video.loop = true;

      // Tiny 1-second silent mp4 (base64) — keeps the screen awake by
      // tricking the browser into thinking media is playing.
      // Source: widely-used NoSleep.js technique.
      video.src =
        "data:video/mp4;base64,AAAAIGZ0eXBpc29tAAACAGlzb21pc28yYXZjMW1wNDEAAAAIZnJlZQAAAu1tZGF0AAACrQYF//+p3EXpvebZSLeWLNgg2SPu73gyNjQgLSBjb3JlIDE1MiByMjg1NCBlOWE1OTAzIC0gSC4yNjQvTVBFRy00IEFWQyBjb2RlYyAtIENvcHlsZWZ0IDIwMDMtMjAxNyAtIGh0dHA6Ly93d3cudmlkZW9sYW4ub3JnL3gyNjQuaHRtbCAtIG9wdGlvbnM6IGNhYmFjPTEgcmVmPTMgZGVibG9jaz0xOjA6MCBhbmFseXNlPTB4MzoweDExMyBtZT1oZXggc3VibWU9NyBwc3k9MSBwc3lfcmQ9MS4wMDowLjAwIG1peGVkX3JlZj0xIG1lX3JhbmdlPTE2IGNocm9tYV9tZT0xIHRyZWxsaXM9MSA4eDhkY3Q9MSBjcW09MCBkZWFkem9uZT0yMSwxMSBmYXN0X3Bza2lwPTEgY2hyb21hX3FwX29mZnNldD0tMiB0aHJlYWRzPTEgbG9va2FoZWFkX3RocmVhZHM9MSBzbGljZWRfdGhyZWFkcz0wIG5yPTAgZGVjaW1hdGU9MSBpbnRlcmxhY2VkPTAgYmx1cmF5X2NvbXBhdD0wIGNvbnN0cmFpbmVkX2ludHJhPTAgYmZyYW1lcz0zIGJfcHlyYW1pZD0yIGJfYWRhcHQ9MSBiX2JpYXM9MCBkaXJlY3Q9MSB3ZWlnaHRiPTEgb3Blbl9nb3A9MCB3ZWlnaHRwPTIga2V5aW50PTI1MCBrZXlpbnRfbWluPTI1IHNjZW5lY3V0PTQwIGludHJhX3JlZnJlc2g9MCByY19sb29rYWhlYWQ9NDAgcmM9Y3JmIG1idHJlZT0xIGNyZj0yMy4wIHFjb21wPTAuNjAgcXBtaW49MCBxcG1heD02OSBxcHN0ZXA9NCBpcF9yYXRpbz0xLjQwIGFxPTE6MS4wMAAAAAAPZYiEAD//8m+P5JcRceMAAAAIbW9vdgAAAGxtdmhkAAAAAAAAAAAAAAAAAAAD6AAAABQAAQAAAQAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAgAAACBzdHRzAAAAAAAAAAEAAAABAAAABAAAAAAUc3RzcwAAAAAAAAABAAAAAQAAABRzdHNjAAAAAAAAAAEAAAABAAAAAgAAAAEAAAAcc3RzegAAAAAAAAAAAAAAAgAAAtEAAAAMAAAAFHN0Y28AAAAAAAAAAQAAADAAAABidWR0YQAAAFptZXRhAAAAAAAAACFoZGxyAAAAAAAAAABtZGlyYXBwbAAAAAAAAAAAAAAAAC1pbHN0AAAAJal0b28AAAAdZGF0YQAAAAEAAAAATGF2ZjU3LjU2LjEwMQ==";

      // Position off-screen so it's invisible
      video.style.position = "fixed";
      video.style.top = "-1px";
      video.style.left = "-1px";
      video.style.width = "1px";
      video.style.height = "1px";
      video.style.opacity = "0.01";

      // If iOS pauses the video (e.g. audio session change when mic
      // activates), immediately try to resume it.
      video.addEventListener("pause", () => {
        void video.play().catch(() => {});
      });

      document.body.appendChild(video);
      void video.play().catch(() => {
        // Autoplay blocked — can't help further without user gesture
      });
      videoRef.current = video;
    } catch {
      // DOM manipulation failed — give up silently
    }
  }, []);

  const stopVideoFallback = useCallback(() => {
    if (!videoRef.current) return;
    videoRef.current.pause();
    videoRef.current.remove();
    videoRef.current = null;
  }, []);

  // ── Wake Lock API + video dual-mode ──────────────────────────────
  useEffect(() => {
    const supportsWakeLock = "wakeLock" in navigator;

    if (!enabled) {
      setStatus("off");
      return;
    }

    // Only start the silent-video fallback when the native Wake Lock API
    // isn't available. On iOS, even a muted <video> registers as an active
    // media session — the keyboard's per-keystroke audio-session interrupts
    // pause the video, our pause→play handler resumes it, and the Dynamic
    // Island flashes a "media started" indicator on every character. Modern
    // iOS (16.4+) supports navigator.wakeLock natively, so the hack is only
    // needed as a true last resort.
    if (!supportsWakeLock) {
      startVideoFallback();
      setStatus(videoRef.current ? "active" : "unsupported");
      return () => {
        stopVideoFallback();
      };
    }

    let disposed = false;
    let retryCount = 0;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;
    let acquiring = false;

    async function requestLock() {
      if (acquiring) return;
      acquiring = true;

      try {
        // Release any existing sentinel before acquiring a new one.
        if (sentinelRef.current) {
          try {
            await sentinelRef.current.release();
          } catch { /* already released */ }
          sentinelRef.current = null;
        }

        if (disposed || !enabledRef.current || document.visibilityState !== "visible") return;

        const sentinel = await navigator.wakeLock.request("screen");
        if (disposed) {
          // Effect was cleaned up while we were awaiting — release immediately
          try { await sentinel.release(); } catch { /* noop */ }
          return;
        }
        sentinelRef.current = sentinel;
        retryCount = 0;
        setStatus("active");

        // Listen for the browser/OS releasing the lock (e.g. power-save
        // policy, tab hidden, etc.) so we can re-acquire when possible.
        sentinel.addEventListener("release", () => {
          // Only act if this sentinel is still the current one
          if (sentinelRef.current !== sentinel) return;
          sentinelRef.current = null;
          if (!enabledRef.current) return;
          setStatus("released");
          // Re-acquire immediately if tab is still visible
          if (document.visibilityState === "visible" && !disposed) {
            void requestLock();
          }
        });
      } catch (err) {
        if (disposed) return;
        retryCount++;
        if (retryCount <= MAX_RETRIES) {
          const delay = BASE_DELAY_MS * Math.pow(2, retryCount - 1);
          console.debug("[useWakeLock] request failed, retry %d/%d in %dms:", retryCount, MAX_RETRIES, delay, err);
          setStatus("released"); // transient — retrying
          retryTimer = setTimeout(() => {
            if (!disposed && enabledRef.current) void requestLock();
          }, delay);
        } else {
          console.debug("[useWakeLock] request denied after %d retries:", MAX_RETRIES, err);
          setStatus("denied");
        }
      } finally {
        acquiring = false;
      }
    }

    async function releaseLock() {
      if (sentinelRef.current) {
        try {
          await sentinelRef.current.release();
        } catch { /* already released */ }
        sentinelRef.current = null;
      }
    }

    function handleVisibilityChange() {
      if (document.visibilityState === "visible" && enabledRef.current && !disposed) {
        retryCount = 0; // fresh start on tab re-focus
        void requestLock();
      }
    }

    void requestLock();
    document.addEventListener("visibilitychange", handleVisibilityChange);

    // Periodic heartbeat: re-acquire sentinel if lost, re-play video if paused
    const heartbeat = setInterval(() => {
      if (disposed || !enabledRef.current) return;

      // Re-play video if iOS audio session paused it
      if (videoRef.current?.paused) {
        void videoRef.current.play().catch(() => {});
      }

      // Re-acquire sentinel if it was silently lost
      if (!sentinelRef.current && document.visibilityState === "visible") {
        void requestLock();
      }
    }, HEARTBEAT_MS);

    return () => {
      disposed = true;
      if (retryTimer) clearTimeout(retryTimer);
      clearInterval(heartbeat);
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      void releaseLock();
      stopVideoFallback();
    };
  }, [enabled, startVideoFallback, stopVideoFallback]);

  return status;
}
