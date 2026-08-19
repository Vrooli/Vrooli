import { BoundedMeter } from "./BoundedMeter";

export function Default({ args }: StoryHarnessProps) {
  return (
    <BoundedMeter
      {...args}
      label="Capacity"
      value={72}
      max={100}
      valueText="72%"
      description="72% used"
    />
  );
}
