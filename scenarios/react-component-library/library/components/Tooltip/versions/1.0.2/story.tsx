import { Tooltip, TooltipContent, TooltipTrigger, type TooltipPlacement } from "./Tooltip";

type TooltipStoryArgs = {
  label: string;
  placement: TooltipPlacement;
  delay?: number;
  closeDelay?: number;
  defaultOpen?: boolean;
};

type TooltipStoryProps = {
  args: TooltipStoryArgs;
};

const triggerStyle = {
  minHeight: 44,
  border: "1px solid var(--color-border-strong, #94a3b8)",
  borderRadius: "var(--radius-control, .625rem)",
  padding: "0 var(--space-md, 1rem)",
  background: "var(--color-surface, #fff)",
  color: "var(--color-foreground, #0f172a)",
  font: "var(--text-label, 600 .8125rem/1.25rem system-ui, sans-serif)",
};

export function Default({ args }: TooltipStoryProps) {
  return (
    <div style={{ display: "grid", minHeight: 180, placeItems: "center", padding: "var(--space-2xl, 3rem)" }}>
      <Tooltip {...args}>
        <TooltipTrigger aria-label="Show tooltip">Show tooltip</TooltipTrigger>
        <TooltipContent>{args.label}</TooltipContent>
      </Tooltip>
    </div>
  );
}

export function Open({ args }: TooltipStoryProps) {
  return (
    <div style={{ display: "grid", minHeight: 180, placeItems: "center", padding: "var(--space-2xl, 3rem)" }}>
      <Tooltip {...args} defaultOpen>
        <TooltipTrigger style={triggerStyle} aria-label="Show tooltip">Maya Chen</TooltipTrigger>
        <TooltipContent>{args.label}</TooltipContent>
      </Tooltip>
    </div>
  );
}

export function Placements({ args }: TooltipStoryProps) {
  return (
    <div style={{ display: "grid", minHeight: 180, placeItems: "center", padding: "var(--space-2xl, 3rem)" }}>
      <Tooltip {...args}>
        <TooltipTrigger style={triggerStyle} aria-label="Show tooltip">Responsive trigger</TooltipTrigger>
        <TooltipContent>{args.label}</TooltipContent>
      </Tooltip>
    </div>
  );
}
