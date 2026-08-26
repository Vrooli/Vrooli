/** @vrooliComponentSource react-component-library:ProgressLadder */
export interface ProgressRung {
  id: string;
  label: string;
  complete: boolean;
  current?: boolean;
}
export function ProgressLadder({ rungs = [] }: { rungs?: ProgressRung[] }) {
  return (
    <ol
      aria-label="Progress ladder"
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
}
