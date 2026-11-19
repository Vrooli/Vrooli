import { useEffect, useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useLocation, useNavigate } from "react-router-dom";
import { ChevronLeft, ChevronRight } from "lucide-react";

import { conversationsClient } from "../../api/notifications";
import { listViewRequest, readListView } from "../asks/askListView";
import { parseStateFilter } from "../asks/askState";
import { preserveRoutedTestMode } from "../routedTestMode";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/** Navigation follows the same filters and ordering as the list that opened the packet. */
export function usePacketNavigation(currentId: string, returnTo: string) {
  const navigate = useNavigate();
  const location = useLocation();
  const queue = !returnTo.startsWith("/history");
  const params = new URLSearchParams(returnTo.split("?")[1] ?? "");
  const list = useQuery({
    queryKey: ["asks", "packet-navigation", returnTo],
    retry: false,
    staleTime: 30_000,
    queryFn: async ({ signal }) => {
      const ids: string[] = [];
      let cursor = "";
      do {
        const page = await conversationsClient.listAsks({
          limit: 200, cursor, query: (params.get("q") ?? "").trim(),
          stateFilter: queue ? "awaiting" : parseStateFilter(params.get("state")),
          ...listViewRequest(readListView(params, queue), queue),
        }, { signal });
        ids.push(...page.asks.map((ask) => ask.id));
        cursor = page.nextCursor;
      } while (cursor);
      return [...new Set(ids)];
    },
  });
  const ids = list.data ?? [];
  const index = ids.indexOf(currentId);
  const move = (offset: number) => {
    const id = index < 0 ? undefined : ids[index + offset];
    if (id) navigate(preserveRoutedTestMode(`/asks/${encodeURIComponent(id)}`, location.search), { state: { from: returnTo } });
  };
  return { index, total: ids.length, previous: () => move(-1), next: () => move(1), hasPrevious: index > 0, hasNext: index >= 0 && index < ids.length - 1 };
}

export type PacketNavigationState = ReturnType<typeof usePacketNavigation>;

export function PacketNavigationButton({ navigation, direction }: { navigation: PacketNavigationState; direction: "previous" | "next" }) {
  const { t } = useTranslation();
  const previous = direction === "previous";
  return <button type="button" aria-label={t(previous ? strings.packet.navigation.previous : strings.packet.navigation.next)} disabled={previous ? !navigation.hasPrevious : !navigation.hasNext} onClick={previous ? navigation.previous : navigation.next} className="inline-flex min-h-11 min-w-11 items-center justify-center rounded-lg hover:bg-app-surface-muted disabled:opacity-40">{previous ? <ChevronLeft size={20} aria-hidden="true" /> : <ChevronRight size={20} aria-hidden="true" />}</button>;
}

/** Compact controls can be placed in the header and the phone action bar. */
export function PacketNavigation({ navigation, showPosition = true }: { navigation: PacketNavigationState; showPosition?: boolean }) {
  const { t } = useTranslation();
  return <nav aria-label={t(strings.packet.navigation.label)} className="inline-flex items-center gap-1">
    <PacketNavigationButton navigation={navigation} direction="previous" />
    {showPosition && <span className="whitespace-nowrap text-sm text-app-muted-foreground">{navigation.index >= 0 ? t(strings.packet.navigation.position, { current: navigation.index + 1, total: navigation.total }) : "—"}</span>}
    <PacketNavigationButton navigation={navigation} direction="next" />
  </nav>;
}

/** Horizontal gestures on the host page move only after a deliberate swipe. */
export function usePacketSwipe(navigation: PacketNavigationState) {
  const latest = useRef(navigation);
  latest.current = navigation;
  useEffect(() => {
    let start: { x: number; y: number } | null = null;
    const onStart = (event: TouchEvent) => {
      const target = event.target;
      if (event.touches.length !== 1 || (target instanceof Element && target.closest("input, textarea, select, [contenteditable=true], [data-no-packet-swipe]"))) { start = null; return; }
      const touch = event.touches[0];
      if (touch) start = { x: touch.clientX, y: touch.clientY };
    };
    const onEnd = (event: TouchEvent) => {
      if (!start || event.changedTouches.length !== 1) return;
      const touch = event.changedTouches[0];
      if (!touch) return;
      const dx = touch.clientX - start.x;
      const dy = touch.clientY - start.y;
      start = null;
      if (Math.abs(dx) < 80 || Math.abs(dx) < Math.abs(dy) * 1.5) return;
      if (dx > 0) latest.current.previous(); else latest.current.next();
    };
    document.addEventListener("touchstart", onStart, { passive: true });
    document.addEventListener("touchend", onEnd, { passive: true });
    const onFrameSwipe = (event: Event) => {
      const direction: unknown = (event as CustomEvent<unknown>).detail;
      if (direction === "previous") latest.current.previous();
      else if (direction === "next") latest.current.next();
    };
    window.addEventListener("notification-hub:packet-swipe", onFrameSwipe);
    return () => { document.removeEventListener("touchstart", onStart); document.removeEventListener("touchend", onEnd); window.removeEventListener("notification-hub:packet-swipe", onFrameSwipe); };
  }, []);
}
