import { useEffect, useState } from "react";

export type MicPermission = "unknown" | "granted" | "denied" | "prompt";

// useMicPermission tracks the browser's microphone Permissions API state.
// Older mobile WebViews and jsdom omit `navigator.permissions` entirely
// even though TS lib.dom declares it as always defined; keep the runtime
// guard so tests and stripped WebViews don't crash.
export function useMicPermission(): MicPermission {
  const [state, setState] = useState<MicPermission>("unknown");
  useEffect(() => {
    // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
    if (!navigator.permissions || typeof navigator.permissions.query !== "function") {
      return;
    }
    let cancelled = false;
    void navigator.permissions
      .query({ name: "microphone" })
      .then((status) => {
        if (cancelled) return;
        setState(status.state);
        status.onchange = () => setState(status.state);
      })
      .catch(() => {
        // Browser does not expose the microphone permission descriptor.
      });
    return () => {
      cancelled = true;
    };
  }, []);
  return state;
}
