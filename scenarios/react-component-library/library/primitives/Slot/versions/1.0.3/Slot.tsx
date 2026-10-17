/**
 * @libraryId react-component-library:Slot
 * @displayName Slot
 * @description The composition primitive that merges behavior, props, and refs onto a consumer-supplied element instead of introducing a wrapper, so every asset can offer an as-child escape hatch.
 * @version 1.0.3
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:Slot
 * @vrooliComponentSourceSlot primitives.slot */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import { cloneElement, isValidElement, type HTMLAttributes, type ReactElement } from "react";

export const Slot = withClassName(function Slot({
  children,
  ...props
}: HTMLAttributes<HTMLElement>) {
  if (!isValidElement(children)) return null;
  return cloneElement(children as ReactElement, {
    ...props,
    "data-testid": "primitives.slot",
  });
});
