import { Button } from "@vrooli/react-component-library/Button/2";
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

export function Segmented() {
  return (
    <ButtonGroup label="Layout" variant="segmented" collapse="never">
      <Button variant="ghost" shape="square" size="lg" aria-pressed="false">
        Grid
      </Button>
      <Button variant="ghost" shape="square" size="lg" aria-pressed="true">
        Sidebar
      </Button>
    </ButtonGroup>
  );
}
