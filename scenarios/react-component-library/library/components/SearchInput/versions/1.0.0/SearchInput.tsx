/** @vrooliComponentSource react-component-library:SearchInput */
import { translate } from "../../../../hooks/useLocale/versions/1.0.0/useLocale";

import { forwardRef, type InputHTMLAttributes } from "react";
const muted = { color: "var(--color-muted-foreground, #64748b)" };
export const SearchInput = forwardRef<
  HTMLInputElement,
  InputHTMLAttributes<HTMLInputElement>
>(function SearchInput({ placeholder = translate("forms.search-input.placeholder.1", "Search"), style, ...props }, ref) {
  return (
    <label style={{ display: "grid", gap: 6, width: "min(100%, 360px)" }}>
      <span
        style={{
          ...muted,
          fontSize: 12,
          fontWeight: 700,
          textTransform: "uppercase",
        }}
      >
        {translate("forms.search-input.text.2", "Search")}
      </span>
      <input data-testid="forms.search-input"
        ref={ref}
        type="search"
        placeholder={placeholder}
        style={{
          minHeight: 44,
          boxSizing: "border-box",
          width: "100%",
          border: "1px solid var(--color-border, #cbd5e1)",
          borderRadius: "var(--radius-control, .5rem)",
          background: "var(--color-surface, #fff)",
          color: "var(--color-foreground, #0f172a)",
          paddingInline: 16,
          font: "inherit",
          ...style,
        }}
        {...props}
      />
    </label>
  );
});
