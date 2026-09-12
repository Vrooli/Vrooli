/**
 * @libraryId react-component-library:Spinner
 * @displayName Spinner
 * @description The restrained indeterminate progress indicator with size, accessible label, appearance delay, contrast handling, and reduced-motion behavior.
 * @version 1.0.6
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource react-component-library:Spinner
 * @vrooliComponentSourceSlot primitives.spinner */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import type { HTMLAttributes } from "react";
import "./Spinner.css";

export const Spinner = withClassName(function Spinner({
  label,
  style,
  ...props
}: HTMLAttributes<HTMLDivElement> & { label?: string }) {
  const libraryStrings = useStrings();
  label = label ?? libraryStrings("primitives.spinner.loading", "Loading");
  return (
    <div
      data-testid="primitives.spinner"
      role="status"
      aria-label={label}
      data-spinner="true"
      data-rcl-spinner
      style={style}
      {...props}
    />
  );
});
