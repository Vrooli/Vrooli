/** @vrooliComponentSource react-component-library:CartesianCharts */
import {
  Chart,
  type ChartDatum,
  type ChartProps,
} from "@vrooli/react-component-library/Chart/1.0.0";

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
