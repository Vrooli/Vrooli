/**
 * @libraryId react-component-library:InspectorLayout
 * @version 1.0.0
 * @status released
 * @deps {"react":"^18"}
 */
import type { ReactNode } from "react";
import { AsyncPanel } from "../../../AsyncPanel/versions/1.0.0/AsyncPanel";
import type { ExperienceSurfaceState } from "../../../ExperienceSurface/versions/1.0.0/ExperienceSurface";

export interface InspectorLayoutProps {
  title: string;
  canvas: ReactNode;
  inspector: ReactNode;
  inspectorState?: ExperienceSurfaceState;
  inspectorSurfaceId?: string;
  toolbar?: ReactNode;
  loading?: ReactNode;
  empty?: ReactNode;
  partial?: ReactNode;
  error?: ReactNode;
  onRetry?: () => void;
  className?: string;
}

// InspectorLayout deliberately models only the independent inspector region;
// the canvas remains a caller-owned primary task and never inherits a sidebar
// failure as its own lifecycle state.
export function InspectorLayout({ title, canvas, inspector, inspectorState = "ready", inspectorSurfaceId = "inspector", toolbar, loading, empty, partial, error, onRetry, className }: InspectorLayoutProps) {
  return <main className={["grid min-h-full gap-4 p-4 xl:grid-cols-[minmax(0,1fr)_22rem]", className].filter(Boolean).join(" ")}>
    <section aria-label={`${title} canvas`} className="min-w-0 rounded-control border border-app-border bg-app-surface">
      {toolbar ? <header className="flex flex-wrap items-center justify-between gap-2 border-b border-app-border p-3"><h1 className="font-semibold text-app-foreground">{title}</h1>{toolbar}</header> : <h1 className="sr-only">{title}</h1>}
      <div className="min-h-64 p-4">{canvas}</div>
    </section>
    <AsyncPanel surfaceId={inspectorSurfaceId} state={inspectorState} loading={loading} empty={empty} partial={partial} error={error} onRetry={onRetry} className="rounded-control border border-app-border bg-app-surface p-4">
      {inspector}
    </AsyncPanel>
  </main>;
}
