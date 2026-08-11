/**
 * @vrooliComponentSource react-component-library:InspectorLayout
 * @vrooliComponentVersion 1.1.0
 * @vrooliComponentAdoption 6323c8e4-8bee-4081-9279-97931a8c27a3
 * @vrooliComponentAppliedAt 2026-08-11T00:47:54Z
 * @vrooliComponentSourceSha256 2353703ab90d21b2c023c1111dbfc0d42079f1a9b6cc5c1c017a7fc62ee2469a
 * @vrooliComponentDriftHash d4bc9077fe7d83740a0637e2a763d8bec5d77353874f4345d1e208bc07c247ff
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { ReactNode } from "react";
import { AsyncPanel } from "./AsyncPanel";
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
