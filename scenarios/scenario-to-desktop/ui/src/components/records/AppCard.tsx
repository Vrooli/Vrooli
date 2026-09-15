import { type KeyboardEvent } from "react";
import {
  Monitor,
  ChevronRight,
  CheckCircle2,
  AlertCircle,
  Clock,
} from "lucide-react";
import { Card } from "../ui/card";
import { Badge } from "../ui/badge";
import { SigningBadge } from "./SigningBadge";
import type { DesktopRecordItemView } from "./recordPresentation";

type RecordItem = DesktopRecordItemView;

interface AppCardProps {
  item: RecordItem;
  onClick: () => void;
}

function BuildBadge({ state }: { state?: string }) {
  if (!state) return null;
  if (state === "ready") {
    return (
      <Badge variant="success" className="gap-1">
        <CheckCircle2 className="h-3 w-3" />
        Ready
      </Badge>
    );
  }
  if (state === "building") {
    return (
      <Badge variant="warning" className="gap-1">
        <Clock className="h-3 w-3" />
        Building
      </Badge>
    );
  }
  if (state === "failed") {
    return (
      <Badge variant="destructive" className="gap-1">
        <AlertCircle className="h-3 w-3" />
        Failed
      </Badge>
    );
  }
  return (
    <Badge variant="outline" className="capitalize">
      {state}
    </Badge>
  );
}

export function AppCard({ item, onClick }: AppCardProps) {
  const rec = item.record;
  const handleKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      onClick();
    }
  };

  return (
    <>
      {/* Desktop card — hidden on mobile */}
      <Card
        className="relative hidden cursor-pointer overflow-hidden border-slate-800/80 bg-slate-950/80 shadow-xl shadow-blue-950/30 backdrop-blur transition-all hover:border-blue-600/60 focus-within:border-blue-600/60 md:flex md:flex-col md:min-h-[140px]"
        role="button"
        tabIndex={0}
        onClick={onClick}
        onKeyDown={handleKeyDown}
      >
        <div className="absolute inset-x-0 top-0 h-1 bg-gradient-to-r from-blue-500 via-cyan-400 to-purple-500" />
        <div className="flex flex-1 items-start gap-3 p-4">
          <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-blue-600/20 to-cyan-600/20 border border-blue-700/30">
            <Monitor className="h-5 w-5 text-blue-300" />
          </div>
          <div className="min-w-0 flex-1 space-y-1.5">
            <h4 className="text-base font-semibold text-slate-50 truncate">
              {rec.app_display_name || rec.scenario_name}
            </h4>
            <p className="text-xs text-slate-400 truncate">
              {rec.scenario_name}
            </p>
            <div className="flex flex-wrap items-center gap-1.5">
              <BuildBadge
                state={item.build_state || item.build_status?.status}
              />
              <SigningBadge scenarioName={rec.scenario_name} />
            </div>
          </div>
          <ChevronRight className="h-5 w-5 shrink-0 text-slate-600" />
        </div>
      </Card>

      {/* Mobile list item — hidden on desktop */}
      <div
        className="flex cursor-pointer items-center gap-3 border-b border-slate-800 py-3 px-4 transition-colors active:bg-slate-900/50 md:hidden"
        role="button"
        tabIndex={0}
        onClick={onClick}
        onKeyDown={handleKeyDown}
      >
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-gradient-to-br from-blue-600/20 to-cyan-600/20 border border-blue-700/30">
          <Monitor className="h-4 w-4 text-blue-300" />
        </div>
        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium text-slate-100 truncate">
            {rec.app_display_name || rec.scenario_name}
          </p>
          <p className="text-xs text-slate-400 truncate">{rec.scenario_name}</p>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <BuildBadge state={item.build_state || item.build_status?.status} />
          <ChevronRight className="h-4 w-4 text-slate-600" />
        </div>
      </div>
    </>
  );
}
