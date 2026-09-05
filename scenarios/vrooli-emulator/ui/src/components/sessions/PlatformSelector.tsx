import { Monitor, Laptop, Smartphone, Terminal } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { Badge } from "../ui/badge";
import { cn } from "../../lib/utils";

interface PlatformOption {
  id: string;
  label: string;
  icon: LucideIcon;
  enabled: boolean;
}

const PLATFORMS: PlatformOption[] = [
  { id: "linux", label: "Linux", icon: Terminal, enabled: true },
  { id: "windows", label: "Windows", icon: Monitor, enabled: false },
  { id: "macos", label: "macOS", icon: Laptop, enabled: false },
  { id: "android", label: "Android", icon: Smartphone, enabled: false },
  { id: "ios", label: "iOS", icon: Smartphone, enabled: false },
];

interface PlatformSelectorProps {
  value: string;
  onChange: (platform: string) => void;
}

export function PlatformSelector({ value, onChange }: PlatformSelectorProps) {
  return (
    <div className="grid grid-cols-3 gap-2">
      {PLATFORMS.map((platform) => {
        const Icon = platform.icon;
        const isSelected = value === platform.id;
        const isDisabled = !platform.enabled;

        return (
          <button
            key={platform.id}
            type="button"
            disabled={isDisabled}
            onClick={() => onChange(platform.id)}
            className={cn(
              "relative flex flex-col items-center gap-1.5 rounded-lg border p-3 transition-colors",
              isSelected && "border-blue-500 bg-blue-950/30 ring-1 ring-blue-500/30",
              !isSelected && !isDisabled && "border-slate-700 bg-slate-900/50 hover:border-slate-600 cursor-pointer",
              isDisabled && "border-slate-800 bg-slate-950/50 opacity-50 cursor-not-allowed",
            )}
          >
            <Icon className={cn("h-5 w-5", isSelected ? "text-blue-400" : "text-slate-400")} />
            <span className={cn("text-xs font-medium", isSelected ? "text-blue-300" : "text-slate-300")}>
              {platform.label}
            </span>
            {isDisabled && (
              <Badge
                variant="outline"
                className="absolute -top-1.5 -right-1.5 text-[9px] px-1 py-0 border-slate-600 text-slate-400 bg-slate-900"
              >
                Soon
              </Badge>
            )}
          </button>
        );
      })}
    </div>
  );
}
