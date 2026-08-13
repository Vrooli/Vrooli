/** @vrooliComponentSource visualization.cartesian-charts */

import {
  CartesianCharts as PublishedCartesianCharts,
  type CartesianChartKind,
  type CartesianChartsProps,
} from "../../../../../../../library/components/CartesianCharts/versions/1.0.0/CartesianCharts";

// This is the adopted asset with a host-level test hook and a product-specific
// empty state. The plot, interaction model, keyboard legend, and accessible
// table all come from the published versioned component above.
export function CartesianCharts(props: CartesianChartsProps) {
  return (
    <div data-testid="cartesian-charts">
      <PublishedCartesianCharts
        {...props}
        emptyMessage={
          props.emptyMessage ?? <span role="status">No progression data is available.</span>
        }
      />
    </div>
  );
}

export type { CartesianChartKind, CartesianChartsProps };
export type { ChartDatum as CartesianPoint } from "../../../../../../../library/components/Chart/versions/1.0.0/Chart";
