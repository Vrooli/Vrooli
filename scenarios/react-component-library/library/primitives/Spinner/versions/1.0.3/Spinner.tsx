/**
 * @libraryId react-component-library:Spinner
 * @displayName Spinner
 * @description
 * @version 1.0.3
 * @tags ["primitive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

/** @vrooliComponentSource primitives.spinner */
import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import type { HTMLAttributes } from "react";
import "./Spinner.css";

export const Spinner = withClassName(function Spinner({
  label = useStrings("primitives.spinner.loading", "Loading"),
  style,
  ...props
}: HTMLAttributes<HTMLDivElement> & { label?: string }) {
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
