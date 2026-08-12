import { useRef, useState } from "react";
import type { RefObject } from "react";
import { PanelRightClose, PanelRightOpen } from "lucide-react";
import { Button } from "../ui/button";
import { useResizablePanel } from "../../hooks/useResizablePanel";
import { cn } from "../../lib/utils";
import type { SessionInspectorSection } from "./session-view-model";
import type { SessionSectionConfig } from "./SessionSectionTabs";
import { SessionSectionTabs } from "./SessionSectionTabs";

const INSPECTOR_STORAGE_KEY = "swarm-manager.session-inspector.width.v1";

interface SessionInspectorProps {
  containerRef: RefObject<HTMLElement | null>;
  sections: SessionSectionConfig[];
  defaultSection: SessionInspectorSection;
  isCollapsed: boolean;
  onCollapsedChange: (isCollapsed: boolean) => void;
  presentation?: "card" | "pane";
}

export function SessionInspector({
  containerRef,
  sections,
  defaultSection,
  isCollapsed,
  onCollapsedChange,
  presentation = "card",
}: SessionInspectorProps) {
  const inspectorRef = useRef<HTMLElement>(null);
  const [activeSection, setActiveSection] = useState<SessionInspectorSection>(defaultSection);
  const { size, isResizing, resizeHandleProps } = useResizablePanel({
    containerRef,
    targetRef: inspectorRef,
    minSize: 280,
    maxSize: 520,
    defaultSize: 340,
    adjacentMinSize: 480,
    handleWidth: 6,
    storageKey: INSPECTOR_STORAGE_KEY,
    resizeEdge: "left",
  });

  if (isCollapsed) {
    return (
      <div className="flex shrink-0 items-start">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onCollapsedChange(false)}
          data-testid="session-inspector-expand"
          aria-label="Open session inspector"
        >
          <PanelRightOpen className="mr-2 h-4 w-4" />
          Inspector
        </Button>
      </div>
    );
  }

  return (
    <>
      <div
        {...resizeHandleProps}
        className={cn(
          "w-1.5 shrink-0 cursor-col-resize bg-transparent transition-colors hover:bg-cyan-500/30",
          presentation === "card" && "my-1 rounded-full",
          isResizing && "bg-cyan-500/40",
        )}
        data-testid="session-inspector-resize-handle"
      />
      <aside
        ref={inspectorRef}
        style={{ width: size }}
        className={cn(
          "flex h-full min-h-0 flex-col bg-slate-950/30",
          presentation === "card" && "rounded-lg border border-white/10 p-3",
          presentation === "pane" && "h-full border-l border-white/10 p-3",
        )}
        data-testid="session-inspector"
      >
        <div className="mb-3 flex items-center justify-between gap-2">
          <h3 className="text-xs font-medium uppercase tracking-wide text-slate-400">Inspector</h3>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => onCollapsedChange(true)}
            data-testid="session-inspector-collapse"
            aria-label="Close session inspector"
          >
            <PanelRightClose className="h-4 w-4" />
          </Button>
        </div>
        <SessionSectionTabs
          sections={sections}
          activeValue={activeSection}
          onValueChange={(value) => setActiveSection(value as SessionInspectorSection)}
          listLabel="Session inspector sections"
          className="min-h-0 flex-1"
          tabBarClassName="border-y border-slate-200/20"
          contentClassName="min-h-0 overflow-y-auto pt-3"
        />
      </aside>
    </>
  );
}
