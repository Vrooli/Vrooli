import { useState } from "react";
import {
  ChevronRight,
  ChevronDown,
  Check,
  X,
  Circle,
  AlertCircle,
  Square,
  CheckSquare
} from "lucide-react";
import { cn } from "../../lib/utils";
import { selectors } from "../../consts/selectors";
import type {
  ModuleSnapshot,
  RequirementItem,
  ValidationItem,
  RequirementImproveInfo
} from "../../lib/api";

interface RequirementsTreeProps {
  modules: ModuleSnapshot[];
  filter?: "all" | "passed" | "failed" | "not_run";
  searchQuery?: string;
  selectionMode?: boolean;
  selectedRequirements?: Map<string, RequirementImproveInfo>;
  onToggleSelection?: (
    req: RequirementItem & { modulePath: string }
  ) => void;
}

export function RequirementsTree({
  modules,
  filter = "all",
  searchQuery = "",
  selectionMode = false,
  selectedRequirements = new Map(),
  onToggleSelection
}: RequirementsTreeProps) {
  const [expandedModules, setExpandedModules] = useState<Set<string>>(
    new Set()
  );
  const [expandedReqs, setExpandedReqs] = useState<Set<string>>(new Set());

  const toggleModule = (name: string) => {
    setExpandedModules((prev) => {
      const next = new Set(prev);
      if (next.has(name)) {
        next.delete(name);
      } else {
        next.add(name);
      }
      return next;
    });
  };

  const toggleReq = (id: string) => {
    setExpandedReqs((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const filteredModules = modules
    .filter((module) => {
      if (searchQuery) {
        const query = searchQuery.toLowerCase();
        if (module.name.toLowerCase().includes(query)) return true;
        if (
          module.requirements?.some(
            (r) =>
              r.id.toLowerCase().includes(query) ||
              r.title.toLowerCase().includes(query)
          )
        )
          return true;
        return false;
      }
      return true;
    })
    .sort((a, b) => a.name.localeCompare(b.name));

  if (filteredModules.length === 0) {
    return (
      <div className="rounded-xl border border-white/10 bg-white/[0.02] p-6 text-center">
        <p className="text-slate-400">
          {modules.length === 0
            ? "No requirements found. Run tests to sync requirements."
            : "No requirements match your search."}
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-2" data-testid={selectors.requirements.tree}>
      {filteredModules.map((module) => (
        <ModuleRow
          key={module.name}
          module={module}
          isExpanded={expandedModules.has(module.name)}
          onToggle={() => toggleModule(module.name)}
          expandedReqs={expandedReqs}
          onToggleReq={toggleReq}
          filter={filter}
          searchQuery={searchQuery}
          selectionMode={selectionMode}
          selectedRequirements={selectedRequirements}
          onToggleSelection={onToggleSelection}
        />
      ))}
    </div>
  );
}

interface ModuleRowProps {
  module: ModuleSnapshot;
  isExpanded: boolean;
  onToggle: () => void;
  expandedReqs: Set<string>;
  onToggleReq: (id: string) => void;
  filter: "all" | "passed" | "failed" | "not_run";
  searchQuery: string;
  selectionMode: boolean;
  selectedRequirements: Map<string, RequirementImproveInfo>;
  onToggleSelection?: (
    req: RequirementItem & { modulePath: string }
  ) => void;
}

function ModuleRow({
  module,
  isExpanded,
  onToggle,
  expandedReqs,
  onToggleReq,
  filter,
  searchQuery,
  selectionMode,
  selectedRequirements,
  onToggleSelection
}: ModuleRowProps) {
  const { name, total, complete, completionRate } = module;
  const hasFailures = complete < total;

  // Determine module status icon
  const StatusIcon =
    completionRate >= 100 ? Check : hasFailures ? AlertCircle : Circle;
  const statusColor =
    completionRate >= 100
      ? "text-emerald-400"
      : hasFailures
        ? "text-amber-400"
        : "text-slate-400";

  return (
    <div className="overflow-hidden rounded-xl border border-white/10 bg-white/[0.02]">
      <button
        type="button"
        onClick={onToggle}
        className="flex w-full items-center gap-3 px-4 py-3 text-left transition hover:bg-white/[0.02]"
      >
        {isExpanded ? (
          <ChevronDown className="h-4 w-4 text-slate-400" />
        ) : (
          <ChevronRight className="h-4 w-4 text-slate-400" />
        )}
        <StatusIcon className={cn("h-4 w-4", statusColor)} />
        <span className="flex-1 font-medium">{name}</span>
        <span className="text-sm text-slate-400">
          {Math.round(completionRate)}%
        </span>
        <span
          className={cn(
            "rounded-full px-2 py-0.5 text-xs",
            completionRate >= 100
              ? "bg-emerald-500/20 text-emerald-300"
              : hasFailures
                ? "bg-amber-400/20 text-amber-200"
                : "bg-slate-500/20 text-slate-300"
          )}
        >
          {complete}/{total}
        </span>
      </button>

      {isExpanded && module.requirements && module.requirements.length > 0 && (
        <div className="border-t border-white/5 px-4 py-2">
          {module.requirements
            .filter((req) => {
              if (filter === "passed" && req.liveStatus !== "passed")
                return false;
              if (filter === "failed" && req.liveStatus !== "failed")
                return false;
              if (
                filter === "not_run" &&
                !["not_run", "skipped", "unknown"].includes(req.liveStatus)
              )
                return false;
              if (searchQuery) {
                const query = searchQuery.toLowerCase();
                return (
                  req.id.toLowerCase().includes(query) ||
                  req.title.toLowerCase().includes(query)
                );
              }
              return true;
            })
            .sort((a, b) =>
              a.id.localeCompare(b.id, undefined, { numeric: true })
            )
            .map((req) => (
              <RequirementRow
                key={req.id}
                requirement={req}
                modulePath={module.filePath}
                isExpanded={expandedReqs.has(req.id)}
                onToggle={() => onToggleReq(req.id)}
                selectionMode={selectionMode}
                isSelected={selectedRequirements.has(req.id)}
                onToggleSelection={onToggleSelection}
              />
            ))}
        </div>
      )}

      {isExpanded &&
        (!module.requirements || module.requirements.length === 0) && (
          <div className="border-t border-white/5 px-4 py-3 text-sm text-slate-500">
            No requirement details available
          </div>
        )}
    </div>
  );
}

interface RequirementRowProps {
  requirement: RequirementItem;
  modulePath: string;
  isExpanded: boolean;
  onToggle: () => void;
  selectionMode: boolean;
  isSelected: boolean;
  onToggleSelection?: (
    req: RequirementItem & { modulePath: string }
  ) => void;
}

function RequirementRow({
  requirement,
  modulePath,
  isExpanded,
  onToggle,
  selectionMode,
  isSelected,
  onToggleSelection
}: RequirementRowProps) {
  const { id, title, liveStatus, validations } = requirement;
  const hasValidations = validations && validations.length > 0;

  // Only improvable (failed or not_run) requirements can be selected
  const isImprovable = liveStatus === "failed" || liveStatus === "not_run";

  const LiveIcon =
    liveStatus === "passed" ? Check : liveStatus === "failed" ? X : Circle;
  const liveColor =
    liveStatus === "passed"
      ? "text-emerald-400"
      : liveStatus === "failed"
        ? "text-red-400"
        : "text-slate-500";

  const handleClick = () => {
    if (selectionMode && isImprovable && onToggleSelection) {
      onToggleSelection({ ...requirement, modulePath });
    } else if (!selectionMode) {
      onToggle();
    }
  };

  return (
    <div className="py-1">
      <button
        type="button"
        onClick={handleClick}
        disabled={!hasValidations && !selectionMode}
        className={cn(
          "flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left text-sm",
          selectionMode && isImprovable
            ? "hover:bg-violet-500/10"
            : hasValidations
              ? "hover:bg-white/[0.03]"
              : "cursor-default",
          selectionMode && isSelected && "bg-violet-500/10"
        )}
      >
        {/* Selection checkbox or expand chevron */}
        {selectionMode ? (
          isImprovable ? (
            isSelected ? (
              <CheckSquare className="h-4 w-4 text-violet-400" />
            ) : (
              <Square className="h-4 w-4 text-slate-500" />
            )
          ) : (
            <span className="h-4 w-4" />
          )
        ) : hasValidations ? (
          isExpanded ? (
            <ChevronDown className="h-3 w-3 text-slate-500" />
          ) : (
            <ChevronRight className="h-3 w-3 text-slate-500" />
          )
        ) : (
          <span className="w-3" />
        )}

        <LiveIcon className={cn("h-3.5 w-3.5", liveColor)} />
        <span className="flex-1 text-slate-200">
          <span className="text-slate-500">{id}</span>
          <span className="mx-2 text-slate-600">|</span>
          {title}
        </span>
        {hasValidations && !selectionMode && (
          <span className="text-xs text-slate-500">
            {validations.length} validation{validations.length !== 1 ? "s" : ""}
          </span>
        )}
      </button>

      {isExpanded && hasValidations && !selectionMode && (
        <div className="ml-6 mt-1 space-y-1 border-l border-white/5 pl-4">
          {validations.map((val, idx) => (
            <ValidationRow key={`${val.ref}-${idx}`} validation={val} />
          ))}
        </div>
      )}
    </div>
  );
}

interface ValidationRowProps {
  validation: ValidationItem;
}

function ValidationRow({ validation }: ValidationRowProps) {
  const { type, ref, phase, liveStatus } = validation;

  const LiveIcon =
    liveStatus === "passed" ? Check : liveStatus === "failed" ? X : Circle;
  const liveColor =
    liveStatus === "passed"
      ? "text-emerald-400"
      : liveStatus === "failed"
        ? "text-red-400"
        : "text-slate-500";

  return (
    <div className="flex items-center gap-2 py-1 text-xs">
      <LiveIcon className={cn("h-3 w-3", liveColor)} />
      <span className="rounded bg-white/5 px-1.5 py-0.5 text-slate-400">
        {type}
      </span>
      {phase && (
        <span className="rounded bg-purple-500/20 px-1.5 py-0.5 text-purple-300">
          {phase}
        </span>
      )}
      <span className="flex-1 truncate text-slate-300" title={ref}>
        {ref}
      </span>
    </div>
  );
}
