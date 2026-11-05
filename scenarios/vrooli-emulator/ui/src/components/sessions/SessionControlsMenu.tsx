import { useState, useCallback } from "react";
import {
  Play,
  Square,
  Camera,
  Circle,
  WifiOff,
  Snail,
  Move,
  Settings,
  ClipboardCopy,
  ClipboardPaste,
  Moon,
  Globe,
  Loader2,
  AlertCircle,
  X,
} from "lucide-react";
import { Popover, PopoverTrigger, PopoverContent } from "../ui/popover";
import { useSessionStore } from "../../store/sessionStore";
import { executeSessionControl, type ControlResult } from "../../lib/api/sessions";

interface MenuItemProps {
  icon: React.ReactNode;
  label: string;
  onClick: () => void;
  disabled?: boolean;
  dimmed?: boolean;
  toggle?: boolean;
  toggleActive?: boolean;
  loading?: boolean;
}

function MenuItem({ icon, label, onClick, disabled, dimmed, toggle, toggleActive, loading }: MenuItemProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled || loading}
      className={`flex items-center gap-2 w-full px-3 py-1.5 text-xs text-left rounded transition
        ${dimmed ? "text-slate-500 cursor-not-allowed" : "text-slate-300 hover:bg-slate-800 hover:text-slate-100"}
        ${disabled ? "opacity-50 cursor-not-allowed" : ""}
      `}
    >
      <span className="w-4 h-4 shrink-0 flex items-center justify-center">
        {loading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : icon}
      </span>
      <span className="flex-1">{label}</span>
      {toggle && (
        <span
          className={`w-3 h-3 rounded-full border ${
            toggleActive
              ? "bg-emerald-400 border-emerald-500"
              : "bg-transparent border-slate-600"
          }`}
        />
      )}
    </button>
  );
}

interface InlineFormProps {
  children: React.ReactNode;
  onClose: () => void;
}

function InlineForm({ children, onClose }: InlineFormProps) {
  return (
    <div className="px-3 py-2 border-t border-slate-800 bg-slate-950/50">
      <div className="flex items-center justify-between mb-2">
        <span className="text-[10px] text-slate-500 uppercase tracking-wider">Settings</span>
        <button type="button" onClick={onClose} className="text-slate-500 hover:text-slate-300">
          <X className="h-3 w-3" />
        </button>
      </div>
      {children}
    </div>
  );
}

export function SessionControlsMenu() {
  const activeSession = useSessionStore((s) => s.activeSession);
  const connectionStatus = useSessionStore((s) => s.connectionStatus);
  const [activeAction, setActiveAction] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [subForm, setSubForm] = useState<string | null>(null);

  const [resizeWidth, setResizeWidth] = useState(1280);
  const [resizeHeight, setResizeHeight] = useState(720);
  const [envInput, setEnvInput] = useState("");
  const [clipboardContent, setClipboardContent] = useState("");
  const [localeInput, setLocaleInput] = useState("");
  const [bandwidthInput, setBandwidthInput] = useState(256);

  const connected = connectionStatus === "connected";
  const sessionId = activeSession?.id;

  const execute = useCallback(
    async (action: string, params?: Record<string, unknown>) => {
      if (!sessionId) return;
      setActiveAction(action);
      setError(null);
      try {
        const result: ControlResult = await executeSessionControl(sessionId, { action, params });
        if (result.status === "error") {
          setError(result.message ?? "Action failed");
        }
        void useSessionStore.getState().refreshSession();
      } catch (err) {
        setError(err instanceof Error ? err.message : "Action failed");
      } finally {
        setActiveAction(null);
      }
    },
    [sessionId],
  );

  if (error) {
    setTimeout(() => setError(null), 5000);
  }

  const isRecording = activeSession?.is_recording ?? false;
  const networkMode = activeSession?.network_mode ?? "normal";
  const darkMode = activeSession?.dark_mode ?? false;
  const appRunning = activeSession?.app_running ?? false;

  return (
    <Popover>
      <PopoverTrigger>
        <button
          type="button"
          className="flex items-center gap-1.5 rounded px-2 py-1 text-xs font-medium text-blue-300 border border-blue-800/50 hover:bg-blue-950/30 hover:text-blue-200 transition"
        >
          <Settings className="h-3.5 w-3.5" />
          Controls
        </button>
      </PopoverTrigger>
      <PopoverContent align="end" side="bottom" className="max-h-[70vh] overflow-y-auto">
        {error && (
          <div className="flex items-center gap-2 px-3 py-1.5 bg-red-950/40 border-b border-red-800/40">
            <AlertCircle className="h-3 w-3 text-red-400 shrink-0" />
            <span className="text-[11px] text-red-300 truncate flex-1">{error}</span>
            <button type="button" onClick={() => setError(null)} className="text-red-400 hover:text-red-200 p-0.5">
              <X className="h-3 w-3" />
            </button>
          </div>
        )}

        <div className="py-1.5">
          <div className="px-3 py-1 text-[10px] text-slate-500 uppercase tracking-wider font-medium">App</div>
          <MenuItem
            icon={<Play className="h-3.5 w-3.5" />}
            label="Launch App"
            onClick={() => void execute("launch_app")}
            disabled={!connected}
            loading={activeAction === "launch_app"}
          />
          <MenuItem
            icon={<Square className="h-3.5 w-3.5" />}
            label="Quit App"
            onClick={() => void execute("quit_app")}
            disabled={!connected}
            dimmed={!appRunning}
            loading={activeAction === "quit_app"}
          />
          <MenuItem
            icon={<Camera className="h-3.5 w-3.5" />}
            label="Screenshot"
            onClick={() => void execute("screenshot")}
            disabled={!connected}
            loading={activeAction === "screenshot"}
          />
          <MenuItem
            icon={<Circle className={`h-3.5 w-3.5 ${isRecording ? "text-red-400 fill-red-400" : ""}`} />}
            label={isRecording ? "Stop Recording" : "Record"}
            onClick={() => void execute(isRecording ? "stop_recording" : "start_recording")}
            disabled={!connected}
            loading={activeAction === "start_recording" || activeAction === "stop_recording"}
          />
        </div>

        <div className="border-t border-slate-800 py-1.5">
          <div className="px-3 py-1 text-[10px] text-slate-500 uppercase tracking-wider font-medium">Environment</div>
          <MenuItem
            icon={<WifiOff className="h-3.5 w-3.5" />}
            label="Offline Mode"
            onClick={() => void execute("offline_mode", { enabled: networkMode !== "offline" })}
            disabled={!connected}
            toggle
            toggleActive={networkMode === "offline"}
            loading={activeAction === "offline_mode"}
          />
          <MenuItem
            icon={<Snail className="h-3.5 w-3.5" />}
            label="Slow Connection"
            onClick={() => {
              if (networkMode === "slow") {
                void execute("slow_connection", { enabled: false });
              } else {
                setSubForm("slow");
              }
            }}
            disabled={!connected}
            toggle
            toggleActive={networkMode === "slow"}
            loading={activeAction === "slow_connection"}
          />
          {subForm === "slow" && (
            <InlineForm onClose={() => setSubForm(null)}>
              <div className="flex items-center gap-2">
                <input
                  type="number"
                  min={1}
                  value={bandwidthInput}
                  onChange={(e) => setBandwidthInput(Number(e.target.value))}
                  className="w-20 rounded border border-white/20 bg-white/5 px-2 py-1 text-xs text-slate-200"
                  placeholder="kbps"
                />
                <span className="text-[10px] text-slate-500">kbps</span>
                <button
                  type="button"
                  onClick={() => {
                    void execute("slow_connection", { enabled: true, bandwidth_kbps: bandwidthInput });
                    setSubForm(null);
                  }}
                  className="rounded bg-blue-600 px-2 py-1 text-xs text-white hover:bg-blue-500"
                >
                  Apply
                </button>
              </div>
            </InlineForm>
          )}
          <MenuItem
            icon={<Move className="h-3.5 w-3.5" />}
            label="Resize Display..."
            onClick={() => setSubForm(subForm === "resize" ? null : "resize")}
            disabled={!connected}
          />
          {subForm === "resize" && (
            <InlineForm onClose={() => setSubForm(null)}>
              <div className="flex items-center gap-2">
                <input
                  type="number"
                  min={640}
                  max={3840}
                  value={resizeWidth}
                  onChange={(e) => setResizeWidth(Number(e.target.value))}
                  className="w-16 rounded border border-white/20 bg-white/5 px-2 py-1 text-xs text-slate-200"
                />
                <span className="text-slate-500">&times;</span>
                <input
                  type="number"
                  min={480}
                  max={2160}
                  value={resizeHeight}
                  onChange={(e) => setResizeHeight(Number(e.target.value))}
                  className="w-16 rounded border border-white/20 bg-white/5 px-2 py-1 text-xs text-slate-200"
                />
                <button
                  type="button"
                  onClick={() => {
                    void execute("resize_display", { width: resizeWidth, height: resizeHeight });
                    setSubForm(null);
                  }}
                  className="rounded bg-blue-600 px-2 py-1 text-xs text-white hover:bg-blue-500"
                >
                  Apply
                </button>
              </div>
            </InlineForm>
          )}
          <MenuItem
            icon={<Settings className="h-3.5 w-3.5" />}
            label="Env Variables..."
            onClick={() => setSubForm(subForm === "env" ? null : "env")}
            disabled={!connected}
          />
          {subForm === "env" && (
            <InlineForm onClose={() => setSubForm(null)}>
              <textarea
                value={envInput}
                onChange={(e) => setEnvInput(e.target.value)}
                placeholder={"KEY=VALUE\nANOTHER=VALUE"}
                rows={3}
                className="w-full rounded border border-white/20 bg-white/5 px-2 py-1 text-xs text-slate-200 mb-2 font-mono"
              />
              <button
                type="button"
                onClick={() => {
                  const vars: Record<string, string> = {};
                  for (const line of envInput.split("\n")) {
                    const idx = line.indexOf("=");
                    if (idx > 0) {
                      vars[line.slice(0, idx).trim()] = line.slice(idx + 1).trim();
                    }
                  }
                  void execute("inject_env", { vars });
                  setSubForm(null);
                }}
                className="rounded bg-blue-600 px-2 py-1 text-xs text-white hover:bg-blue-500"
              >
                Inject
              </button>
            </InlineForm>
          )}
        </div>

        <div className="border-t border-slate-800 py-1.5">
          <div className="px-3 py-1 text-[10px] text-slate-500 uppercase tracking-wider font-medium">Advanced</div>
          <MenuItem
            icon={<ClipboardCopy className="h-3.5 w-3.5" />}
            label="Read Clipboard"
            onClick={() => void execute("clipboard_read")}
            disabled={!connected}
            loading={activeAction === "clipboard_read"}
          />
          <MenuItem
            icon={<ClipboardPaste className="h-3.5 w-3.5" />}
            label="Write Clipboard..."
            onClick={() => setSubForm(subForm === "clipboard" ? null : "clipboard")}
            disabled={!connected}
          />
          {subForm === "clipboard" && (
            <InlineForm onClose={() => setSubForm(null)}>
              <textarea
                value={clipboardContent}
                onChange={(e) => setClipboardContent(e.target.value)}
                placeholder="Clipboard content..."
                rows={2}
                className="w-full rounded border border-white/20 bg-white/5 px-2 py-1 text-xs text-slate-200 mb-2"
              />
              <button
                type="button"
                onClick={() => {
                  void execute("clipboard_write", { content: clipboardContent });
                  setSubForm(null);
                }}
                className="rounded bg-blue-600 px-2 py-1 text-xs text-white hover:bg-blue-500"
              >
                Write
              </button>
            </InlineForm>
          )}
          <MenuItem
            icon={<Moon className="h-3.5 w-3.5" />}
            label="Dark Mode"
            onClick={() => void execute("dark_mode", { enabled: !darkMode })}
            disabled={!connected}
            toggle
            toggleActive={darkMode}
            loading={activeAction === "dark_mode"}
          />
          <MenuItem
            icon={<Globe className="h-3.5 w-3.5" />}
            label="Locale..."
            onClick={() => setSubForm(subForm === "locale" ? null : "locale")}
            disabled={!connected}
          />
          {subForm === "locale" && (
            <InlineForm onClose={() => setSubForm(null)}>
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  value={localeInput}
                  onChange={(e) => setLocaleInput(e.target.value)}
                  placeholder="e.g., fr_FR.UTF-8"
                  className="flex-1 rounded border border-white/20 bg-white/5 px-2 py-1 text-xs text-slate-200"
                />
                <button
                  type="button"
                  onClick={() => {
                    void execute("locale", { locale: localeInput });
                    setSubForm(null);
                  }}
                  className="rounded bg-blue-600 px-2 py-1 text-xs text-white hover:bg-blue-500"
                >
                  Apply
                </button>
              </div>
            </InlineForm>
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
}
