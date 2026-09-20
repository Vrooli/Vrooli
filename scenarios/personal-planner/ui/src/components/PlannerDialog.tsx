import type { ComponentProps } from "react";
import { Dialog as SharedDialog } from "@vrooli/react-component-library/Dialog/1.3.8";

type PlannerDialogProps = ComponentProps<typeof SharedDialog>;

export function PlannerDialog(props: PlannerDialogProps) {
  return <SharedDialog {...props} />;
}
