import type { ComponentProps, ReactNode } from "react";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1.3.8";
import { OverlayActionBar, type OverlayActionList } from "@vrooli/react-component-library/OverlayActionBar/1";

type PlannerDialogProps = Omit<ComponentProps<typeof ResponsiveDialog>, "title" | "footer"> & {
  title: ReactNode;
  description?: ReactNode;
  actions?: OverlayActionList;
};

export function PlannerDialog({ title, description, ariaLabel, contentPadding = "comfortable", actions, ...props }: PlannerDialogProps) {
  return <ResponsiveDialog
    {...props}
    footer={actions ? <OverlayActionBar className="flex-row-reverse" actions={actions} /> : undefined}
    ariaLabel={ariaLabel ?? (typeof title === "string" ? title : undefined)}
    contentPadding={contentPadding}
    title={<span className="planner-dialog-title"><span>{title}</span>{description && <small>{description}</small>}</span>}
  />;
}
