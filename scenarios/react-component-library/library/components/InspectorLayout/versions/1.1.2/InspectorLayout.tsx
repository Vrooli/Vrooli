/**
 * @libraryId react-component-library:InspectorLayout
 * @displayName Inspector Layout
 * @description A responsive workbench composition that keeps a primary canvas usable while an inspector region reports its lifecycle.
 * @version 1.1.2
 * @tags ["layout","inspector","workbench","experience"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import type { ReactNode } from "react";
import { AsyncPanel } from "@vrooli/react-component-library/AsyncPanel/1.0.2";
import type { ExperienceSurfaceState } from "@vrooli/react-component-library/ExperienceSurface/1.0.2";
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

export const InspectorLayout = withClassName(function InspectorLayout({
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
      <StyleSheet name="inspector-layout-1-1-1" css={inspectorLayoutStyles} />
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
});
