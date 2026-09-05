/**
 * @libraryId react-component-library:Slot
 * @displayName Slot
 * @version 1.0.2
 * @tags ["primitive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource primitives.slot */
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
