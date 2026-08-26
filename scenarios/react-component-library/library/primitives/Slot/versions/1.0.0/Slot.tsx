/** @vrooliComponentSource primitives.slot */
import {
  cloneElement,
  isValidElement,
  type HTMLAttributes,
  type ReactElement,
} from "react";

export function Slot({ children, ...props }: HTMLAttributes<HTMLElement>) {
  if (!isValidElement(children)) return null;
  return cloneElement(children as ReactElement, props);
}
