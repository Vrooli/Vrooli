import { useRef, useState } from "react";
import type { RefObject } from "react";
import { PanelRightClose, PanelRightOpen } from "lucide-react";
import { Button } from "../ui/button";
import { useResizablePanel } from "@vrooli/react-component-library/useResizablePanel/1.0.0";
import { ResizeHandle } from "@vrooli/react-component-library/ResizeHandle/1.0.0";
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
  const { separatorProps, panelProps } = useResizablePanel({
    containerRef,
    panelRef: inspectorRef,
    edge: "start",
    min: 280,
    max: 520,
    defaultSize: 340,
    adjacentMin: 480,
    storageKey: INSPECTOR_STORAGE_KEY,
    panelName: "Inspector",
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
      <aside
        ref={inspectorRef}
        {...panelProps}
        style={panelProps.style}
        className={cn(
          "relative flex h-full min-h-0 flex-col bg-slate-950/30",
          presentation === "card" && "rounded-lg border border-white/10 p-3",
          presentation === "pane" && "h-full border-l border-white/10 p-3",
        )}
        data-testid="session-inspector"
      >
        <ResizeHandle
          separatorProps={separatorProps}
          testId="session-inspector-resize-handle"
          className={cn(presentation === "card" && "my-1")}
        />
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
