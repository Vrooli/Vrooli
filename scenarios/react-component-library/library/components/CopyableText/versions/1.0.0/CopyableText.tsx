/** @vrooliComponentSource react-component-library:CopyableText */
export function CopyableText({ value }: { value: string }) {
  return (
    <button
      type="button"
      aria-label="Copy text"
      style={{
        minHeight: 44,
        border: "1px solid var(--color-border, #cbd5e1)",
        borderRadius: 8,
        background: "var(--color-surface-muted, #f1f5f9)",
        paddingInline: 16,
        font: "inherit",
      }}
    >
      {value}
    </button>
  );
}
