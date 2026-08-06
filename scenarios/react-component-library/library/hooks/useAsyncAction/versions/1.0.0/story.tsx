import { useAsyncAction } from "./useAsyncAction";
export function Default({ log }: StoryHarnessProps) {
  const action = useAsyncAction(() => Promise.resolve("done"));
  return (
    <button
      type="button"
      onClick={() => {
        void action.run().then(() => log("completed"));
      }}
    >
      {action.status}
    </button>
  );
}
