import type { ReactNode } from "react";

import { useCanWrite } from "./useWriteAccess";

/**
 * Disables every control in a write form until Money Ledger has said this
 * browser may write. The page-level access notice says why and how to sign in;
 * the API refuses the write regardless, so this is presentation, not the gate.
 */
export function WriteFieldset({ children, className }: { children: ReactNode; className?: string }) {
  const canWrite = useCanWrite();
  return (
    <fieldset disabled={!canWrite} aria-describedby={canWrite ? undefined : ACCESS_NOTICE_ID} className={["m-0 min-w-0 border-0 p-0", className].filter(Boolean).join(" ")}>
      {children}
    </fieldset>
  );
}

/** The element id of the page's access notice, which write controls cite. */
export const ACCESS_NOTICE_ID = "money-ledger-access-notice";
