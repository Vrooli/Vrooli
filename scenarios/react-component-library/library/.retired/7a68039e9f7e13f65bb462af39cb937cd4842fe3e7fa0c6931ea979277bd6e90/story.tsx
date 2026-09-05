import { useState } from "react";
import { usePress } from "./usePress";

export function Default({ args, log }: StoryHarnessProps<{ label: string }>) {
  const [pressed, setPressed] = useState(false);
  const press = usePress(() => {
    setPressed(true);
    log("pressed", args.label);
  });
  return (
    <button data-testid="hooks.use-press" type="button" {...press} aria-pressed={pressed}>
      {args.label}
    </button>
  );
}
