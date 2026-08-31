/**
 * @libraryId react-component-library:Page
 * @displayName Page
 * @version 1.0.6
 * @tags ["navigation","layout","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:Page */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { ReactNode } from "react";

const pageStyles = `
[data-rcl-page] { display: grid; min-block-size: 100dvh; min-inline-size: 0; grid-template-columns: minmax(15rem, 20rem) minmax(0, 1fr); background: var(--color-background); color: var(--color-foreground); }
[data-rcl-page-navigation] { min-block-size: 0; min-inline-size: 0; overflow: auto; border-inline-end: var(--border-hairline) solid var(--color-border); background: var(--color-surface); padding: var(--space-sm); }
[data-rcl-page-content] { min-block-size: 0; min-inline-size: 0; padding: clamp(var(--space-sm), 3vw, var(--space-xl)); }
[data-rcl-page] [data-rcl-page-navigation] > nav { min-block-size: 100%; }
@media (max-width: 48rem) { [data-rcl-page] { grid-template-columns: 1fr; } [data-rcl-page-navigation] { max-block-size: none; border-block-end: var(--border-hairline) solid var(--color-border); border-inline-end: 0; padding: var(--space-xs); } [data-rcl-page-content] { padding: var(--space-sm); } }
`;

export const Page = withClassName(function Page({
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
  const strings = useStrings();
  return (
    <>
      <StyleSheet name="page-1-0-5-1" css={pageStyles} />
      <div data-testid="navigation.page" data-page-state={state} data-rcl-page>
        {navigation ? (
          <aside
            data-rcl-page-navigation
            aria-label={strings("navigation.page.page-navigation", "Page navigation")}
          >
            {navigation}
          </aside>
        ) : null}
        <main tabIndex={-1} data-rcl-page-content>
          {content ?? children}
        </main>
      </div>
    </>
  );
});
