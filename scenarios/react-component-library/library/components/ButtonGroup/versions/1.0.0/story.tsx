import { Button } from "@vrooli/react-component-library/Button/2.0.0";
import { ButtonGroup } from "./ButtonGroup";

export function ButtonGroupStory({ args }: StoryHarnessProps<{ label?: string }>) {
  return (
    <ButtonGroup label={args.label ?? "Actions"}>
      <Button variant="secondary">Save</Button>
      <Button variant="primary">Publish</Button>
    </ButtonGroup>
  );
}

export const Default = ButtonGroupStory;
