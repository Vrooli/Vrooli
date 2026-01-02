/**
 * ManualToolDialog Component
 *
 * A dialog for manually executing a tool by filling in its parameters.
 * Shows the form, executes the tool, and displays results.
 */

import { useState, useCallback } from "react";
import { CheckCircle, XCircle, Copy, Check, Clock, Zap } from "lucide-react";
import type { EffectiveTool, ManualToolExecuteResponse } from "../../lib/api";
import { executeToolManually } from "../../lib/api";
import { Dialog, DialogHeader, DialogBody } from "../ui/dialog";
import { Button } from "../ui/button";
import { SchemaForm } from "./SchemaForm";

export interface ManualToolDialogProps {
  /** Whether the dialog is open */
  open: boolean;
  /** Callback when dialog should close */
  onClose: () => void;
  /** The tool to execute */
  tool: EffectiveTool;
  /** Optional chat ID - if provided, result is added to chat history */
  chatId?: string;
  /** Callback after successful execution */
  onSuccess?: () => void;
}

type DialogState = "form" | "executing" | "result";

/**
 * Dialog for manually triggering tool execution.
 */
export function ManualToolDialog({
  open,
  onClose,
  tool,
  chatId,
  onSuccess,
}: ManualToolDialogProps) {
  const [state, setState] = useState<DialogState>("form");
  const [result, setResult] = useState<ManualToolExecuteResponse | null>(null);
  const [copied, setCopied] = useState(false);

  // Reset state when dialog opens
  const handleClose = useCallback(() => {
    setState("form");
    setResult(null);
    setCopied(false);
    onClose();
  }, [onClose]);

  // Execute the tool
  const handleSubmit = useCallback(
    async (values: Record<string, unknown>) => {
      setState("executing");

      try {
        const response = await executeToolManually({
          scenario: tool.scenario,
          tool_name: tool.tool.name,
          arguments: values,
          chat_id: chatId,
        });

        setResult(response);
        setState("result");

        // If successful and in chat context, trigger refresh
        if (response.success && chatId && onSuccess) {
          onSuccess();
        }
      } catch (error) {
        setResult({
          success: false,
          status: "failed",
          error: error instanceof Error ? error.message : "Unknown error",
          result: null,
          execution_time_ms: 0,
        });
        setState("result");
      }
    },
    [tool, chatId, onSuccess]
  );

  // Copy result to clipboard
  const handleCopy = useCallback(() => {
    if (result?.result) {
      const text =
        typeof result.result === "string"
          ? result.result
          : JSON.stringify(result.result, null, 2);
      navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  }, [result]);

  // Format execution time
  const formatTime = (ms: number): string => {
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(2)}s`;
  };

  return (
    <Dialog open={open} onClose={handleClose} className="max-w-lg">
      <DialogHeader onClose={handleClose}>
        <div className="flex items-center gap-2">
          <Zap className="h-5 w-5 text-indigo-400" />
          {tool.tool.name}
        </div>
      </DialogHeader>

      <DialogBody>
        {/* Tool description */}
        <p className="text-sm text-slate-400 mb-4">{tool.tool.description}</p>

        {/* Form state */}
        {state === "form" && (
          <SchemaForm
            parameters={tool.tool.parameters}
            onSubmit={handleSubmit}
            onCancel={handleClose}
            submitLabel="Execute Tool"
          />
        )}

        {/* Executing state */}
        {state === "executing" && (
          <div className="flex flex-col items-center justify-center py-8">
            <div className="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin mb-4" />
            <p className="text-sm text-slate-400">Executing tool...</p>
          </div>
        )}

        {/* Result state */}
        {state === "result" && result && (
          <div className="space-y-4">
            {/* Status indicator */}
            <div
              className={`flex items-center gap-2 p-3 rounded-lg ${
                result.success
                  ? "bg-green-500/10 border border-green-500/30"
                  : "bg-red-500/10 border border-red-500/30"
              }`}
            >
              {result.success ? (
                <CheckCircle className="h-5 w-5 text-green-400" />
              ) : (
                <XCircle className="h-5 w-5 text-red-400" />
              )}
              <span
                className={`font-medium ${
                  result.success ? "text-green-400" : "text-red-400"
                }`}
              >
                {result.success ? "Execution Successful" : "Execution Failed"}
              </span>
              <span className="ml-auto flex items-center gap-1 text-xs text-slate-500">
                <Clock className="h-3 w-3" />
                {formatTime(result.execution_time_ms)}
              </span>
            </div>

            {/* Error message */}
            {result.error && (
              <div className="bg-red-500/10 border border-red-500/30 rounded-lg p-3">
                <p className="text-sm text-red-400">{result.error}</p>
              </div>
            )}

            {/* Result output */}
            {result.result && (
              <div className="relative">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-xs font-medium text-slate-500 uppercase">
                    Result
                  </span>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleCopy}
                    className="text-xs"
                  >
                    {copied ? (
                      <>
                        <Check className="h-3 w-3 mr-1" />
                        Copied
                      </>
                    ) : (
                      <>
                        <Copy className="h-3 w-3 mr-1" />
                        Copy
                      </>
                    )}
                  </Button>
                </div>
                <pre className="bg-white/5 border border-white/10 rounded-lg p-3 text-xs text-slate-300 overflow-x-auto max-h-64 overflow-y-auto">
                  {typeof result.result === "string"
                    ? result.result
                    : JSON.stringify(result.result, null, 2)}
                </pre>
              </div>
            )}

            {/* Chat integration note */}
            {chatId && result.success && result.tool_call_record && (
              <p className="text-xs text-slate-500">
                Result has been added to the chat history.
              </p>
            )}

            {/* Close button */}
            <div className="flex justify-end pt-2 border-t border-white/10">
              <Button onClick={handleClose}>
                {chatId && result.success ? "Done" : "Close"}
              </Button>
            </div>
          </div>
        )}
      </DialogBody>
    </Dialog>
  );
}
