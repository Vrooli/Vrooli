/**
 * @libraryId react-component-library:CopyableText
 * @displayName CopyableText
 * @description A compact text value with a clear copy affordance and stable feedback state.
 * @version 1.0.6
 * @tags ["primitive","feedback","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:CopyableText */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

export const CopyableText = withClassName(function CopyableText({ value }: { value: string }) {
  const strings = useStrings();
  return (
    <button
      data-testid="primitives.copyable-text"
      type="button"
      aria-label={strings("primitives.copyable-text.copy-text", "Copy text")}
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
});
