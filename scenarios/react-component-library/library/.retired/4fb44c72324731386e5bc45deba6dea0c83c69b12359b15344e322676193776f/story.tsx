import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
  type TooltipPlacement,
} from "./Tooltip";

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
  border:
    "1px solid var(--color-border-strong, color-mix(in srgb, var(--color-border) 72%, var(--color-foreground)))",
  borderRadius: "var(--radius-control, 0.375rem)",
  padding: "0 var(--space-md, 24px)",
  background: "var(--color-surface, #ffffff)",
  color: "var(--color-foreground, #0f172a)",
  font: "var(--text-label, 500 var(--text-label-size) / var(--text-label-line) var(--font-sans))",
};

export function Default({ args }: TooltipStoryProps) {
  return (
    <div
      style={{
        display: "grid",
        minHeight: 180,
        placeItems: "center",
        padding: "var(--space-2xl, 48px)",
      }}
    >
      <Tooltip {...args}>
        <TooltipTrigger aria-label="Show tooltip">Show tooltip</TooltipTrigger>
        <TooltipContent>{args.label}</TooltipContent>
      </Tooltip>
    </div>
  );
}

export function Open({ args }: TooltipStoryProps) {
  return (
    <div
      style={{
        display: "grid",
        minHeight: 180,
        placeItems: "center",
        padding: "var(--space-2xl, 48px)",
      }}
    >
      <Tooltip {...args} defaultOpen>
        <TooltipTrigger style={triggerStyle} aria-label="Show tooltip">
          Maya Chen
        </TooltipTrigger>
        <TooltipContent>{args.label}</TooltipContent>
      </Tooltip>
    </div>
  );
}

export function Placements({ args }: TooltipStoryProps) {
  return (
    <div
      style={{
        display: "grid",
        minHeight: 180,
        placeItems: "center",
        padding: "var(--space-2xl, 48px)",
      }}
    >
      <Tooltip {...args}>
        <TooltipTrigger style={triggerStyle} aria-label="Show tooltip">
          Responsive trigger
        </TooltipTrigger>
        <TooltipContent>{args.label}</TooltipContent>
      </Tooltip>
    </div>
  );
}
