/** @vrooliComponentSource foundations.contracts */
import { createContext, useContext, type Context, type ReactNode } from "react";

export type Density = "comfortable" | "compact";
export type Direction = "ltr" | "rtl";
export type LayerName = "base" | "raised" | "popover" | "modal" | "toast";
export type SurfaceElevation = "flat" | "raised" | "floating" | "overlay";
export type SurfaceContextKey =
  | "elevation"
  | "density"
  | "tone"
  | "columns"
  | "disabled";
export type SurfaceContextValue = Partial<{
  elevation: SurfaceElevation;
  density: Density;
  tone: string;
  columns: number;
  disabled: boolean;
}>;

// Preview bundles are intentionally isolated per adopted asset. Register the
// one source-level context on the host runtime so independently bundled Card
// and CardGrid modules still share the same provider identity.
const surfaceContextRuntime = globalThis as typeof globalThis & {
  __vrooliSurfaceContext?: Context<SurfaceContextValue>;
};
export const SurfaceContext =
  surfaceContextRuntime.__vrooliSurfaceContext ??
  (surfaceContextRuntime.__vrooliSurfaceContext =
    createContext<SurfaceContextValue>({}));

export function SurfaceProvider({
  value,
  children,
}: {
  value: SurfaceContextValue;
  children: ReactNode;
}) {
  return (
    <SurfaceContext.Provider value={value}>{children}</SurfaceContext.Provider>
  );
}

export function useSurfaceContext(): SurfaceContextValue {
  return useContext(SurfaceContext);
}

export type AsyncStatus = "idle" | "pending" | "success" | "error" | "aborted";
export type ControllableValue<T> = {
  value?: T;
  defaultValue?: T;
  onChange?: (value: T) => void;
};
export type FocusReturnTarget = HTMLElement | null | (() => HTMLElement | null);
export interface DismissReason {
  source: "escape" | "outside" | "programmatic";
  originalEvent?: Event;
}
export interface StateTransition<T> {
  status: AsyncStatus;
  value?: T;
  error?: unknown;
}
