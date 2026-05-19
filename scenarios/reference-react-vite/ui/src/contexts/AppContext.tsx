// Illustrative client-side state for navigation contexts declared in
// ui/flow/navigation.json. Phase 7 of the navigation-integrity rollout
// uses these toggles to exercise responsive layout, role gating, and the
// feature_beta flag without involving a real backend.
import { createContext, useContext, useMemo, useState, type ReactNode } from "react";

export type AuthState = "logged_out" | "logged_in";
export type Role = "viewer" | "editor" | "admin";

export interface AppContextValue {
  auth: AuthState;
  role: Role;
  featureBeta: boolean;
  setAuth: (next: AuthState) => void;
  setRole: (next: Role) => void;
  setFeatureBeta: (next: boolean) => void;
}

const AppContext = createContext<AppContextValue | undefined>(undefined);

export interface AppContextProviderProps {
  children: ReactNode;
  initialAuth?: AuthState;
  initialRole?: Role;
  initialFeatureBeta?: boolean;
}

export function AppContextProvider({
  children,
  initialAuth = "logged_out",
  initialRole = "viewer",
  initialFeatureBeta = false,
}: AppContextProviderProps) {
  const [auth, setAuth] = useState<AuthState>(initialAuth);
  const [role, setRole] = useState<Role>(initialRole);
  const [featureBeta, setFeatureBeta] = useState(initialFeatureBeta);

  const value = useMemo<AppContextValue>(
    () => ({ auth, role, featureBeta, setAuth, setRole, setFeatureBeta }),
    [auth, role, featureBeta],
  );
  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
}

export function useAppContext(): AppContextValue {
  const v = useContext(AppContext);
  if (!v) throw new Error("useAppContext must be used inside AppContextProvider");
  return v;
}
