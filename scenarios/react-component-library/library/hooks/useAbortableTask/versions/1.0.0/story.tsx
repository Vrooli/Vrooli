import { useState } from "react";
import { useAbortableTask } from "./useAbortableTask";
export function Default({ log }: StoryHarnessProps) {
  const [done, setDone] = useState(false);
  const { run } = useAbortableTask(() => Promise.resolve("done"));
  return (
    <button data-testid="hooks.use-abortable-task"
      type="button"
      onClick={() => {
        void run().then(() => {
          setDone(true);
          log("completed");
        });
      }}
    >
      Task {done ? "done" : "ready"}
    </button>
  );
}
