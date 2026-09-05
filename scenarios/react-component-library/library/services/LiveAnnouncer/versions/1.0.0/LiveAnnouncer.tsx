/** @vrooliComponentSource services.live-announcer */
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";

export type AnnouncementPriority = "polite" | "assertive";

export interface AnnouncementOptions {
  priority?: AnnouncementPriority;
  durationMs?: number;
}

export interface LiveAnnouncerHandle {
  announce: (message: string, options?: AnnouncementOptions) => void;
  clear: () => void;
}

interface QueuedAnnouncement {
  id: number;
  message: string;
  priority: AnnouncementPriority;
  durationMs: number;
}

const LiveAnnouncerContext = createContext<LiveAnnouncerHandle | null>(null);
let nextAnnouncementID = 0;
let fallbackHandle: LiveAnnouncerHandle | undefined;

function getFallbackHandle(): LiveAnnouncerHandle {
  if (fallbackHandle) return fallbackHandle;
  let node: HTMLElement | undefined;
  const ensureNode = () => {
    if (typeof document === "undefined") return undefined;
    node ??=
      document.querySelector<HTMLElement>("[data-vrooli-live-announcer]") ??
      undefined;
    if (!node) {
      node = document.createElement("div");
      node.dataset.vrooliLiveAnnouncer = "true";
      node.setAttribute("aria-live", "polite");
      node.setAttribute("aria-atomic", "true");
      node.style.position = "fixed";
      node.style.inlineSize = "1px";
      node.style.blockSize = "1px";
      node.style.overflow = "hidden";
      node.style.clipPath = "inset(50%)";
      document.body.appendChild(node);
    }
    return node;
  };
  fallbackHandle = {
    announce(message, options = {}) {
      const target = ensureNode();
      if (!target || !message.trim()) return;
      target.setAttribute("aria-live", options.priority ?? "polite");
      target.textContent = message.trim();
    },
    clear() {
      if (node) node.textContent = "";
    },
  };
  return fallbackHandle;
}

export function LiveAnnouncer({ children }: { children?: ReactNode }) {
  const [current, setCurrent] = useState<QueuedAnnouncement | null>(null);
  const queue = useRef<QueuedAnnouncement[]>([]);

  const clear = useCallback(() => {
    queue.current = [];
    setCurrent(null);
  }, []);

  const announce = useCallback(
    (message: string, options: AnnouncementOptions = {}) => {
      const normalized = message.trim();
      if (!normalized) return;
      const item: QueuedAnnouncement = {
        id: ++nextAnnouncementID,
        message: normalized,
        priority: options.priority ?? "polite",
        durationMs: Math.max(options.durationMs ?? 1800, 900),
      };
      queue.current = [...queue.current, item].slice(-8);
      setCurrent((visible) => visible ?? queue.current.shift() ?? null);
    },
    [],
  );

  useEffect(() => {
    if (!current) return undefined;
    if (typeof window === "undefined") return undefined;
    const timer = window.setTimeout(() => {
      setCurrent(queue.current.shift() ?? null);
    }, current.durationMs);
    return () => window.clearTimeout(timer);
  }, [current]);

  const handle = useMemo<LiveAnnouncerHandle>(
    () => ({ announce, clear }),
    [announce, clear],
  );

  return (
    <LiveAnnouncerContext.Provider value={handle}>
      {children}
      <div
        data-vrooli-live-announcer
        data-announcement-id={current?.id}
        data-announcement-priority={current?.priority ?? "polite"}
        role="status"
        aria-live={current?.priority ?? "polite"}
        aria-atomic="true"
        style={{
          position: "fixed",
          inlineSize: "1px",
          blockSize: "1px",
          overflow: "hidden",
          clipPath: "inset(50%)",
          whiteSpace: "nowrap",
        }}
      >
        {current?.message ?? ""}
      </div>
    </LiveAnnouncerContext.Provider>
  );
}

export const LiveAnnouncerProvider = LiveAnnouncer;

export function useLiveAnnouncer(): LiveAnnouncerHandle {
  return useContext(LiveAnnouncerContext) ?? getFallbackHandle();
}
