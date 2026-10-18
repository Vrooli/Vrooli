import { useEffect, useRef, useState } from "react";
import { copyText } from "../../lib/clipboard";

export function usePreviewClipboard() {
  const [status, setStatus] = useState<"copied" | "failed" | null>(null);
  const lifetime = useRef({ alive: true, request: 0, timer: 0 });
  useEffect(() => {
    const state = lifetime.current;
    state.alive = true;
    return () => { state.alive = false; window.clearTimeout(state.timer); };
  }, []);
  const copy = async (value: string) => {
    const state = lifetime.current;
    const request = ++state.request;
    const copied = await copyText(value);
    if (!state.alive || request !== state.request) return;
    setStatus(copied ? "copied" : "failed");
    window.clearTimeout(state.timer);
    state.timer = window.setTimeout(() => { setStatus(null); }, 2000);
  };
  return { copy, status };
}
