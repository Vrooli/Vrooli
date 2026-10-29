import { useCallback, useState, type ReactNode } from "react";
import { AlertDialog } from "@vrooli/react-component-library/AlertDialog/2";
import { presentError } from "../../lib/errorPresentation";

export interface ConfirmRequest {
  title: string;
  description: string;
  confirmLabel: string;
  busyLabel?: string;
  /** What the action applies to, e.g. the run's title. */
  subject?: string;
  action: () => Promise<void>;
}

/**
 * The confirmation for destructive actions (replaces window.confirm). The action
 * runs from the dialog; a failure stays in the dialog as a human sentence so the
 * operator can retry or back out.
 */
export function useConfirmAction(testIdPrefix = "confirm-action"): { confirm: (request: ConfirmRequest) => void; pending: ConfirmRequest | null; dialog: ReactNode } {
  const [request, setRequest] = useState<ConfirmRequest | null>(null);
  const [error, setError] = useState<string | null>(null);
  const confirm = useCallback((next: ConfirmRequest) => {
    setError(null);
    setRequest(next);
  }, []);
  const close = () => {
    setRequest(null);
    setError(null);
  };
  const dialog = (
    <AlertDialog
      open={request !== null}
      destructive
      title={request?.title ?? ""}
      description={request?.description ?? ""}
      confirmLabel={request?.confirmLabel}
      busyLabel={request?.busyLabel}
      status={error ? "error" : "default"}
      errorMessage={error ?? undefined}
      onConfirm={async () => {
        if (!request) return;
        setError(null);
        try {
          await request.action();
          close();
        } catch (err) {
          setError(presentError(err).message);
        }
      }}
      onCancel={close}
      testIdPrefix={testIdPrefix}
    >
      {request?.subject ? <p className="break-words text-sm font-medium">{request.subject}</p> : error ? <span /> : null}
    </AlertDialog>
  );
  return { confirm, pending: request, dialog };
}
