import { useControllableState } from "./useControllableState";
export function Default({ log }: StoryHarnessProps) {
  const [value, setValue] = useControllableState({
    defaultValue: "ready",
    onChange: () => log("changed"),
  });
  return (
    <button data-testid="hooks.use-controllable-state" type="button" onClick={() => setValue("changed")}>
      {value}
    </button>
  );
}
