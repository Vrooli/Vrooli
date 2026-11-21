type CalibrationItem = { id: string; label: string };
type UsageContractCalibrationProps = { items: CalibrationItem[] };

export function UsageContractCalibration({ items }: UsageContractCalibrationProps) {
  return <ul>{items.map((item) => <li key={item.id}>{item.label}</li>)}</ul>;
}
