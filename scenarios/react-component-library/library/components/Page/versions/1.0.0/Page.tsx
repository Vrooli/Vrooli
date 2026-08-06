/** @vrooliComponentSource react-component-library:Page */
import type { ReactNode } from "react";
export function Page({
  navigation,
  content,
  children,
  state = "ready",
}: {
  navigation?: ReactNode;
  content?: ReactNode;
  children?: ReactNode;
  state?: string;
}) {
  return (
    <>
      <style
        data-rcl-page-styles
        dangerouslySetInnerHTML={{
          __html: `
        [data-rcl-page] { min-height: 100vh; display: grid; grid-template-columns: minmax(14rem, 18rem) minmax(0, 1fr); background: var(--color-background); color: var(--color-foreground); }
        [data-rcl-page-navigation] { min-width: 0; padding: var(--space-md); border-inline-end: var(--border-hairline) solid var(--color-border); background: var(--color-surface); }
        [data-rcl-page-content] { min-width: 0; padding: clamp(var(--space-md), 4vw, var(--space-2xl)); }
        @media (max-width: 720px) { [data-rcl-page] { grid-template-columns: 1fr; } [data-rcl-page-navigation] { border-inline-end: 0; border-block-end: var(--border-hairline) solid var(--color-border); padding: var(--space-sm); } [data-rcl-page-content] { padding: var(--space-sm); } }
      `,
        }}
      />
      <div data-page-state={state} data-rcl-page>
        {navigation ? (
          <aside data-rcl-page-navigation>{navigation}</aside>
        ) : null}
        <main tabIndex={-1} data-rcl-page-content>
          {content ?? children}
        </main>
      </div>
    </>
  );
}
