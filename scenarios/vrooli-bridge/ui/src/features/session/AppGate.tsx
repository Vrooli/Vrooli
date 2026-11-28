import type { ReactNode } from "react";

import { SignInScreen } from "./SignInScreen";
import { useSession } from "./sessionContext";

/**
 * Owner gate. The whole bridge console is owner-gated, so when no owner token is
 * present we render the sign-in surface instead of the app shell; once the owner
 * signs in the real shell (children) renders. Placed just inside the providers
 * in `App.tsx` so the shell, routing, and every owner-gated query mount only
 * after there is a token to authorize them.
 */
export function AppGate({ children }: { children: ReactNode }) {
  const { isOwner, restoreStatus } = useSession();
  return isOwner && restoreStatus === "ready" ? <>{children}</> : <SignInScreen />;
}
