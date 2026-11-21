import { useState } from "react";
import ColorPicker from "./ColorPicker";

export function ControlledWithRecents({
  args,
  log,
}: StoryHarnessProps<{ value: string; palette: string[] }>) {
  const [value, setValue] = useState(args.value);
  const [recents, setRecents] = useState<string[]>([]);
  return (
    <div className="space-y-space-xs">
      <ColorPicker
        palette={args.palette}
        value={value}
        onChange={(next) => {
          setValue(next);
          log("change", next);
        }}
        recentColors={recents}
        onRecordRecent={(color) => {
          setRecents((current) =>
            [color, ...current.filter((entry) => entry !== color)].slice(0, 5),
          );
          log("recordRecent", color);
        }}
        labels={{ heading: "Choose a color" }}
      />
      <output data-testid="selected-color">Selected: {value}</output>
    </div>
  );
}

export function RecentsCommitOnBlur({
  args,
  log,
}: StoryHarnessProps<{ value: string; palette: string[] }>) {
  const [value, setValue] = useState(args.value);
  const [recents, setRecents] = useState<string[]>([]);
  const record = (color: string) => {
    setRecents((current) => [color, ...current.filter((entry) => entry !== color)].slice(0, 5));
    log("recordRecent", color);
  };
  return (
    <div className="space-y-space-xs">
      <ColorPicker
        palette={args.palette}
        value={value}
        onChange={(next) => {
          setValue(next);
          log("change", next);
        }}
        recentColors={recents}
        onRecordRecent={record}
        labels={{ heading: "Choose a color", recents: "Recent colors" }}
      />
      <output>{recents.map((color) => `Recent: ${color}`).join(", ") || "No recents"}</output>
    </div>
  );
}
