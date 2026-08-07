import { Chart, type ChartDatum } from "./Chart";

const data: ChartDatum[] = [
  { id: "mon", label: "Mon", value: 42, detail: "steady" },
  { id: "tue", label: "Tue", value: 58, detail: "+38%" },
  { id: "wed", label: "Wed", value: 51, detail: "steady" },
  { id: "thu", label: "Thu", value: 76, detail: "+49%" },
  { id: "fri", label: "Fri", value: 68, detail: "healthy" },
  { id: "sat", label: "Sat", value: 84, detail: "peak" },
];

export function Default() {
  return <Chart title="Weekly activation" data={data} />;
}

export function Loading() {
  return <Chart title="Weekly activation" data={data} status="pending" />;
}

export function Refreshing() {
  return <Chart title="Weekly activation" data={data} status="refreshing" />;
}

export function Empty() {
  return (
    <Chart
      title="Weekly activation"
      data={[]}
      status="empty"
      emptyMessage="Connect a data source to see this trend."
    />
  );
}

export function RequestError() {
  return (
    <Chart
      title="Weekly activation"
      data={data}
      status="error"
      onRetry={() => undefined}
    />
  );
}

export function Offline() {
  return <Chart title="Weekly activation" data={data} status="offline" />;
}
