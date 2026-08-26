import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { CartesianCharts } from "./CartesianCharts";

const data = [
  { id: "one", label: "v1.0.0", value: 42, detail: "released" },
  { id: "two", label: "v1.1.0", value: 68, detail: "released" },
  { id: "three", label: "v1.2.0", value: 84, detail: "latest" },
];

export function Default() {
  const libraryStrings = useStrings();
  return (
    <CartesianCharts
      title={libraryStrings("visualization.cartesian-charts.title", "Progression")}
      data={data}
    />
  );
}
export function Area() {
  const libraryStrings = useStrings();
  return (
    <CartesianCharts
      title={libraryStrings("visualization.cartesian-charts.title", "Progression")}
      data={data}
      kind="area"
    />
  );
}
export function Bar() {
  const libraryStrings = useStrings();
  return (
    <CartesianCharts
      title={libraryStrings("visualization.cartesian-charts.title", "Progression")}
      data={data}
      kind="bar"
    />
  );
}
export function StackedBar() {
  const libraryStrings = useStrings();
  return (
    <CartesianCharts
      title={libraryStrings("visualization.cartesian-charts.title", "Progression")}
      data={data}
      kind="stacked-bar"
    />
  );
}
export function Scatter() {
  const libraryStrings = useStrings();
  return (
    <CartesianCharts
      title={libraryStrings("visualization.cartesian-charts.title", "Progression")}
      data={data}
      kind="scatter"
    />
  );
}
export function Histogram() {
  const libraryStrings = useStrings();
  return (
    <CartesianCharts
      title={libraryStrings("visualization.cartesian-charts.title", "Progression")}
      data={data}
      kind="histogram"
    />
  );
}
