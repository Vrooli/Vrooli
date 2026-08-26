/**
 * @libraryId react-component-library:CartesianCharts
 * @displayName Cartesian Charts
 * @description
 * @version 1.0.5
 * @tags ["visualization","charts","accessible","responsive"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:CartesianCharts */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import {
  Chart,
  type ChartDatum,
  type ChartProps,
} from "@vrooli/react-component-library/Chart/1.0.0";

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
  return (
    <Chart data-testid="visualization.cartesian-charts" {...props} description={kindDescription} />
  );
});
