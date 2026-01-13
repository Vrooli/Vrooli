/**
 * IconSelector - Popover grid for selecting icons.
 *
 * Shows current icon as trigger, opens a grid popover to select from available icons.
 * Each icon shows the actual icon visual + name.
 */

import { useState, useRef, useEffect, useCallback, type ComponentType, type SVGProps } from "react";
import { ChevronDown } from "lucide-react";
import * as LucideIcons from "lucide-react";
import { cn } from "@/lib/utils";

// Type for Lucide icon components
type IconComponent = ComponentType<SVGProps<SVGSVGElement> & { className?: string }>;

interface IconSelectorProps {
  value: string;
  onChange: (value: string) => void;
  icons: string[];
  disabled?: boolean;
  className?: string;
}

// Get icon component from name
function getIconComponent(name: string): IconComponent | null {
  const Icon = (LucideIcons as unknown as Record<string, IconComponent>)[name];
  return Icon || null;
}

export function IconSelector({
  value,
  onChange,
  icons,
  disabled,
  className,
}: IconSelectorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const handleClickOutside = useCallback((event: MouseEvent) => {
    if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
      setIsOpen(false);
    }
  }, []);

  const handleEscape = useCallback((event: KeyboardEvent) => {
    if (event.key === "Escape") {
      setIsOpen(false);
    }
  }, []);

  useEffect(() => {
    if (isOpen) {
      document.addEventListener("mousedown", handleClickOutside);
      document.addEventListener("keydown", handleEscape);
    }
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
      document.removeEventListener("keydown", handleEscape);
    };
  }, [isOpen, handleClickOutside, handleEscape]);

  const handleSelect = useCallback(
    (iconName: string) => {
      onChange(iconName);
      setIsOpen(false);
    },
    [onChange]
  );

  const CurrentIcon = getIconComponent(value);

  return (
    <div ref={containerRef} className={cn("relative", className)}>
      {/* Trigger button */}
      <button
        type="button"
        onClick={() => !disabled && setIsOpen(!isOpen)}
        disabled={disabled}
        className={cn(
          "flex items-center gap-2 px-3 py-2 w-full bg-slate-800 border border-white/10 rounded-lg text-white transition-colors",
          "hover:bg-slate-700 focus:outline-none focus:ring-2 focus:ring-indigo-500",
          disabled && "opacity-50 cursor-not-allowed"
        )}
      >
        {CurrentIcon && <CurrentIcon className="h-4 w-4 text-slate-300" />}
        <span className="flex-1 text-left text-sm">{value}</span>
        <ChevronDown
          className={cn(
            "h-4 w-4 text-slate-400 transition-transform",
            isOpen && "rotate-180"
          )}
        />
      </button>

      {/* Popover grid */}
      {isOpen && (
        <div
          className={cn(
            "absolute z-50 mt-1 w-full min-w-[280px] max-h-[320px] overflow-y-auto",
            "bg-slate-900 border border-white/10 rounded-lg shadow-xl p-2",
            "animate-in fade-in-0 zoom-in-95 duration-100"
          )}
        >
          <div className="grid grid-cols-4 gap-1">
            {icons.map((iconName) => {
              const Icon = getIconComponent(iconName);
              const isSelected = iconName === value;

              return (
                <button
                  key={iconName}
                  type="button"
                  onClick={() => handleSelect(iconName)}
                  className={cn(
                    "flex flex-col items-center gap-1 p-2 rounded-lg transition-colors",
                    isSelected
                      ? "bg-indigo-600/30 ring-1 ring-indigo-500"
                      : "hover:bg-white/5"
                  )}
                  title={iconName}
                >
                  {Icon && (
                    <Icon
                      className={cn(
                        "h-5 w-5",
                        isSelected ? "text-indigo-300" : "text-slate-300"
                      )}
                    />
                  )}
                  <span
                    className={cn(
                      "text-[10px] truncate w-full text-center",
                      isSelected ? "text-indigo-200" : "text-slate-500"
                    )}
                  >
                    {iconName}
                  </span>
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}

// Export icon options that can be reused
export const SKILL_ICON_OPTIONS = [
  "BookOpen",
  "Brain",
  "Building2",
  "Code",
  "Database",
  "FileCode",
  "FlaskConical",
  "Gauge",
  "GraduationCap",
  "Layout",
  "Lightbulb",
  "Package",
  "Puzzle",
  "Route",
  "Search",
  "Server",
  "Shield",
  "Sparkles",
  "Target",
  "Terminal",
  "Wand2",
  "Wrench",
  "Zap",
];

export const TEMPLATE_ICON_OPTIONS = [
  "Sparkles",
  "Bug",
  "Search",
  "RefreshCw",
  "FlaskConical",
  "GraduationCap",
  "Eye",
  "FolderTree",
  "Package",
  "Lightbulb",
  "Gauge",
  "FileType",
  "Server",
  "Layout",
  "Wand2",
  "Building2",
  "ArrowRightLeft",
  "Puzzle",
  "Route",
  "MessageCircleQuestion",
  "Shield",
  "Accessibility",
  "FileCode",
  "Terminal",
  "Database",
  "Cloud",
  "Zap",
];
