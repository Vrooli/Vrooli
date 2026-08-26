/**
 * @libraryId react-component-library:CartesianCharts
 * @displayName Cartesian Charts
 * @description Accessible line, area, bar, stacked-bar, scatter, and histogram chart family.
 * @version 1.0.2
 * @tags ["visualization","charts","accessible","responsive"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:CartesianCharts */
import { withClassName } from "../../../../foundations/ClassMerge/versions/1.0.1/ClassMerge";

import { Chart, type ChartDatum, type ChartProps } from "../../../Chart/versions/1.0.0/Chart";

export type CartesianChartKind = "line" | "area" | "bar" | "stacked-bar" | "scatter" | "histogram";
export interface CartesianChartsProps extends Omit<ChartProps, "data" | "title"> {
  data: ChartDatum[];
  title: string;
  kind?: CartesianChartKind;
}

export const CartesianCharts = withClassName(function CartesianCharts({
  kind = "line",
  description,
  ...props
}: CartesianChartsProps) {
  const kindDescription = `${kind} chart. ${description ?? "Select a value from the keyboard-readable legend."}`;
  return <Chart {...props} description={kindDescription} />;
});
