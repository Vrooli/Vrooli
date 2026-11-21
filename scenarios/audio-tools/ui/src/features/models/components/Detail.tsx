import type { ReactNode } from "react";
import { cn } from "../../../lib/utils";

interface Props {
  label: string;
  children: ReactNode;
  className?: string;
}

/** Definition-list pair used by registry detail rows and the resolve result. */
export function Detail({ label, children, className }: Props) {
  return (
    <div className={cn("flex flex-col gap-0.5", className)}>
      <dt className="font-medium text-app-muted-foreground">{label}</dt>
      <dd className="break-words text-app-foreground">{children}</dd>
    </div>
  );
}
