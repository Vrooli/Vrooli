import { Button } from "../../../Button/versions/2.0.0/Button";
import { ButtonGroup } from "./ButtonGroup";

export function ButtonGroupStory(args: { label?: string }) {
  return (
    <ButtonGroup label={args.label ?? "Actions"}>
      <Button variant="secondary">Save draft</Button>
      <Button variant="primary">Publish</Button>
    </ButtonGroup>
  );
}
