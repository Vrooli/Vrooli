/**
 * @libraryId react-component-library:ProgressLadder
 * @displayName ProgressLadder
 * @description A rung-by-rung maturity ladder showing achieved and target states.
 * @version 1.0.4
 * @tags ["data-display","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:ProgressLadder */
import { translate } from "../../../../hooks/useLocale/versions/1.0.1/useLocale";
import { withClassName } from "../../../../foundations/ClassMerge/versions/1.0.1/ClassMerge";

export interface ProgressRung {
  id: string;
  label: string;
  complete: boolean;
  current?: boolean;
}
export const ProgressLadder = withClassName(function ProgressLadder({
  rungs = [],
}: {
  rungs?: ProgressRung[];
}) {
  return (
    <ol
      aria-label={translate("data-display.progress-ladder.aria-label.1", "Progress ladder")}
      style={{
        display: "grid",
        gap: "var(--space-2xs)",
        padding: 0,
        listStyle: "none",
      }}
    >
      {rungs.map((rung, index) => (
        <li
          key={rung.id}
          aria-current={rung.current ? "step" : undefined}
          data-complete={rung.complete}
          style={{
            display: "grid",
            gridTemplateColumns: "2rem 1fr",
            gap: "var(--space-xs)",
            alignItems: "center",
          }}
        >
          <span aria-hidden="true">{rung.complete ? "✓" : index + 1}</span>
          <span>{rung.label}</span>
        </li>
      ))}
    </ol>
  );
});
