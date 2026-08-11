/**
 * @libraryId react-component-library:InspectorLayout
 * @version 1.1.0
 * @status released
 * @deps {"react":"^18"}
 */
import type { ReactNode } from "react";
import { AsyncPanel } from "../../../AsyncPanel/versions/1.0.0/AsyncPanel";
import type { ExperienceSurfaceState } from "../../../ExperienceSurface/versions/1.0.0/ExperienceSurface";
import { inspectorLayoutStyles } from "./styles";

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

export function InspectorLayout({
  title,
  canvas,
  inspector,
  inspectorState = "ready",
  inspectorSurfaceId = "inspector",
  toolbar,
  loading,
  empty,
  partial,
  error,
  onRetry,
  className,
}: InspectorLayoutProps) {
  return (
    <>
      <style
        data-rcl-inspector-layout-styles
        dangerouslySetInnerHTML={{ __html: inspectorLayoutStyles }}
      />
      <main data-rcl-inspector-layout className={className}>
        <section aria-label={`${title} canvas`} data-rcl-inspector-canvas>
          {toolbar ? (
            <header data-rcl-inspector-toolbar>
              <h1 data-rcl-inspector-title>{title}</h1>
              {toolbar}
            </header>
          ) : (
            <h1 data-rcl-inspector-title data-rcl-visually-hidden>
              {title}
            </h1>
          )}
          <div data-rcl-inspector-canvas-body>{canvas}</div>
        </section>
        <AsyncPanel
          surfaceId={inspectorSurfaceId}
          state={inspectorState}
          loading={loading}
          empty={empty}
          partial={partial}
          error={error}
          onRetry={onRetry}
          className="rcl-inspector-panel"
        >
          {inspector}
        </AsyncPanel>
      </main>
    </>
  );
}
