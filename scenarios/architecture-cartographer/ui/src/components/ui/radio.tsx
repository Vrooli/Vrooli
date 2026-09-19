import * as React from "react";
import { cn } from "../../lib/utils";

export type RadioProps = Omit<React.InputHTMLAttributes<HTMLInputElement>, "type">;

const Radio = React.forwardRef<HTMLInputElement, RadioProps>(({ className, ...props }, ref) => (
  <input
    ref={ref}
    type="radio"
    className={cn(
      "h-4 w-4 border border-app-border bg-app-surface text-app-primary accent-app-primary focus:outline-none focus:ring-2 focus:ring-app-focus disabled:cursor-not-allowed disabled:opacity-60",
      className,
    )}
    {...props}
  />
));
Radio.displayName = "Radio";

export { Radio };
