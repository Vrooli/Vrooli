import { useState, useMemo, useEffect } from "react";
import {
  Zap,
  Clock,
  MessageSquare,
  Lock,
  Link,
  Loader2,
  ChevronDown,
  ChevronRight,
  Copy,
  Check,
  Play,
  Terminal,
  AlertCircle,
  Cpu,
  HardDrive,
  Users,
  FileText,
  Network,
} from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogClose,
} from "./ui/dialog";
import { Button } from "./ui/button";
import { Input, Label } from "./ui/input";
import type { Sandbox } from "../lib/api";
import { SELECTORS } from "../consts/selectors";

// --- Types ---

export type ExecutionMode = "exec" | "run" | "interactive";
export type IsolationLevel = "full" | "vrooli-aware";

export interface LaunchConfig {
  mode: ExecutionMode;
  command: string;
  args: string[];
  isolationLevel: IsolationLevel;
  memoryLimitMB?: number;
  cpuTimeSec?: number;
  timeoutSec?: number;
  maxProcesses?: number;
  maxOpenFiles?: number;
  allowNetwork: boolean;
  env: Record<string, string>;
  workingDir: string;
  name?: string;
}

interface LaunchAgentDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  sandbox: Sandbox;
  onLaunch: (config: LaunchConfig) => void;
  isLaunching: boolean;
}

// --- Execution Mode Definitions ---

const EXECUTION_MODES: Array<{
  id: ExecutionMode;
  icon: React.ElementType;
  title: string;
  description: string;
  details: string;
  available: boolean;
}> = [
  {
    id: "exec",
    icon: Zap,
    title: "Quick Command",
    description: "Run once, get output when done",
    details: "Blocks until complete. Best for builds, tests, scripts.",
    available: true,
  },
  {
    id: "run",
    icon: Clock,
    title: "Background Process",
    description: "Start & detach, check logs later",
    details: "Returns immediately with PID. Monitor via logs.",
    available: true,
  },
  {
    id: "interactive",
    icon: MessageSquare,
    title: "Interactive Session",
    description: "Real-time I/O, streaming output",
    details: "Full PTY support for REPLs and interactive tools.",
    available: true,
  },
];

const ISOLATION_LEVELS: Array<{
  id: IsolationLevel;
  icon: React.ElementType;
  title: string;
  description: string;
  access: string[];
  blocked: string[];
}> = [
  {
    id: "full",
    icon: Lock,
    title: "Full Isolation",
    description: "Agent only sees /workspace. Maximum safety.",
    access: ["/workspace (read/write)", "/workspace-readonly (read-only)", "System binaries"],
    blocked: ["Network", "Home directory", "Vrooli CLIs", "Other sandboxes"],
  },
  {
    id: "vrooli-aware",
    icon: Link,
    title: "Vrooli-Aware",
    description: "Can access Vrooli CLIs, configs, and localhost APIs.",
    access: [
      "/workspace (read/write)",
      "~/.local/bin/ (Vrooli CLIs)",
      "~/.config/vrooli/ (configs)",
      "Localhost network",
    ],
    blocked: ["External network", "Other sandboxes", "Home directory (rest)"],
  },
];

// --- Helper Components ---

function ModeCard({
  mode,
  selected,
  onSelect,
}: {
  mode: (typeof EXECUTION_MODES)[0];
  selected: boolean;
  onSelect: () => void;
}) {
  const Icon = mode.icon;

  return (
    <button
      type="button"
      onClick={onSelect}
      disabled={!mode.available}
      className={`
        relative flex flex-col items-center p-3 rounded-lg border transition-all text-left
        ${
          selected
            ? "border-blue-500 bg-blue-500/10 ring-1 ring-blue-500"
            : mode.available
            ? "border-slate-700 bg-slate-800/50 hover:border-slate-600 hover:bg-slate-800"
            : "border-slate-800 bg-slate-900/50 opacity-50 cursor-not-allowed"
        }
      `}
    >
      <Icon
        className={`h-5 w-5 mb-2 ${selected ? "text-blue-400" : "text-slate-400"}`}
      />
      <span
        className={`text-sm font-medium ${
          selected ? "text-blue-200" : "text-slate-200"
        }`}
      >
        {mode.title}
      </span>
      <span className="text-xs text-slate-500 mt-1 text-center">
        {mode.description}
      </span>
      {!mode.available && (
        <span className="absolute top-1 right-1 text-[10px] text-slate-500 bg-slate-800 px-1 rounded">
          Soon
        </span>
      )}
    </button>
  );
}

function IsolationCard({
  level,
  selected,
  onSelect,
}: {
  level: (typeof ISOLATION_LEVELS)[0];
  selected: boolean;
  onSelect: () => void;
}) {
  const Icon = level.icon;

  return (
    <button
      type="button"
      onClick={onSelect}
      className={`
        flex-1 flex flex-col p-3 rounded-lg border transition-all text-left
        ${
          selected
            ? "border-emerald-500 bg-emerald-500/10 ring-1 ring-emerald-500"
            : "border-slate-700 bg-slate-800/50 hover:border-slate-600 hover:bg-slate-800"
        }
      `}
    >
      <div className="flex items-center gap-2 mb-2">
        <Icon
          className={`h-4 w-4 ${selected ? "text-emerald-400" : "text-slate-400"}`}
        />
        <span
          className={`text-sm font-medium ${
            selected ? "text-emerald-200" : "text-slate-200"
          }`}
        >
          {level.title}
        </span>
      </div>
      <span className="text-xs text-slate-500">{level.description}</span>
    </button>
  );
}

function ResourceInput({
  icon: Icon,
  label,
  value,
  onChange,
  placeholder,
  unit,
  min = 0,
}: {
  icon: React.ElementType;
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  unit?: string;
  min?: number;
}) {
  return (
    <div className="flex items-center gap-2">
      <Icon className="h-4 w-4 text-slate-500 flex-shrink-0" />
      <span className="text-xs text-slate-400 w-20 flex-shrink-0">{label}</span>
      <div className="relative flex-1">
        <input
          type="number"
          min={min}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          className="w-full bg-slate-800 border border-slate-700 rounded px-2 py-1 text-sm text-slate-200 placeholder-slate-500 focus:outline-none focus:border-slate-600"
        />
        {unit && (
          <span className="absolute right-2 top-1/2 -translate-y-1/2 text-xs text-slate-500">
            {unit}
          </span>
        )}
      </div>
    </div>
  );
}

function CopyableCommand({
  label,
  command,
}: {
  label: string;
  command: string;
}) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(command);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between">
        <span className="text-xs text-slate-500">{label}</span>
        <button
          type="button"
          onClick={handleCopy}
          className="flex items-center gap-1 text-xs text-slate-400 hover:text-slate-200 transition-colors"
        >
          {copied ? (
            <>
              <Check className="h-3 w-3 text-emerald-400" />
              <span className="text-emerald-400">Copied</span>
            </>
          ) : (
            <>
              <Copy className="h-3 w-3" />
              <span>Copy</span>
            </>
          )}
        </button>
      </div>
      <pre className="text-xs bg-slate-950 rounded p-2 overflow-x-auto text-slate-300 font-mono whitespace-pre-wrap break-all">
        {command}
      </pre>
    </div>
  );
}

// --- Main Component ---

export function LaunchAgentDialog({
  open,
  onOpenChange,
  sandbox,
  onLaunch,
  isLaunching,
}: LaunchAgentDialogProps) {
  // Form state
  const [mode, setMode] = useState<ExecutionMode>("exec");
  const [isolationLevel, setIsolationLevel] = useState<IsolationLevel>("full");
  const [commandInput, setCommandInput] = useState("");
  const [name, setName] = useState("");
  const [workingDir, setWorkingDir] = useState("/workspace");

  // Resource limits
  const [memoryMB, setMemoryMB] = useState("");
  const [cpuTimeSec, setCpuTimeSec] = useState("");
  const [timeoutSec, setTimeoutSec] = useState("");
  const [maxProcs, setMaxProcs] = useState("");
  const [maxFiles, setMaxFiles] = useState("");
  const [allowNetwork, setAllowNetwork] = useState(false);

  // UI state
  const [showResources, setShowResources] = useState(false);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [showCommands, setShowCommands] = useState(false);

  // Auto-expand commands section for interactive mode
  useEffect(() => {
    if (mode === "interactive") {
      setShowCommands(true);
    }
  }, [mode]);

  // Parse command into command + args
  const { command, args } = useMemo(() => {
    const parts = commandInput.trim().split(/\s+/);
    return {
      command: parts[0] || "",
      args: parts.slice(1),
    };
  }, [commandInput]);

  // Build the LaunchConfig
  const config: LaunchConfig = useMemo(
    () => ({
      mode,
      command,
      args,
      isolationLevel,
      memoryLimitMB: memoryMB ? parseInt(memoryMB, 10) : undefined,
      cpuTimeSec: cpuTimeSec ? parseInt(cpuTimeSec, 10) : undefined,
      timeoutSec: timeoutSec ? parseInt(timeoutSec, 10) : undefined,
      maxProcesses: maxProcs ? parseInt(maxProcs, 10) : undefined,
      maxOpenFiles: maxFiles ? parseInt(maxFiles, 10) : undefined,
      allowNetwork: isolationLevel === "vrooli-aware" ? true : allowNetwork,
      env: {},
      workingDir,
      name: name || undefined,
    }),
    [
      mode,
      command,
      args,
      isolationLevel,
      memoryMB,
      cpuTimeSec,
      timeoutSec,
      maxProcs,
      maxFiles,
      allowNetwork,
      workingDir,
      name,
    ]
  );

  // Generate CLI command
  const cliCommand = useMemo(() => {
    // For interactive mode, use shell (if no command) or attach (with command)
    let cliMode: string;
    if (mode === "interactive") {
      cliMode = command ? "attach" : "shell";
    } else if (mode === "run") {
      cliMode = "run";
    } else {
      cliMode = "exec";
    }

    const parts = ["workspace-sandbox", cliMode, sandbox.id];

    if (name && mode === "run") parts.push(`--name=${name}`);
    if (memoryMB) parts.push(`--memory=${memoryMB}`);
    if (cpuTimeSec) parts.push(`--cpu-time=${cpuTimeSec}`);
    if (timeoutSec && mode === "exec") parts.push(`--timeout=${timeoutSec}`);
    if (maxProcs) parts.push(`--max-procs=${maxProcs}`);
    if (maxFiles) parts.push(`--max-files=${maxFiles}`);
    if (isolationLevel === "vrooli-aware") parts.push("--vrooli-aware");
    if (allowNetwork && isolationLevel !== "vrooli-aware") parts.push("--network");
    if (workingDir !== "/workspace") parts.push(`--workdir=${workingDir}`);

    // For shell command without explicit command, don't add --
    if (mode === "interactive" && !command) {
      // Shell opens default shell, no command needed
    } else {
      parts.push("--");
      parts.push(commandInput.trim());
    }

    return parts.join(" \\\n  ");
  }, [
    mode,
    sandbox.id,
    name,
    memoryMB,
    cpuTimeSec,
    timeoutSec,
    maxProcs,
    maxFiles,
    isolationLevel,
    allowNetwork,
    workingDir,
    commandInput,
    command,
  ]);

  // Generate curl command (or WebSocket URL for interactive)
  const curlCommand = useMemo(() => {
    // For interactive mode, show WebSocket connection info
    if (mode === "interactive") {
      const wsBody: Record<string, unknown> = {
        command: command || "/bin/sh",
        args: args.length > 0 ? args : undefined,
        isolationLevel,
        cols: 80,
        rows: 24,
      };

      if (memoryMB) wsBody.memoryLimitMB = parseInt(memoryMB, 10);
      if (cpuTimeSec) wsBody.cpuTimeSec = parseInt(cpuTimeSec, 10);
      if (maxProcs) wsBody.maxProcesses = parseInt(maxProcs, 10);
      if (maxFiles) wsBody.maxOpenFiles = parseInt(maxFiles, 10);
      if (allowNetwork || isolationLevel === "vrooli-aware") wsBody.allowNetwork = true;
      if (workingDir !== "/workspace") wsBody.workingDir = workingDir;

      // Clean up undefined values
      Object.keys(wsBody).forEach((key) => {
        if (wsBody[key] === undefined) delete wsBody[key];
      });

      return `# WebSocket URL (connect and send start message):
ws://localhost:15427/api/v1/sandboxes/${sandbox.id}/exec-interactive

# First message to send after connecting:
${JSON.stringify(wsBody, null, 2)}`;
    }

    const endpoint =
      mode === "run"
        ? `/api/v1/sandboxes/${sandbox.id}/processes`
        : `/api/v1/sandboxes/${sandbox.id}/exec`;

    const body: Record<string, unknown> = {
      command,
      args: args.length > 0 ? args : undefined,
      isolationLevel,
    };

    if (name && mode === "run") body.name = name;
    if (memoryMB) body.memoryLimitMB = parseInt(memoryMB, 10);
    if (cpuTimeSec) body.cpuTimeSec = parseInt(cpuTimeSec, 10);
    if (timeoutSec && mode === "exec") body.timeoutSec = parseInt(timeoutSec, 10);
    if (maxProcs) body.maxProcesses = parseInt(maxProcs, 10);
    if (maxFiles) body.maxOpenFiles = parseInt(maxFiles, 10);
    if (allowNetwork || isolationLevel === "vrooli-aware") body.allowNetwork = true;
    if (workingDir !== "/workspace") body.workingDir = workingDir;

    // Clean up undefined values
    Object.keys(body).forEach((key) => {
      if (body[key] === undefined) delete body[key];
    });

    return `curl -X POST localhost:15427${endpoint} \\
  -H "Content-Type: application/json" \\
  -d '${JSON.stringify(body, null, 2)}'`;
  }, [
    mode,
    sandbox.id,
    command,
    args,
    name,
    memoryMB,
    cpuTimeSec,
    timeoutSec,
    maxProcs,
    maxFiles,
    isolationLevel,
    allowNetwork,
    workingDir,
  ]);

  // Get selected isolation level details
  const selectedIsolation = ISOLATION_LEVELS.find((l) => l.id === isolationLevel)!;

  // For exec/run: need command and not launching
  // For interactive: can't launch from web UI, must use CLI
  const canSubmit = command.trim() && !isLaunching && mode !== "interactive";
  const isInteractiveMode = mode === "interactive";

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;
    onLaunch(config);
  };

  const handleClose = () => {
    if (!isLaunching) {
      onOpenChange(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent
        className="max-w-3xl max-h-[90vh] overflow-y-auto"
        data-testid={SELECTORS.launchDialog}
      >
        <DialogClose onClose={handleClose} />

        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Terminal className="h-5 w-5 text-slate-400" />
            Launch Agent in Sandbox
          </DialogTitle>
          <DialogDescription>
            Configure how to run commands in this isolated workspace
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="mt-4 space-y-5">
          {/* Execution Mode Selection */}
          <div className="space-y-2">
            <Label>Execution Mode</Label>
            <div className="grid grid-cols-3 gap-2">
              {EXECUTION_MODES.map((m) => (
                <ModeCard
                  key={m.id}
                  mode={m}
                  selected={mode === m.id}
                  onSelect={() => m.available && setMode(m.id)}
                />
              ))}
            </div>
          </div>

          {/* Isolation Level Selection */}
          <div className="space-y-2">
            <Label>Isolation Level</Label>
            <div className="grid grid-cols-2 gap-2">
              {ISOLATION_LEVELS.map((l) => (
                <IsolationCard
                  key={l.id}
                  level={l}
                  selected={isolationLevel === l.id}
                  onSelect={() => setIsolationLevel(l.id)}
                />
              ))}
            </div>
          </div>

          {/* Command Input */}
          <div className="space-y-2">
            <Label htmlFor="command">
              Command{" "}
              {mode !== "interactive" && <span className="text-red-400">*</span>}
              {mode === "interactive" && (
                <span className="text-slate-500 font-normal">(optional)</span>
              )}
            </Label>
            <Input
              id="command"
              placeholder={
                mode === "interactive"
                  ? "Leave empty for shell, or specify command..."
                  : "python agent.py --task fix-bug"
              }
              value={commandInput}
              onChange={(e) => setCommandInput(e.target.value)}
              required={mode !== "interactive"}
              data-testid={SELECTORS.launchCommandInput}
            />
            <p className="text-xs text-slate-500">
              {mode === "interactive"
                ? "Leave empty to open a shell, or specify a command for attach mode."
                : "The command to execute. Arguments will be parsed automatically."}
            </p>
          </div>

          {/* Process Name (for background) */}
          {mode === "run" && (
            <div className="space-y-2">
              <Label htmlFor="name">Process Name (optional)</Label>
              <Input
                id="name"
                placeholder="my-agent"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
              <p className="text-xs text-slate-500">
                A friendly name for easier identification in logs.
              </p>
            </div>
          )}

          {/* Resource Limits (Collapsible) */}
          <div className="space-y-2">
            <button
              type="button"
              className="flex items-center gap-1.5 text-sm text-slate-400 hover:text-slate-200 transition-colors"
              onClick={() => setShowResources(!showResources)}
            >
              {showResources ? (
                <ChevronDown className="h-4 w-4" />
              ) : (
                <ChevronRight className="h-4 w-4" />
              )}
              Resource Limits
            </button>

            {showResources && (
              <div className="space-y-3 pl-4 border-l-2 border-slate-700 py-2">
                <ResourceInput
                  icon={HardDrive}
                  label="Memory"
                  value={memoryMB}
                  onChange={setMemoryMB}
                  placeholder="512"
                  unit="MB"
                />
                <ResourceInput
                  icon={Cpu}
                  label="CPU Time"
                  value={cpuTimeSec}
                  onChange={setCpuTimeSec}
                  placeholder="300"
                  unit="sec"
                />
                {mode === "exec" && (
                  <ResourceInput
                    icon={Clock}
                    label="Timeout"
                    value={timeoutSec}
                    onChange={setTimeoutSec}
                    placeholder="3600"
                    unit="sec"
                  />
                )}
                <ResourceInput
                  icon={Users}
                  label="Max Procs"
                  value={maxProcs}
                  onChange={setMaxProcs}
                  placeholder="100"
                />
                <ResourceInput
                  icon={FileText}
                  label="Max Files"
                  value={maxFiles}
                  onChange={setMaxFiles}
                  placeholder="1024"
                />
              </div>
            )}
          </div>

          {/* Advanced Options (Collapsible) */}
          <div className="space-y-2">
            <button
              type="button"
              className="flex items-center gap-1.5 text-sm text-slate-400 hover:text-slate-200 transition-colors"
              onClick={() => setShowAdvanced(!showAdvanced)}
            >
              {showAdvanced ? (
                <ChevronDown className="h-4 w-4" />
              ) : (
                <ChevronRight className="h-4 w-4" />
              )}
              Advanced Options
            </button>

            {showAdvanced && (
              <div className="space-y-3 pl-4 border-l-2 border-slate-700 py-2">
                {/* Network checkbox (only for full isolation) */}
                {isolationLevel === "full" && (
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={allowNetwork}
                      onChange={(e) => setAllowNetwork(e.target.checked)}
                      className="rounded border-slate-600 bg-slate-800 text-blue-500 focus:ring-blue-500 focus:ring-offset-slate-900"
                    />
                    <Network className="h-4 w-4 text-slate-500" />
                    <span className="text-sm text-slate-300">Allow network access</span>
                  </label>
                )}

                {/* Working directory */}
                <div className="space-y-1">
                  <Label htmlFor="workdir" className="text-xs">
                    Working Directory
                  </Label>
                  <Input
                    id="workdir"
                    value={workingDir}
                    onChange={(e) => setWorkingDir(e.target.value)}
                    placeholder="/workspace"
                    className="text-sm"
                  />
                </div>
              </div>
            )}
          </div>

          {/* Configuration Summary */}
          <div className="rounded-lg bg-slate-800/50 border border-slate-700 p-3 space-y-2">
            <div className="flex items-center gap-2 text-sm font-medium text-slate-300">
              <AlertCircle className="h-4 w-4 text-slate-400" />
              Configuration Summary
            </div>
            <div className="text-xs text-slate-400 space-y-1">
              <p>
                Your agent will run in a{" "}
                <span className="text-slate-200">{selectedIsolation.title.toLowerCase()}</span>{" "}
                container with:
              </p>
              <ul className="space-y-0.5 mt-2">
                {selectedIsolation.access.map((item) => (
                  <li key={item} className="flex items-center gap-1.5">
                    <Check className="h-3 w-3 text-emerald-400" />
                    <span>{item}</span>
                  </li>
                ))}
                {selectedIsolation.blocked.map((item) => (
                  <li key={item} className="flex items-center gap-1.5">
                    <span className="h-3 w-3 text-red-400 font-bold text-center">×</span>
                    <span className="text-slate-500">{item}</span>
                  </li>
                ))}
              </ul>
              {(memoryMB || cpuTimeSec || timeoutSec) && (
                <p className="mt-2 pt-2 border-t border-slate-700">
                  Limits:{" "}
                  {[
                    memoryMB && `${memoryMB}MB RAM`,
                    cpuTimeSec && `${cpuTimeSec}s CPU`,
                    timeoutSec && mode === "exec" && `${timeoutSec}s timeout`,
                  ]
                    .filter(Boolean)
                    .join(", ")}
                </p>
              )}
              <p className="mt-2 pt-2 border-t border-slate-700">
                {mode === "exec"
                  ? "Output will be returned after the command completes."
                  : mode === "run"
                  ? "Process will run in background. Monitor via logs."
                  : "Opens an interactive terminal session via CLI. Use the commands below."}
              </p>
            </div>
          </div>

          {/* Commands (Collapsible) */}
          <div className="space-y-2">
            <button
              type="button"
              className="flex items-center gap-1.5 text-sm text-slate-400 hover:text-slate-200 transition-colors"
              onClick={() => setShowCommands(!showCommands)}
            >
              {showCommands ? (
                <ChevronDown className="h-4 w-4" />
              ) : (
                <ChevronRight className="h-4 w-4" />
              )}
              Commands
            </button>

            {showCommands && (command || mode === "interactive") && (
              <div className="space-y-3 pl-4 border-l-2 border-slate-700 py-2">
                <CopyableCommand label="CLI" command={cliCommand} />
                <CopyableCommand
                  label={mode === "interactive" ? "WebSocket" : "API (curl)"}
                  command={curlCommand}
                />
              </div>
            )}
          </div>

          {/* Footer */}
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={handleClose}
              disabled={isLaunching}
            >
              {isInteractiveMode ? "Close" : "Cancel"}
            </Button>
            {isInteractiveMode ? (
              <Button
                type="button"
                variant="secondary"
                onClick={() => setShowCommands(true)}
                data-testid={SELECTORS.launchSubmit}
              >
                <Terminal className="h-4 w-4 mr-2" />
                Show CLI Commands
              </Button>
            ) : (
              <Button
                type="submit"
                disabled={!canSubmit}
                data-testid={SELECTORS.launchSubmit}
              >
                {isLaunching ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Launching...
                  </>
                ) : (
                  <>
                    <Play className="h-4 w-4 mr-2" />
                    {mode === "run" ? "Start Process" : "Execute"}
                  </>
                )}
              </Button>
            )}
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
