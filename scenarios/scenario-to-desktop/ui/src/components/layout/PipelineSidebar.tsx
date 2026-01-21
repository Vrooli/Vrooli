/**
 * Collapsible sidebar containing pipeline info and section navigation.
 */

import { ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "../ui/button";
import { SidebarHeader } from "./SidebarHeader";
import { SidebarNavigation } from "./SidebarNavigation";
import { useSidebarStore, type SectionId } from "../../store/sidebarStore";
import { cn } from "../../lib/utils";

interface PipelineSidebarProps {
  /** Callback when a section is clicked */
  onSectionClick: (section: SectionId) => void;
}

export function PipelineSidebar({ onSectionClick }: PipelineSidebarProps) {
  const collapsed = useSidebarStore((s) => s.collapsed);
  const toggleCollapsed = useSidebarStore((s) => s.toggleCollapsed);

  return (
    <aside
      className={cn(
        "relative flex h-screen flex-col border-r border-white/10 bg-slate-950/50 transition-all duration-300",
        collapsed ? "w-14" : "w-72"
      )}
    >
      {/* Scrollable content */}
      <div className="flex-1 overflow-y-auto">
        {!collapsed && (
          <>
            <SidebarHeader />
            <SidebarNavigation onSectionClick={onSectionClick} />
          </>
        )}
        {collapsed && (
          <div className="flex flex-col items-center gap-2 py-4">
            <SidebarNavigation onSectionClick={onSectionClick} collapsed />
          </div>
        )}
      </div>

      {/* Collapse toggle button */}
      <div className="border-t border-white/10 p-2">
        <Button
          variant="ghost"
          size="sm"
          onClick={toggleCollapsed}
          className={cn(
            "w-full justify-center text-slate-400 hover:text-slate-200",
            collapsed && "px-0"
          )}
        >
          {collapsed ? (
            <ChevronRight className="h-4 w-4" />
          ) : (
            <>
              <ChevronLeft className="h-4 w-4 mr-2" />
              Collapse
            </>
          )}
        </Button>
      </div>
    </aside>
  );
}
