/** @vrooliComponentSource react-component-library:ButtonGroup */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import type { ReactNode } from "react";
import type { HTMLAttributes } from "react";

const styles = `
[data-rcl-button-group] { display: inline-flex; align-items: center; flex-wrap: wrap; gap: var(--space-xs); max-inline-size: 100%; }
@media (max-width: 36rem) { [data-rcl-button-group] { inline-size: 100%; } [data-rcl-button-group] > * { flex: 1 1 auto; } }
`;

export function ButtonGroup({
  children,
  label = "Actions",
  ...props
}: HTMLAttributes<HTMLDivElement> & { label?: string; children?: ReactNode }) {
  return (
    <>
      <StyleSheet name="buttongroup-1-0-0-1" css={styles} />
      <div role="group" aria-label={label} data-rcl-button-group {...props}>
        {children}
      </div>
    </>
  );
}
