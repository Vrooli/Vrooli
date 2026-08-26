import { useState } from "react";
import { Dialog, type DialogProps } from "./Dialog";

type DialogStoryProps = {
  args: DialogProps;
  log: (event: string, value?: unknown) => void;
};

/** Controlled harness proving the complete open/close interaction loop. */
export function Interactive({ args, log }: DialogStoryProps) {
  const [open, setOpen] = useState(args.open);
  const close = () => {
    setOpen(false);
    log("dialog-close");
  };

  return (
    <div className="grid min-h-content place-items-center gap-space-md p-space-lg">
      <button
        type="button"
        className="rounded-control bg-app-primary px-space-md py-space-xs text-app-primary-foreground"
        onClick={() => {
          setOpen(true);
          log("dialog-open");
        }}
      >
        Open dialog
      </button>
      <Dialog {...args} open={open} onClose={close} />
    </div>
  );
}

export const ConfirmOpen = Interactive;
export const DetailsOpen = Interactive;
export const MinimalOpen = Interactive;
