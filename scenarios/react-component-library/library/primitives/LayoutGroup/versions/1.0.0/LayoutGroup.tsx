/** @vrooliComponentSource motion.layout-group */
import type { HTMLAttributes } from "react";

export function LayoutGroup({
  children,
  ...props
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <div data-layout-group="true" {...props}>
      {children}
    </div>
  );
}
