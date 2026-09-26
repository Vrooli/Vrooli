import type { ComponentProps, ReactNode } from "react";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1.3.1";

type PlannerDialogProps = Omit<ComponentProps<typeof ResponsiveDialog>, "title"> & {
  title: ReactNode;
  description?: ReactNode;
};

export function PlannerDialog({ title, description, ariaLabel, contentPadding = "comfortable", ...props }: PlannerDialogProps) {
  return <ResponsiveDialog
    {...props}
    ariaLabel={ariaLabel ?? (typeof title === "string" ? title : undefined)}
    contentPadding={contentPadding}
    title={<span className="planner-dialog-title"><span>{title}</span>{description && <small>{description}</small>}</span>}
  />;
}
