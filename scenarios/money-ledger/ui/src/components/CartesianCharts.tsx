/**
 * @vrooliComponentSource react-component-library:CartesianCharts
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 4d11cc7f-ae55-45d7-b8d3-76906825794f
 * @vrooliComponentAppliedAt 2026-08-17T08:31:47Z
 * @vrooliComponentSourceSha256 dec464a543ba04542d107afb2f065912296780fedab8564205909e1960a75cea
 * @vrooliComponentDriftHash 8f5210cff3d7b60239f12c2e668b5f78b0f728ef1936600d17eb789c112bc2cf
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import {
  Chart,
  type ChartDatum,
  type ChartProps,
} from "./Chart";

export type CartesianChartKind =
  | "line"
  | "area"
  | "bar"
  | "stacked-bar"
  | "scatter"
  | "histogram";
export interface CartesianChartsProps
  extends Omit<ChartProps, "data" | "title"> {
  data: ChartDatum[];
  title: string;
  kind?: CartesianChartKind;
}

export function CartesianCharts({
  kind = "line",
  description,
  ...props
}: CartesianChartsProps) {
  const kindDescription = `${kind} chart. ${description ?? "Select a value from the keyboard-readable legend."}`;
  return <Chart {...props} description={kindDescription} />;
}
