import { List, Settings, Sparkles, Plus } from "lucide-react";
import { useDraggablePosition } from "../hooks/useDraggablePosition";
import { useLongPress } from "../hooks/useLongPress";
import { Button } from "./ui/button";

interface FloatingToolbarProps {
  onOpenSessions: () => void;
  onOpenSettings: () => void;
  onOpenAi: () => void;
  onNewTerminal: () => void;
  onOpenLauncher: () => void;
  isCreating: boolean;
}

export default function FloatingToolbar({
  onOpenSessions,
  onOpenSettings,
  onOpenAi,
  onNewTerminal,
  onOpenLauncher,
  isCreating,
}: FloatingToolbarProps) {
  const { elementRef, floatingStyle, pointerHandlers, handleClickCapture } =
    useDraggablePosition({
      isActive: true,
      storageKey: "wc-toolbar-pos",
      defaultPosition: () => {
        if (typeof window === "undefined") return { x: 100, y: 12 };
        return { x: window.innerWidth - 180, y: 12 };
      },
    });

  const plusHandlers = useLongPress({
    onPress: onNewTerminal,
    onLongPress: onOpenLauncher,
  });

  return (
    <div
      ref={(node) => { elementRef.current = node; }}
      data-testid="floating-toolbar"
      className="fixed left-0 top-0 z-[2600] flex items-center gap-1 rounded-full border border-wc-default bg-wc-surface-raised/95 backdrop-blur-md shadow-lg px-2 py-1 cursor-grab active:cursor-grabbing select-none touch-none"
      style={floatingStyle}
      onPointerDown={pointerHandlers.onPointerDown}
      onPointerMove={pointerHandlers.onPointerMove}
      onPointerUp={pointerHandlers.onPointerUp}
      onPointerCancel={pointerHandlers.onPointerCancel}
      onClickCapture={handleClickCapture}
    >
      <Button
        data-testid="toolbar-sessions"
        variant="ghost"
        size="icon"
        className="h-7 w-7"
        onClick={onOpenSessions}
        title="Sessions"
      >
        <List className="h-4 w-4" />
      </Button>
      <Button
        data-testid="toolbar-settings"
        variant="ghost"
        size="icon"
        className="h-7 w-7"
        onClick={onOpenSettings}
        title="Settings"
      >
        <Settings className="h-4 w-4" />
      </Button>
      <Button
        data-testid="toolbar-ai"
        variant="ghost"
        size="icon"
        className="h-7 w-7"
        onClick={onOpenAi}
        title="AI Command"
      >
        <Sparkles className="h-4 w-4" />
      </Button>
      <Button
        data-testid="toolbar-new"
        variant="ghost"
        size="icon"
        className="h-7 w-7"
        disabled={isCreating}
        title="New terminal (long-press for launcher)"
        onPointerDown={plusHandlers.onPointerDown}
        onPointerUp={plusHandlers.onPointerUp}
        onPointerCancel={plusHandlers.onPointerCancel}
        onContextMenu={plusHandlers.onContextMenu}
      >
        <Plus className="h-4 w-4" />
      </Button>
    </div>
  );
}
