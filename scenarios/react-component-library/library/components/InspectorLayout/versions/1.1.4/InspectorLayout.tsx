/**
 * @libraryId react-component-library:InspectorLayout
 * @displayName Inspector Layout
 * @version 1.1.4
 * @tags ["layout","inspector","workbench","experience"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { ReactNode } from "react";
import { AsyncPanel } from "@vrooli/react-component-library/AsyncPanel/1";
import type { ExperienceSurfaceState } from "@vrooli/react-component-library/ExperienceSurface/1";
export const inspectorLayoutStyles = `
[data-rcl-inspector-layout] { display: grid; min-inline-size: 0; min-block-size: 100%; grid-template-columns: minmax(0, 1fr); gap: var(--space-md); padding: var(--space-md); color: var(--color-foreground); }
[data-rcl-inspector-canvas] { display: grid; min-inline-size: 0; min-block-size: calc(var(--space-xl) * 6); grid-template-rows: auto minmax(0, 1fr); overflow: hidden; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); box-shadow: var(--elev-flat); }
[data-rcl-inspector-toolbar] { display: flex; min-inline-size: 0; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: var(--space-xs); border-block-end: var(--border-hairline) solid var(--color-border); background: var(--color-surface-raised); padding: var(--space-sm) var(--space-md); }
[data-rcl-inspector-title] { min-inline-size: 0; margin: 0; color: var(--color-foreground); font: var(--text-title); overflow-wrap: anywhere; }
[data-rcl-visually-hidden] { position: absolute; inline-size: 1px; block-size: 1px; overflow: hidden; clip: rect(0 0 0 0); clip-path: inset(50%); white-space: nowrap; }
[data-rcl-inspector-toolbar] > button:not([data-rcl-control]), [data-rcl-inspector-toolbar] > a:not([data-rcl-control]) { min-block-size: var(--tap-target-min); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface); color: var(--color-foreground); padding-inline: var(--space-sm); font: var(--text-label); text-decoration: none; cursor: pointer; }
[data-rcl-inspector-canvas-body] { min-inline-size: 0; min-block-size: 0; padding: var(--space-md); background: radial-gradient(circle at 1px 1px, color-mix(in srgb, var(--color-border) 55%, transparent) 1px, transparent 0) 0 0 / var(--space-sm) var(--space-sm), var(--color-surface-muted); }
[data-rcl-inspector-panel], .rcl-inspector-panel { min-inline-size: 0; align-self: start; overflow: hidden; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); padding: var(--space-md); box-shadow: var(--elev-flat); }
[data-rcl-inspector-layout] :focus-visible { outline: var(--border-strong) solid var(--color-focus); outline-offset: var(--space-3xs); }
@media (min-width: 60rem) { [data-rcl-inspector-layout] { grid-template-columns: minmax(0, 1fr) minmax(0, min(var(--sidebar-width), 36%)); } }
@media (max-width: 36rem) { [data-rcl-inspector-layout] { padding: var(--space-sm); } [data-rcl-inspector-toolbar] { align-items: flex-start; } [data-rcl-inspector-toolbar] > * { max-inline-size: 100%; } }
@media (forced-colors: active) { [data-rcl-inspector-canvas], [data-rcl-inspector-panel] { border-color: CanvasText; background: Canvas; box-shadow: none; } [data-rcl-inspector-canvas-body] { background: Canvas; } }
`;
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
