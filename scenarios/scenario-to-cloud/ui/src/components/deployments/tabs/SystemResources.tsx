import { Cpu, HardDrive, Database, Clock } from "lucide-react";
import type { SystemState } from "../../../lib/api";
import { formatUptime, formatMB } from "../../../hooks/useLiveState";
import { cn } from "../../../lib/utils";

interface SystemResourcesProps {
  system: SystemState;
}

export function SystemResources({ system }: SystemResourcesProps) {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
      {/* CPU */}
      <ResourceCard
        icon={Cpu}
        title="CPU"
        subtitle={system.cpu.model || `${system.cpu.cores} cores`}
      >
        <div className="space-y-3">
          <ProgressBar
            label="Usage"
            value={system.cpu.usage_percent}
            max={100}
            color="blue"
          />
          {system.cpu.load_average.length > 0 && (
            <div className="flex items-center justify-between text-xs">
              <span className="text-slate-500">Load Average:</span>
              <span className="text-slate-300 font-mono">
                {system.cpu.load_average.map((l) => l.toFixed(2)).join(" / ")}
              </span>
            </div>
          )}
          <div className="flex items-center justify-between text-xs">
            <span className="text-slate-500">Cores:</span>
            <span className="text-slate-300">{system.cpu.cores}</span>
          </div>
        </div>
      </ResourceCard>

      {/* Memory */}
      <ResourceCard
        icon={Database}
        title="Memory"
        subtitle={formatMB(system.memory.total_mb)}
      >
        <div className="space-y-3">
          <ProgressBar
            label="Usage"
            value={system.memory.usage_percent}
            max={100}
            color={system.memory.usage_percent > 80 ? "red" : "emerald"}
          />
          <div className="flex items-center justify-between text-xs">
            <span className="text-slate-500">Used:</span>
            <span className="text-slate-300">
              {formatMB(system.memory.used_mb)} / {formatMB(system.memory.total_mb)}
            </span>
          </div>
          <div className="flex items-center justify-between text-xs">
            <span className="text-slate-500">Free:</span>
            <span className="text-slate-300">{formatMB(system.memory.free_mb)}</span>
          </div>
        </div>
      </ResourceCard>

      {/* Disk */}
      <ResourceCard
        icon={HardDrive}
        title="Disk"
        subtitle={`${system.disk.total_gb} GB total`}
      >
        <div className="space-y-3">
          <ProgressBar
            label="Usage"
            value={system.disk.usage_percent}
            max={100}
            color={system.disk.usage_percent > 80 ? "red" : "purple"}
          />
          <div className="flex items-center justify-between text-xs">
            <span className="text-slate-500">Used:</span>
            <span className="text-slate-300">
              {system.disk.used_gb} GB / {system.disk.total_gb} GB
            </span>
          </div>
          <div className="flex items-center justify-between text-xs">
            <span className="text-slate-500">Free:</span>
            <span className="text-slate-300">{system.disk.free_gb} GB</span>
          </div>
        </div>
      </ResourceCard>

      {/* Swap & Uptime */}
      <ResourceCard icon={Clock} title="System" subtitle="Swap & Uptime">
        <div className="space-y-3">
          {system.swap.total_mb > 0 ? (
            <>
              <ProgressBar
                label="Swap Usage"
                value={system.swap.usage_percent}
                max={100}
                color={system.swap.usage_percent > 50 ? "amber" : "slate"}
              />
              <div className="flex items-center justify-between text-xs">
                <span className="text-slate-500">Swap:</span>
                <span className="text-slate-300">
                  {system.swap.used_mb} MB / {system.swap.total_mb} MB
                </span>
              </div>
            </>
          ) : (
            <div className="flex items-center justify-between text-xs">
              <span className="text-slate-500">Swap:</span>
              <span className="text-slate-400">Not configured</span>
            </div>
          )}
          <div className="flex items-center justify-between text-xs">
            <span className="text-slate-500">Uptime:</span>
            <span className="text-slate-300">{formatUptime(system.uptime_seconds)}</span>
          </div>
        </div>
      </ResourceCard>
    </div>
  );
}

interface ResourceCardProps {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  subtitle: string;
  children: React.ReactNode;
}

function ResourceCard({ icon: Icon, title, subtitle, children }: ResourceCardProps) {
  return (
    <div className="border border-white/10 rounded-lg bg-slate-900/50 p-4">
      <div className="flex items-center gap-3 mb-4">
        <div className="p-2 rounded-lg bg-slate-800">
          <Icon className="h-5 w-5 text-slate-400" />
        </div>
        <div>
          <h4 className="font-medium text-white">{title}</h4>
          <p className="text-xs text-slate-500">{subtitle}</p>
        </div>
      </div>
      {children}
    </div>
  );
}

interface ProgressBarProps {
  label: string;
  value: number;
  max: number;
  color: "blue" | "emerald" | "purple" | "red" | "amber" | "slate";
}

function ProgressBar({ label, value, max, color }: ProgressBarProps) {
  const percentage = Math.min((value / max) * 100, 100);

  const colorClasses = {
    blue: "bg-blue-500",
    emerald: "bg-emerald-500",
    purple: "bg-purple-500",
    red: "bg-red-500",
    amber: "bg-amber-500",
    slate: "bg-slate-500",
  };

  return (
    <div>
      <div className="flex items-center justify-between text-xs mb-1">
        <span className="text-slate-500">{label}</span>
        <span className="text-slate-300">{value.toFixed(1)}%</span>
      </div>
      <div className="h-2 bg-slate-800 rounded-full overflow-hidden">
        <div
          className={cn("h-full rounded-full transition-all", colorClasses[color])}
          style={{ width: `${percentage}%` }}
        />
      </div>
    </div>
  );
}
