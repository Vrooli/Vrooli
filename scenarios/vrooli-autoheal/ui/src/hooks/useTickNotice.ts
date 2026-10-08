import { useEffect, useState } from "react";

export type TickNoticeTone = "info" | "success" | "warning" | "danger";

export interface TickNotice {
  tone: TickNoticeTone;
  message: string;
  detail?: string;
}

export function useTickNotice() {
  const [tickNotice, setTickNotice] = useState<TickNotice | null>(null);

  useEffect(() => {
    if (!tickNotice) {
      return;
    }
    const timer = window.setTimeout(() => setTickNotice(null), 6000);
    return () => window.clearTimeout(timer);
  }, [tickNotice]);

  return { tickNotice, setTickNotice };
}
