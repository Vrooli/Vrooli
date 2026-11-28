import type { CardProps } from "@vrooli/react-component-library/Card/1";
import { Card } from "@vrooli/react-component-library/Card/1";

/** Scenario content surface backed by the governed Card primitive. */
export function Panel({ children, className, ...props }: CardProps) {
  return <Card {...props} className={className}>{children}</Card>;
}
