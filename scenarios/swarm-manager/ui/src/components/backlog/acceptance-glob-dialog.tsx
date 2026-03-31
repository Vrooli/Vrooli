/**
 * AcceptanceGlobDialog — Modal for editing acceptance_allow and acceptance_deny glob arrays.
 *
 * Two stacked textareas (one per line per pattern) with client-side validation
 * and optional API-side path-existence checking.
 */

import { useState, useCallback, useRef, useEffect } from "react";
import { FolderOpen, X as XIcon } from "lucide-react";
import { Dialog } from "../ui/dialog";
import { Button } from "../ui/button";
import {
  validateGlobLines,
  parseGlobTextarea,
  type GlobLineError,
} from "../../lib/glob-validation";
import { defaultApiClient } from "../../lib/api-client";
import { API_ENDPOINTS } from "../../lib/api-endpoints";

// ── Types ──────────────────────────────────────────────────────────────

export interface AcceptanceGlobDialogProps {
  isOpen: boolean;
  onClose: () => void;
  initialAllow: string[];
  initialDeny: string[];
  onSave: (allow: string[], deny: string[]) => void;
  isSubmitting?: boolean;
}

interface GlobWarning {
  line: number;
  warning: string;
}

interface ValidateGlobsResult {
  pattern: string;
  matchCount: number;
  valid: boolean;
  error?: string;
  warning?: string;
}

// ── Helpers ────────────────────────────────────────────────────────────

function arrayToTextarea(arr: string[]): string {
  return arr.join("\n");
}

const PLACEHOLDER = "scenarios/my-app/ui/**\nsrc/components/*.tsx";
const HELPER_TEXT = "One glob pattern per line. Relative to project root.";
const DEBOUNCE_MS = 500;

// ── Component ──────────────────────────────────────────────────────────

export function AcceptanceGlobDialog({
  isOpen,
  onClose,
  initialAllow,
  initialDeny,
  onSave,
  isSubmitting = false,
}: AcceptanceGlobDialogProps) {
  const [allowText, setAllowText] = useState("");
  const [denyText, setDenyText] = useState("");
  const [allowErrors, setAllowErrors] = useState<GlobLineError[]>([]);
  const [denyErrors, setDenyErrors] = useState<GlobLineError[]>([]);
  const [allowWarnings, setAllowWarnings] = useState<GlobWarning[]>([]);
  const [denyWarnings, setDenyWarnings] = useState<GlobWarning[]>([]);

  const allowTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const denyTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const prevOpenRef = useRef(false);

  // Reset state only on the false→true transition (dialog opening).
  // initialAllow/initialDeny are intentionally excluded from deps to avoid
  // resetting while the user is typing (parent re-renders create new array refs).
  useEffect(() => {
    if (isOpen && !prevOpenRef.current) {
      setAllowText(arrayToTextarea(initialAllow));
      setDenyText(arrayToTextarea(initialDeny));
      setAllowErrors([]);
      setDenyErrors([]);
      setAllowWarnings([]);
      setDenyWarnings([]);
    }
    prevOpenRef.current = isOpen;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen]);

  // Client-side validation for a given field
  const validateField = useCallback(
    (text: string, setErrors: (e: GlobLineError[]) => void) => {
      const result = validateGlobLines(text);
      setErrors(result.errors);
      return result.valid;
    },
    [],
  );

  // API path-existence check (on blur only)
  const checkPathExistence = useCallback(
    async (text: string, setWarnings: (w: GlobWarning[]) => void) => {
      const patterns = parseGlobTextarea(text);
      if (patterns.length === 0) {
        setWarnings([]);
        return;
      }
      try {
        const resp = await defaultApiClient.post<{
          results: ValidateGlobsResult[];
        }>(API_ENDPOINTS.backlogValidateGlobs, { patterns });
        const warnings: GlobWarning[] = [];
        const lines = text.split("\n");
        for (const r of resp.results) {
          if (r.valid && r.matchCount === 0) {
            // Find the 1-based line number for this pattern
            const lineIdx = lines.findIndex(
              (l) => l.trim() === r.pattern,
            );
            if (lineIdx >= 0) {
              warnings.push({
                line: lineIdx + 1,
                warning: "no files match this pattern",
              });
            }
          }
        }
        setWarnings(warnings);
      } catch {
        // Silently skip API check failures — client-side validation still works
        setWarnings([]);
      }
    },
    [],
  );

  // Debounced client-side validation on typing
  const handleAllowChange = useCallback(
    (value: string) => {
      setAllowText(value);
      if (allowTimerRef.current) clearTimeout(allowTimerRef.current);
      allowTimerRef.current = setTimeout(() => {
        validateField(value, setAllowErrors);
      }, DEBOUNCE_MS);
    },
    [validateField],
  );

  const handleDenyChange = useCallback(
    (value: string) => {
      setDenyText(value);
      if (denyTimerRef.current) clearTimeout(denyTimerRef.current);
      denyTimerRef.current = setTimeout(() => {
        validateField(value, setDenyErrors);
      }, DEBOUNCE_MS);
    },
    [validateField],
  );

  // On blur: run both client-side + API validation
  const handleAllowBlur = useCallback(() => {
    validateField(allowText, setAllowErrors);
    checkPathExistence(allowText, setAllowWarnings);
  }, [allowText, validateField, checkPathExistence]);

  const handleDenyBlur = useCallback(() => {
    validateField(denyText, setDenyErrors);
    checkPathExistence(denyText, setDenyWarnings);
  }, [denyText, validateField, checkPathExistence]);

  const hasErrors = allowErrors.length > 0 || denyErrors.length > 0;

  const handleSave = useCallback(() => {
    // Final validation before save
    const allowValid = validateField(allowText, setAllowErrors);
    const denyValid = validateField(denyText, setDenyErrors);
    if (!allowValid || !denyValid) return;
    onSave(parseGlobTextarea(allowText), parseGlobTextarea(denyText));
  }, [allowText, denyText, validateField, onSave]);

  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      title="Edit Acceptance Globs"
      maxWidth="max-w-xl"
      isLoading={isSubmitting}
      testId="acceptance-glob-dialog"
    >
      <div className="space-y-5">
        {/* Allow section */}
        <div>
          <label className="flex items-center gap-1.5 text-sm font-medium text-slate-300">
            <FolderOpen className="h-4 w-4" />
            Allowed Paths
          </label>
          <p className="mt-1 text-xs text-slate-500">{HELPER_TEXT}</p>
          <textarea
            value={allowText}
            onChange={(e) => handleAllowChange(e.target.value)}
            onBlur={handleAllowBlur}
            placeholder={PLACEHOLDER}
            disabled={isSubmitting}
            rows={5}
            className="mt-2 w-full rounded-md border border-slate-700 bg-slate-800 px-3 py-2 font-mono text-sm text-slate-200 placeholder:text-slate-600 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:opacity-50"
            data-testid="allow-textarea"
          />
          {allowErrors.length > 0 && (
            <div className="mt-1.5 space-y-0.5" data-testid="allow-errors">
              {allowErrors.map((e) => (
                <p key={e.line} className="text-xs text-red-400">
                  Line {e.line}: {e.error}
                </p>
              ))}
            </div>
          )}
          {allowWarnings.length > 0 && allowErrors.length === 0 && (
            <div className="mt-1.5 space-y-0.5" data-testid="allow-warnings">
              {allowWarnings.map((w) => (
                <p key={w.line} className="text-xs text-amber-400">
                  Line {w.line}: {w.warning}
                </p>
              ))}
            </div>
          )}
        </div>

        {/* Deny section */}
        <div>
          <label className="flex items-center gap-1.5 text-sm font-medium text-slate-300">
            <XIcon className="h-4 w-4" />
            Denied Paths
          </label>
          <p className="mt-1 text-xs text-slate-500">{HELPER_TEXT}</p>
          <textarea
            value={denyText}
            onChange={(e) => handleDenyChange(e.target.value)}
            onBlur={handleDenyBlur}
            placeholder={PLACEHOLDER}
            disabled={isSubmitting}
            rows={5}
            className="mt-2 w-full rounded-md border border-slate-700 bg-slate-800 px-3 py-2 font-mono text-sm text-slate-200 placeholder:text-slate-600 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:opacity-50"
            data-testid="deny-textarea"
          />
          {denyErrors.length > 0 && (
            <div className="mt-1.5 space-y-0.5" data-testid="deny-errors">
              {denyErrors.map((e) => (
                <p key={e.line} className="text-xs text-red-400">
                  Line {e.line}: {e.error}
                </p>
              ))}
            </div>
          )}
          {denyWarnings.length > 0 && denyErrors.length === 0 && (
            <div className="mt-1.5 space-y-0.5" data-testid="deny-warnings">
              {denyWarnings.map((w) => (
                <p key={w.line} className="text-xs text-amber-400">
                  Line {w.line}: {w.warning}
                </p>
              ))}
            </div>
          )}
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-3 border-t border-slate-800 pt-4">
          <Button
            variant="ghost"
            onClick={onClose}
            disabled={isSubmitting}
            data-testid="glob-dialog-cancel"
          >
            Cancel
          </Button>
          <Button
            onClick={handleSave}
            disabled={hasErrors || isSubmitting}
            data-testid="glob-dialog-save"
          >
            {isSubmitting ? "Saving…" : "Save"}
          </Button>
        </div>
      </div>
    </Dialog>
  );
}
