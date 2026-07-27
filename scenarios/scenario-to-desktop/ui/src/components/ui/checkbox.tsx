import { cn } from "../../lib/utils";
import { useId } from "react";

export interface CheckboxProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string;
}

export function Checkbox({ className, label, ...props }: CheckboxProps) {
  const generatedId = useId();
  const id = props.id ?? generatedId;
  return (
    <div className="flex items-center gap-2">
      <input
        type="checkbox"
        className={cn(
          "h-4 w-4 rounded border-white/20 bg-white/5 text-blue-500 focus:ring-2 focus:ring-white/50 focus:ring-offset-0",
          className,
        )}
        id={id}
        {...props}
      />
      {label && <label htmlFor={id} className="text-sm text-slate-300">{label}</label>}
    </div>
  );
}
