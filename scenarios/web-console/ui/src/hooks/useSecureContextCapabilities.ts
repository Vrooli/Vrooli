import { useEffect, useState } from "react";
import { isClipboardSupported } from "../lib/clipboard";

export type SecureCapabilityState = "available" | "unsupported" | "unknown";
export interface SecureContextCapabilities {
  clipboard: SecureCapabilityState;
  microphone: SecureCapabilityState;
  wakeLock: SecureCapabilityState;
  installability: SecureCapabilityState;
}

const UNKNOWN: SecureContextCapabilities = {
  clipboard: "unknown",
  microphone: "unknown",
  wakeLock: "unknown",
  installability: "unknown",
};

export function useSecureContextCapabilities(): SecureContextCapabilities {
  const [capabilities, setCapabilities] = useState<SecureContextCapabilities>(UNKNOWN);

  useEffect(() => {
    const secure = typeof window !== "undefined" && window.isSecureContext;
    const next: SecureContextCapabilities = {
      clipboard: secure && isClipboardSupported() ? "available" : "unsupported",
      microphone: secure && !!navigator.mediaDevices?.getUserMedia ? "available" : "unsupported",
      wakeLock: secure && "wakeLock" in navigator ? "available" : "unsupported",
      installability: "beforeinstallprompt" in window ? "available" : "unknown",
    };
    const onBeforeInstallPrompt = () => setCapabilities((current) => ({ ...current, installability: "available" }));
    window.addEventListener("beforeinstallprompt", onBeforeInstallPrompt);
    setCapabilities(next);
    return () => window.removeEventListener("beforeinstallprompt", onBeforeInstallPrompt);
  }, []);

  return capabilities;
}
