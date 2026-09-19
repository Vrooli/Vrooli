import { Chart } from "./Chart";

const data = [
  { id: "jan", label: "Jan", value: 42, detail: "Measured" },
  { id: "feb", label: "Feb", value: 58, detail: "Measured" },
  { id: "mar", label: "Mar", value: 71, detail: "Measured" },
];

export function Default() {
  return <Chart data={data} title="Monthly performance" description="A compact trend specimen for the catalog." />;
}
