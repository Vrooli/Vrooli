import { MetricBreakdown } from "./MetricBreakdown";
export default function MetricBreakdownStory() {
  return (
    <MetricBreakdown
      items={[{ id: "passed", label: "Passed", value: 8, total: 10 }]}
    />
  );
}
