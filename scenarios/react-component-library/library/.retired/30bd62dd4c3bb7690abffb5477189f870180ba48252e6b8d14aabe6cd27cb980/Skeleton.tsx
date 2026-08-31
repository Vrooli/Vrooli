/**
 * @libraryId react-component-library:Skeleton
 * @displayName Skeleton
 * @description
 * @version 1.0.4
 * @tags ["primitive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource primitives.skeleton */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import type { HTMLAttributes } from "react";
import "./Skeleton.css";

export const Skeleton = withClassName(function Skeleton({
  label,
  style,
  ...props
}: HTMLAttributes<HTMLDivElement> & { label?: string }) {
  const libraryStrings = useStrings();
  label = label ?? libraryStrings("primitives.skeleton.loading", "Loading");
  return (
    <div
      data-testid="primitives.skeleton"
      role="status"
      aria-label={label}
      data-skeleton="true"
      data-rcl-skeleton
      style={style}
      {...props}
    />
  );
});
