/**
 * @vrooliComponentSource react-component-library:CollapsibleRegion
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 386a33cb-0355-4a03-8854-186503c5d655
 * @vrooliComponentAppliedAt 2026-08-16T04:09:29Z
 * @vrooliComponentSourceSha256 85cae206d5f6bb93e7f9957bcbb48c097f7e52b35584c1cd5390ed35763d2140
 * @vrooliComponentDriftHash fbdb02498a8addd4d6e61db586f6b37a8a0ebce35e72fd5421ae161ef6b046ae
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { ReactNode } from "react";
export function CollapsibleRegion({
  open = true,
  children,
}: {
  open?: boolean;
  children?: ReactNode;
}) {
  return (
    <div
      data-collapsible-region
      data-open={open}
      aria-hidden={!open || undefined}
      style={{
        overflow: "hidden",
        opacity: open ? 1 : 0,
        transition: "opacity var(--dur-moderate, 280ms) ease",
      }}
    >
      {open ? children : null}
    </div>
  );
}
