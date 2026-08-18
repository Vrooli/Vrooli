import { ProgressLadder } from "./ProgressLadder";
export default function ProgressLadderStory() {
  return (
    <ProgressLadder
      rungs={[
        { id: "implemented", label: "Implemented", complete: true },
        { id: "verified", label: "Verified", complete: false, current: true },
      ]}
    />
  );
}
