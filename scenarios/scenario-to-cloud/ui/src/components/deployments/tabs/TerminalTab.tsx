import { useEffect, useRef, useState, useCallback } from "react";
import {
  Terminal,
  RefreshCw,
  Loader2,
  AlertCircle,
  PlugZap,
  Plug,
  Maximize2,
  Minimize2,
} from "lucide-react";
import { cn } from "../../../lib/utils";
import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

interface TerminalTabProps {
  deploymentId: string;
}

type ConnectionState = "disconnected" | "connecting" | "connected" | "error";

export function TerminalTab({ deploymentId }: TerminalTabProps) {
  const [connectionState, setConnectionState] = useState<ConnectionState>("disconnected");
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [isFullscreen, setIsFullscreen] = useState(false);

  const wsRef = useRef<WebSocket | null>(null);
  const terminalRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const [output, setOutput] = useState<string[]>([]);
  const [inputBuffer, setInputBuffer] = useState("");

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    setConnectionState("connecting");
    setErrorMessage(null);

    // Build WebSocket URL
    const apiBase = resolveApiBase({ appendSuffix: true });
    const wsUrl = apiBase
      .replace(/^http/, "ws")
      .replace(/\/$/, "") + `/deployments/${deploymentId}/terminal`;

    try {
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        setConnectionState("connected");
        setOutput((prev) => [...prev, "\x1b[32m>>> Connected to VPS terminal\x1b[0m\n"]);
      };

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          if (msg.type === "data") {
            setOutput((prev) => [...prev, msg.data]);
          } else if (msg.type === "error") {
            setOutput((prev) => [...prev, `\x1b[31mError: ${msg.data}\x1b[0m\n`]);
          }
        } catch {
          // Raw data
          setOutput((prev) => [...prev, event.data]);
        }
      };

      ws.onerror = () => {
        setConnectionState("error");
        setErrorMessage("WebSocket connection error");
      };

      ws.onclose = () => {
        setConnectionState("disconnected");
        setOutput((prev) => [...prev, "\x1b[33m>>> Disconnected from VPS\x1b[0m\n"]);
        wsRef.current = null;
      };
    } catch (err) {
      setConnectionState("error");
      setErrorMessage(err instanceof Error ? err.message : "Failed to connect");
    }
  }, [deploymentId]);

  const disconnect = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    setConnectionState("disconnected");
  }, []);

  const sendInput = useCallback((data: string) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type: "data", data }));
    }
  }, []);

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (connectionState !== "connected") return;

    // Handle special keys that shouldn't go through textarea onChange
    if (e.key === "Enter") {
      e.preventDefault();
      sendInput("\r"); // Just send carriage return, chars already sent via onChange
      setInputBuffer("");
    } else if (e.key === "Backspace") {
      e.preventDefault();
      sendInput("\x7f"); // Send backspace character
      setInputBuffer((prev) => prev.slice(0, -1)); // Update local buffer
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      sendInput("\x1b[A"); // Up arrow
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      sendInput("\x1b[B"); // Down arrow
    } else if (e.key === "ArrowRight") {
      e.preventDefault();
      sendInput("\x1b[C"); // Right arrow
    } else if (e.key === "ArrowLeft") {
      e.preventDefault();
      sendInput("\x1b[D"); // Left arrow
    } else if (e.key === "Tab") {
      e.preventDefault();
      sendInput("\t"); // Tab for autocomplete
    } else if (e.ctrlKey && e.key === "c") {
      e.preventDefault();
      sendInput("\x03"); // Ctrl+C
      setInputBuffer("");
    } else if (e.ctrlKey && e.key === "d") {
      e.preventDefault();
      sendInput("\x04"); // Ctrl+D
    } else if (e.ctrlKey && e.key === "l") {
      e.preventDefault();
      setOutput([]);
      sendInput("\x0c"); // Ctrl+L (clear screen)
    } else if (e.ctrlKey && e.key === "a") {
      e.preventDefault();
      sendInput("\x01"); // Ctrl+A (beginning of line)
    } else if (e.ctrlKey && e.key === "e") {
      e.preventDefault();
      sendInput("\x05"); // Ctrl+E (end of line)
    } else if (e.ctrlKey && e.key === "u") {
      e.preventDefault();
      sendInput("\x15"); // Ctrl+U (clear line)
      setInputBuffer("");
    } else if (e.ctrlKey && e.key === "k") {
      e.preventDefault();
      sendInput("\x0b"); // Ctrl+K (kill to end of line)
    } else if (e.ctrlKey && e.key === "w") {
      e.preventDefault();
      sendInput("\x17"); // Ctrl+W (delete word)
    }
  }, [connectionState, sendInput]);

  const handleInputChange = useCallback((e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const value = e.target.value;
    const diff = value.slice(inputBuffer.length);
    if (diff) {
      sendInput(diff);
    }
    setInputBuffer(value);
  }, [inputBuffer, sendInput]);

  // Auto-scroll to bottom
  useEffect(() => {
    if (terminalRef.current) {
      terminalRef.current.scrollTop = terminalRef.current.scrollHeight;
    }
  }, [output]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      disconnect();
    };
  }, [disconnect]);

  // Process ANSI escape codes for display
  const processAnsi = (text: string): string => {
    // Remove ANSI codes - a full terminal would use xterm.js for proper handling
    return text
      // Remove OSC sequences (window title, etc) - \x1b]...(\x07|\x1b\\)
      .replace(/\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)/g, "")
      // Remove CSI sequences - \x1b[...letter
      .replace(/\x1b\[[0-9;?]*[A-Za-z]/g, "")
      // Remove other escape sequences
      .replace(/\x1b[=>]/g, "")
      // Remove standalone control characters (bell, etc)
      .replace(/[\x07]/g, "");
  };

  return (
    <div
      className={cn(
        "space-y-4",
        isFullscreen && "fixed inset-0 z-50 bg-slate-950 p-4"
      )}
    >
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Terminal className="h-5 w-5 text-slate-400" />
          <span className="text-sm font-medium text-white">VPS Terminal</span>
          <div className="flex items-center gap-2">
            {connectionState === "connected" && (
              <span className="flex items-center gap-1 text-xs text-emerald-400">
                <PlugZap className="h-3 w-3" />
                Connected
              </span>
            )}
            {connectionState === "connecting" && (
              <span className="flex items-center gap-1 text-xs text-blue-400">
                <Loader2 className="h-3 w-3 animate-spin" />
                Connecting...
              </span>
            )}
            {connectionState === "disconnected" && (
              <span className="flex items-center gap-1 text-xs text-slate-400">
                <Plug className="h-3 w-3" />
                Disconnected
              </span>
            )}
            {connectionState === "error" && (
              <span className="flex items-center gap-1 text-xs text-red-400">
                <AlertCircle className="h-3 w-3" />
                Error
              </span>
            )}
          </div>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={() => setIsFullscreen(!isFullscreen)}
            className="p-1.5 rounded border border-white/10 hover:bg-white/5 transition-colors"
          >
            {isFullscreen ? (
              <Minimize2 className="h-4 w-4 text-slate-400" />
            ) : (
              <Maximize2 className="h-4 w-4 text-slate-400" />
            )}
          </button>

          {connectionState === "connected" ? (
            <button
              onClick={disconnect}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs transition-colors border border-red-500/30 bg-red-500/10 text-red-400 hover:bg-red-500/20"
            >
              <Plug className="h-3 w-3" />
              Disconnect
            </button>
          ) : (
            <button
              onClick={connect}
              disabled={connectionState === "connecting"}
              className={cn(
                "flex items-center gap-1.5 px-3 py-1.5 rounded text-xs transition-colors",
                "border border-emerald-500/30 bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20",
                connectionState === "connecting" && "opacity-50 cursor-not-allowed"
              )}
            >
              {connectionState === "connecting" ? (
                <Loader2 className="h-3 w-3 animate-spin" />
              ) : (
                <PlugZap className="h-3 w-3" />
              )}
              Connect
            </button>
          )}
        </div>
      </div>

      {/* Error message */}
      {errorMessage && (
        <div className="p-3 rounded-lg border border-red-500/30 bg-red-500/10 text-red-400 text-sm">
          <div className="flex items-center gap-2">
            <AlertCircle className="h-4 w-4" />
            {errorMessage}
          </div>
        </div>
      )}

      {/* Terminal */}
      <div
        className={cn(
          "relative rounded-lg border border-white/10 bg-slate-950",
          isFullscreen ? "flex-1 h-full" : "h-[500px]"
        )}
      >
        {connectionState === "disconnected" && output.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-slate-400">
            <Terminal className="h-16 w-16 mb-4 opacity-30" />
            <p className="text-sm">Click "Connect" to open a terminal session</p>
            <p className="text-xs text-slate-500 mt-1">
              This will open an SSH connection to the VPS
            </p>
          </div>
        ) : (
          <>
            {/* Output area */}
            <div
              ref={terminalRef}
              className="h-full overflow-y-auto pt-1 px-4 pb-4 font-mono text-sm text-slate-200"
              onClick={() => inputRef.current?.focus()}
            >
              <pre className="whitespace-pre-wrap break-all m-0">{output.map((line, i) => (
                  <span key={i}>{processAnsi(line)}</span>
                ))}{/* Cursor indicator inline with output */}{connectionState === "connected" && (
                  <span className="animate-pulse">▋</span>
                )}</pre>
            </div>

            {/* Hidden input for keyboard capture */}
            <textarea
              ref={inputRef}
              value={inputBuffer}
              onChange={handleInputChange}
              onKeyDown={handleKeyDown}
              className="absolute opacity-0 pointer-events-none"
              autoFocus={connectionState === "connected"}
            />
          </>
        )}
      </div>

      {/* Keyboard shortcuts hint */}
      {connectionState === "connected" && (
        <div className="flex items-center gap-4 text-xs text-slate-500">
          <span>
            <kbd className="px-1.5 py-0.5 rounded bg-slate-800 text-slate-400">Ctrl+C</kbd> Interrupt
          </span>
          <span>
            <kbd className="px-1.5 py-0.5 rounded bg-slate-800 text-slate-400">Ctrl+D</kbd> EOF
          </span>
          <span>
            <kbd className="px-1.5 py-0.5 rounded bg-slate-800 text-slate-400">Ctrl+L</kbd> Clear
          </span>
        </div>
      )}
    </div>
  );
}
