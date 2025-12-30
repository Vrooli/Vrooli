import {
  FileJson,
  Settings,
  Shield,
  FileCode,
  Key,
} from "lucide-react";
import { cn } from "../../../lib/utils";

interface QuickAccessProps {
  deploymentId: string;
  onSelect: (path: string) => void;
  selectedPath: string | null;
}

// Common files to quick-access
const QUICK_ACCESS_FILES = [
  {
    id: "manifest",
    label: "manifest.json",
    path: ".vrooli/cloud/manifest.json",
    icon: FileJson,
    description: "Deployment manifest",
  },
  {
    id: "autoheal-scope",
    label: "autoheal-scope.json",
    path: ".vrooli/cloud/autoheal-scope.json",
    icon: Shield,
    description: "Autoheal configuration",
  },
  {
    id: "caddyfile",
    label: "Caddyfile",
    path: "/etc/caddy/Caddyfile",
    icon: Settings,
    description: "Caddy reverse proxy config",
  },
  {
    id: "service-json",
    label: "service.json",
    path: ".vrooli/service.json",
    icon: FileCode,
    description: "Vrooli service configuration",
  },
  {
    id: "env",
    label: ".env",
    path: ".env",
    icon: Key,
    description: "Environment variables",
  },
];

export function QuickAccess({ deploymentId, onSelect, selectedPath }: QuickAccessProps) {
  return (
    <div className="border border-white/10 rounded-lg bg-slate-900/50 p-4">
      <h3 className="text-sm font-medium text-slate-400 mb-3">Quick Access</h3>
      <div className="flex flex-wrap gap-2">
        {QUICK_ACCESS_FILES.map((file) => {
          const Icon = file.icon;
          const isSelected = selectedPath === file.path;

          return (
            <button
              key={file.id}
              onClick={() => onSelect(file.path)}
              title={file.description}
              className={cn(
                "flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm",
                "border transition-colors",
                isSelected
                  ? "bg-blue-500/20 border-blue-500/30 text-blue-400"
                  : "border-white/10 text-slate-300 hover:text-white hover:bg-white/5"
              )}
            >
              <Icon className="h-4 w-4" />
              {file.label}
            </button>
          );
        })}
      </div>
    </div>
  );
}
