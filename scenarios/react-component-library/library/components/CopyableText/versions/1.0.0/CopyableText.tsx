/** @vrooliComponentSource react-component-library:CopyableText */
import { translate } from "../../../../hooks/useLocale/versions/1.0.0/useLocale";

export function CopyableText({ value }: { value: string }) {
  return (
    <button data-testid="primitives.copyable-text"
      type="button"
      aria-label={translate("primitives.copyable-text.aria-label.1", "Copy text")}
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
