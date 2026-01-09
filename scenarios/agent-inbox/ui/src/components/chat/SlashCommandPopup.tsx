/**
 * SlashCommandPopup - Autocomplete popup for slash commands.
 *
 * Appears when user types "/" in the message input.
 * Supports keyboard navigation (up/down arrows, enter, escape).
 */
import { useEffect, useState, useRef } from "react";
import {
  FileText,
  BookOpen,
  Wrench,
  Globe,
  Sparkles,
  Code,
  Building2,
  TestTube,
  type LucideIcon,
} from "lucide-react";
import type { SlashCommand } from "@/lib/types/templates";

// Map icon names to Lucide components
const ICON_MAP: Record<string, LucideIcon> = {
  FileTemplate: FileText,
  FileText,
  BookOpen,
  Wrench,
  Globe,
  Sparkles,
  Code,
  Building2,
  TestTube,
};

// Type to icon and color mapping
const TYPE_STYLES: Record<
  string,
  { icon: LucideIcon; color: string; bg: string }
> = {
  template: { icon: FileText, color: "text-blue-400", bg: "bg-blue-500/20" },
  skill: { icon: BookOpen, color: "text-amber-400", bg: "bg-amber-500/20" },
  tool: { icon: Wrench, color: "text-violet-400", bg: "bg-violet-500/20" },
  search: { icon: Globe, color: "text-green-400", bg: "bg-green-500/20" },
  "direct-template": {
    icon: FileText,
    color: "text-blue-400",
    bg: "bg-blue-500/20",
  },
  "direct-skill": {
    icon: BookOpen,
    color: "text-amber-400",
    bg: "bg-amber-500/20",
  },
};

interface SlashCommandPopupProps {
  commands: SlashCommand[];
  selectedIndex: number;
  onSelect: (command: SlashCommand) => void;
  onClose: () => void;
  position: { bottom: number; left: number };
}

export function SlashCommandPopup({
  commands,
  selectedIndex,
  onSelect,
  onClose,
  position,
}: SlashCommandPopupProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const selectedRef = useRef<HTMLButtonElement>(null);

  // Scroll selected item into view
  useEffect(() => {
    if (selectedRef.current) {
      selectedRef.current.scrollIntoView({
        block: "nearest",
        behavior: "smooth",
      });
    }
  }, [selectedIndex]);

  // Close on click outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (
        containerRef.current &&
        !containerRef.current.contains(e.target as Node)
      ) {
        onClose();
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [onClose]);

  // Handle Escape key with capture to prevent closing chat
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        e.stopPropagation();
        onClose();
      }
    };

    document.addEventListener("keydown", handleEscape, { capture: true });
    return () => document.removeEventListener("keydown", handleEscape, { capture: true });
  }, [onClose]);

  if (commands.length === 0) {
    return (
      <div
        ref={containerRef}
        className="absolute z-50 w-72 bg-slate-900 border border-white/10 rounded-lg shadow-xl overflow-hidden"
        style={{ bottom: position.bottom, left: position.left }}
        data-testid="slash-command-popup"
      >
        <div className="px-3 py-4 text-sm text-slate-400 text-center">
          No commands found
        </div>
      </div>
    );
  }

  // Group commands by category
  const groupedCommands: { category: string; items: SlashCommand[] }[] = [];
  let currentCategory = "";
  let currentGroup: SlashCommand[] = [];

  for (const cmd of commands) {
    const category = cmd.category || "";
    if (category !== currentCategory) {
      if (currentGroup.length > 0) {
        groupedCommands.push({ category: currentCategory, items: currentGroup });
      }
      currentCategory = category;
      currentGroup = [cmd];
    } else {
      currentGroup.push(cmd);
    }
  }
  if (currentGroup.length > 0) {
    groupedCommands.push({ category: currentCategory, items: currentGroup });
  }

  let globalIndex = 0;

  return (
    <div
      ref={containerRef}
      className="absolute z-50 w-72 max-h-64 bg-slate-900 border border-white/10 rounded-lg shadow-xl overflow-y-auto"
      style={{ bottom: position.bottom, left: position.left }}
      data-testid="slash-command-popup"
    >
      {groupedCommands.map((group) => (
        <div key={group.category || "main"}>
          {group.category && (
            <div className="px-3 py-1.5 text-xs font-medium text-slate-500 uppercase tracking-wider bg-slate-800/50 sticky top-0">
              {group.category}
            </div>
          )}
          {group.items.map((command) => {
            const index = globalIndex++;
            const isSelected = index === selectedIndex;
            const style = TYPE_STYLES[command.type] || TYPE_STYLES.template;
            const IconComponent = command.icon
              ? ICON_MAP[command.icon] || style.icon
              : style.icon;

            return (
              <button
                key={command.id}
                ref={isSelected ? selectedRef : null}
                onClick={() => onSelect(command)}
                className={`
                  w-full flex items-center gap-3 px-3 py-2 text-left transition-colors
                  ${isSelected ? "bg-slate-800" : "hover:bg-slate-800/50"}
                `}
                data-testid={`slash-command-${command.id}`}
              >
                <div className={`p-1.5 rounded ${style.bg}`}>
                  <IconComponent className={`h-4 w-4 ${style.color}`} />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium text-white">
                    {command.name}
                  </div>
                  <div className="text-xs text-slate-400 truncate">
                    {command.description}
                  </div>
                </div>
              </button>
            );
          })}
        </div>
      ))}
    </div>
  );
}
