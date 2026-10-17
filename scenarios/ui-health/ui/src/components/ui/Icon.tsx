import * as React from "react";
import type { LucideIcon, LucideProps } from "lucide-react";

import { cn } from "../../lib/utils";

export interface IconProps extends Omit<LucideProps, "ref"> {
  icon: LucideIcon;
  label?: string;
}

export const Icon = React.forwardRef<SVGSVGElement, IconProps>(function Icon(
  { icon: Lucide, label, className, ...props },
  ref,
) {
  const a11y = label
    ? { role: "img" as const, "aria-label": label }
    : { "aria-hidden": true };
  return <Lucide ref={ref} className={cn("h-4 w-4", className)} {...a11y} {...props} />;
});
