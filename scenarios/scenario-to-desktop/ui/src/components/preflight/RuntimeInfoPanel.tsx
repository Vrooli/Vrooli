/**
 * Panel showing runtime identity information.
 */

import { formatTimestamp } from "../../lib/preflight-utils";

interface RuntimeInfo {
  instance_id?: string;
  started_at?: string;
  dry_run?: boolean;
  manifest_hash?: string;
  app_name?: string;
  app_version?: string;
  ipc_host?: string;
  ipc_port?: number;
  runtime_version?: string;
  build_version?: string;
  bundle_root?: string;
  app_data_dir?: string;
}

interface RuntimeInfoPanelProps {
  runtimeInfo: RuntimeInfo;
}

export function RuntimeInfoPanel({ runtimeInfo }: RuntimeInfoPanelProps) {
  return (
    <div className="rounded-md border border-slate-800 bg-slate-950/40 p-3 text-xs text-slate-200 space-y-2">
      <p className="font-semibold text-slate-100">Runtime identity</p>
      <div className="space-y-1 text-[11px] text-slate-300">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <span className="text-slate-400">Instance</span>
          <span className="text-slate-200" title={runtimeInfo.instance_id || ""}>
            {runtimeInfo.instance_id?.slice(0, 12) || "Unknown"}
          </span>
        </div>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <span className="text-slate-400">Started</span>
          <span className="text-slate-200">{runtimeInfo.started_at ? formatTimestamp(runtimeInfo.started_at) : "Unknown"}</span>
        </div>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <span className="text-slate-400">Dry run</span>
          <span className="text-slate-200">{runtimeInfo.dry_run ? "yes" : "no"}</span>
        </div>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <span className="text-slate-400">Manifest hash</span>
          <span className="text-slate-200" title={runtimeInfo.manifest_hash || ""}>
            {runtimeInfo.manifest_hash?.slice(0, 12) || "Unknown"}
          </span>
        </div>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <span className="text-slate-400">App</span>
          <span className="text-slate-200">
            {(runtimeInfo.app_name || "Unknown")} {runtimeInfo.app_version ? `v${runtimeInfo.app_version}` : ""}
          </span>
        </div>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <span className="text-slate-400">IPC</span>
          <span className="text-slate-200">
            {runtimeInfo.ipc_host ? `${runtimeInfo.ipc_host}:${runtimeInfo.ipc_port ?? ""}` : "Unknown"}
          </span>
        </div>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <span className="text-slate-400">Runtime</span>
          <span className="text-slate-200">
            {runtimeInfo.runtime_version || "Unknown"}
            {runtimeInfo.build_version ? ` · build ${runtimeInfo.build_version}` : ""}
          </span>
        </div>
        {runtimeInfo.bundle_root && (
          <div className="flex flex-wrap items-center justify-between gap-2">
            <span className="text-slate-400">Bundle root</span>
            <span className="text-slate-200" title={runtimeInfo.bundle_root}>{runtimeInfo.bundle_root}</span>
          </div>
        )}
        {runtimeInfo.app_data_dir && (
          <div className="flex flex-wrap items-center justify-between gap-2">
            <span className="text-slate-400">App data</span>
            <span className="text-slate-200" title={runtimeInfo.app_data_dir}>{runtimeInfo.app_data_dir}</span>
          </div>
        )}
      </div>
    </div>
  );
}
