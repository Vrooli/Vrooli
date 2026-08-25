/** @vrooliComponentSource react-component-library:Page */
import { translate } from "../../../../hooks/useLocale/versions/1.0.0/useLocale";

import type { ReactNode } from "react";

const pageStyles = `
[data-rcl-page] { display: grid; min-block-size: 100dvh; min-inline-size: 0; grid-template-columns: minmax(15rem, 20rem) minmax(0, 1fr); background: var(--color-background); color: var(--color-foreground); }
[data-rcl-page-navigation] { min-block-size: 0; min-inline-size: 0; overflow: auto; border-inline-end: var(--border-hairline) solid var(--color-border); background: var(--color-surface); padding: var(--space-sm); }
[data-rcl-page-content] { min-block-size: 0; min-inline-size: 0; padding: clamp(var(--space-sm), 3vw, var(--space-xl)); }
[data-rcl-page] [data-rcl-page-navigation] > nav { min-block-size: 100%; }
@media (max-width: 48rem) { [data-rcl-page] { grid-template-columns: 1fr; } [data-rcl-page-navigation] { max-block-size: none; border-block-end: var(--border-hairline) solid var(--color-border); border-inline-end: 0; padding: var(--space-xs); } [data-rcl-page-content] { padding: var(--space-sm); } }
`;

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
        dangerouslySetInnerHTML={{ __html: pageStyles }}
      />
      <div data-page-state={state} data-rcl-page>
        {navigation ? (
          <aside data-rcl-page-navigation aria-label={translate("navigation.page.aria-label.1", "Page navigation")}>
            {navigation}
          </aside>
        ) : null}
        <main tabIndex={-1} data-rcl-page-content>
          {content ?? children}
        </main>
      </div>
    </>
  );
}
