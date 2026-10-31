import { useEffect, useRef } from "react";
import { ShieldAlert } from "lucide-react";
import { describeDenial } from "../../../lib/consoleActions";
import type { AuthzMatrix } from "../../../types/console";
import { NextActionControl } from "./NextActionControl";

export interface DeniedStateProps {
  error: unknown;
  matrix?: AuthzMatrix | null;
  /** The refused route, used to name the scope when the refusal carries none. */
  method?: string;
  path?: string;
  target?: string;
  /** Move focus to the refusal heading when it appears. */
  autoFocus?: boolean;
  testId?: string;
}

/**
 * DeniedState explains an authority refusal in terms of the specific
 * permission that is missing and offers the API's remediation. It renders
 * nothing for errors that are not authority refusals.
 */
export function DeniedState({ error, matrix, method, path, target, autoFocus = true, testId = "console-denied" }: DeniedStateProps) {
  const headingRef = useRef<HTMLHeadingElement>(null);
  const denial = describeDenial(error, { matrix, method, path, target });
  useEffect(() => {
    if (denial && autoFocus) headingRef.current?.focus();
  }, [denial?.code, autoFocus]); // eslint-disable-line react-hooks/exhaustive-deps
  if (!denial) return null;
  return (
    <div data-testid={testId} data-state="denied" data-code={denial.code} role="region" aria-labelledby={`${testId}-title`} className="rounded-lg border border-red-500/40 bg-red-500/10 p-4">
      <div className="flex items-start gap-3">
        <ShieldAlert className="h-5 w-5 shrink-0 text-red-300" aria-hidden="true" />
        <div className="min-w-0 space-y-2">
          <h3 id={`${testId}-title`} ref={headingRef} tabIndex={-1} className="text-sm font-semibold text-red-100 outline-none focus-visible:ring-2 focus-visible:ring-blue-400 rounded">
            {denial.title}
          </h3>
          <p className="text-sm text-red-100/90">{denial.explanation}</p>
          <dl className="text-xs text-red-100/80 space-y-1">
            {denial.missingScope && (
              <div className="flex gap-2">
                <dt>Missing scope</dt>
                <dd className="font-mono" data-testid={`${testId}-scope`}>
                  {denial.missingScope}
                </dd>
              </div>
            )}
            {denial.target && (
              <div className="flex gap-2">
                <dt>Target</dt>
                <dd className="font-mono" data-testid={`${testId}-target`}>
                  {denial.target}
                </dd>
              </div>
            )}
            <div className="flex gap-2">
              <dt>Code</dt>
              <dd className="font-mono">{denial.code}</dd>
            </div>
          </dl>
          <NextActionControl next={denial.nextAction} testId={`${testId}-next-action`} />
        </div>
      </div>
    </div>
  );
}
