import { BoundedMeter } from "./BoundedMeter";

export function Default() {
  return <BoundedMeter label="Capacity" value={35} valueText="35%" tone="neutral" />;
}
export function Warning() {
  return (
    <BoundedMeter label="Capacity" value={85} valueText="85% \u2014 near capacity" tone="warning" />
  );
}
export function ToneSuccess() {
  return <BoundedMeter label="Capacity" value={65} valueText="65%" tone="success" />;
}
export function ToneDanger() {
  return <BoundedMeter label="Capacity" value={65} valueText="65%" tone="danger" />;
}
