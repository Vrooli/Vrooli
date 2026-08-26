import { ButtonGroup } from "./ButtonGroup";

// ButtonGroup 1.1.1 reuses the verified 1.0.0 specimen contract.
export { ButtonGroupStory } from "../1.0.0/story";

export function Default({
  args,
}: {
  args: Record<string, unknown>;
}) {
  return (
    <ButtonGroup label={String(args.label ?? "Actions")}>
      <button type="button">Actions</button>
      <button type="button">Save</button>
    </ButtonGroup>
  );
}
