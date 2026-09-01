import { useEffect } from "react";
import type { NavigateFunction } from "react-router-dom";
import type { BoardIntent } from "../components/BoardController";
import { emitShortcutIntent } from "@vrooli/iframe-bridge";

interface BoardRoom { id: string }

export function useBoardKeyboard({ rooms, search, navigate, dispatch }: { rooms: BoardRoom[]; search: string; navigate: NavigateFunction; dispatch: (intent: BoardIntent) => void }) {
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      const key = event.key.toLowerCase();
      const map: Record<string, BoardIntent> = { arrowright: "navigate-right", arrowleft: "navigate-left", arrowup: "navigate-up", arrowdown: "navigate-down", " ": "pause-cycle", f: "menu", "?": "show-help", enter: "inspect" };
      if (/^[1-9]$/.test(key)) {
        const room = rooms[Number(key) - 1];
        if (room) navigate(`/${room.id}${search}`);
      } else if (map[key]) {
        event.preventDefault();
        emitShortcutIntent({ action: `command-center.${map[key]}`, outcome: "handled", chord: event.key, source: "keyboard" });
        dispatch(map[key]);
      }
    };
    const onPointerMove = () => dispatch("reveal-controls");
    window.addEventListener("keydown", onKey);
    window.addEventListener("pointermove", onPointerMove, { passive: true });
    return () => { window.removeEventListener("keydown", onKey); window.removeEventListener("pointermove", onPointerMove); };
  }, [dispatch, navigate, rooms, search]);
}
