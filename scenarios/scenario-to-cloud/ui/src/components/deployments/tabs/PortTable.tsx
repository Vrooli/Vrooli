import { CheckCircle2, AlertTriangle, Network } from "lucide-react";
import type { PortBinding } from "../../../lib/api";
import { cn } from "../../../lib/utils";

interface PortTableProps {
  ports: PortBinding[];
}

export function PortTable({ ports }: PortTableProps) {
  // Sort ports by number
  const sortedPorts = [...ports].sort((a, b) => a.port - b.port);

  // Group by type for summary
  const typeCounts = ports.reduce(
    (acc, port) => {
      acc[port.type] = (acc[port.type] || 0) + 1;
      return acc;
    },
    {} as Record<string, number>
  );

  return (
    <div className="space-y-4">
      {/* Summary badges */}
      <div className="flex flex-wrap gap-2">
        {Object.entries(typeCounts).map(([type, count]) => (
          <TypeBadge key={type} type={type as PortBinding["type"]} count={count} />
        ))}
      </div>

      {/* Ports table */}
      <div className="border border-white/10 rounded-lg overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-slate-800/50">
            <tr>
              <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">
                Port
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">
                Type
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">
                Process
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">
                Status
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-white/5">
            {sortedPorts.length === 0 ? (
              <tr>
                <td colSpan={4} className="px-4 py-8 text-center text-slate-500">
                  <div className="flex flex-col items-center gap-2">
                    <Network className="h-8 w-8 text-slate-600" />
                    <span>No listening ports detected</span>
                  </div>
                </td>
              </tr>
            ) : (
              sortedPorts.map((port) => (
                <PortRow key={port.port} port={port} />
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

interface TypeBadgeProps {
  type: PortBinding["type"];
  count: number;
}

function TypeBadge({ type, count }: TypeBadgeProps) {
  const config = getTypeConfig(type);
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium",
        config.bgClass,
        config.textClass
      )}
    >
      {config.label}: {count}
    </span>
  );
}

interface PortRowProps {
  port: PortBinding;
}

function PortRow({ port }: PortRowProps) {
  const config = getTypeConfig(port.type);

  return (
    <tr className="hover:bg-white/5 transition-colors">
      <td className="px-4 py-3">
        <span className="font-mono text-white">{port.port}</span>
      </td>
      <td className="px-4 py-3">
        <span
          className={cn(
            "inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium",
            config.bgClass,
            config.textClass
          )}
        >
          {config.label}
        </span>
      </td>
      <td className="px-4 py-3">
        <span className="text-slate-300">{port.process || "-"}</span>
        {port.pid && (
          <span className="text-slate-500 ml-2 text-xs">(PID: {port.pid})</span>
        )}
      </td>
      <td className="px-4 py-3">
        {port.matches_manifest === true ? (
          <span className="inline-flex items-center gap-1 text-emerald-400 text-xs">
            <CheckCircle2 className="h-3 w-3" />
            Expected
          </span>
        ) : port.matches_manifest === false ? (
          <span className="inline-flex items-center gap-1 text-amber-400 text-xs">
            <AlertTriangle className="h-3 w-3" />
            Unexpected
          </span>
        ) : (
          <span className="text-slate-500 text-xs">-</span>
        )}
      </td>
    </tr>
  );
}

function getTypeConfig(type: PortBinding["type"]) {
  switch (type) {
    case "system":
      return {
        label: "System",
        bgClass: "bg-slate-500/20",
        textClass: "text-slate-400",
      };
    case "edge":
      return {
        label: "Edge",
        bgClass: "bg-blue-500/20",
        textClass: "text-blue-400",
      };
    case "scenario":
      return {
        label: "Scenario",
        bgClass: "bg-emerald-500/20",
        textClass: "text-emerald-400",
      };
    case "resource":
      return {
        label: "Resource",
        bgClass: "bg-purple-500/20",
        textClass: "text-purple-400",
      };
    case "unexpected":
      return {
        label: "Unexpected",
        bgClass: "bg-amber-500/20",
        textClass: "text-amber-400",
      };
    default:
      return {
        label: type,
        bgClass: "bg-slate-500/20",
        textClass: "text-slate-400",
      };
  }
}
