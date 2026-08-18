import { FindingList } from "./FindingList";
export default function FindingListStory() {
  return (
    <FindingList
      findings={[
        {
          id: "a11y",
          assetId: "score-gauge",
          severity: "error",
          message: "Missing evidence",
          remediation: "Capture the declared viewports.",
        },
      ]}
    />
  );
}
