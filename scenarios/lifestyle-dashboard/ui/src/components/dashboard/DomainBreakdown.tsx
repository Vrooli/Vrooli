/**
 * DomainBreakdown displays a horizontal bar chart of events per domain.
 * Shows relative distribution of activity across registered domains.
 *
 * [REQ:LD-QUERY-AGGREGATE] - Domain-level aggregation display
 */

interface DomainBreakdownItem {
  domain: string;
  count: number;
}

interface DomainBreakdownProps {
  data: DomainBreakdownItem[];
}

export function DomainBreakdown({ data }: DomainBreakdownProps) {
  if (!data || data.length === 0) {
    return <p className="text-slate-500 text-sm">No events recorded yet</p>;
  }

  const maxCount = Math.max(...data.map((d) => d.count), 1);

  return (
    <div className="space-y-3">
      {data.map((item) => {
        const width = (item.count / maxCount) * 100;
        return (
          <div key={item.domain}>
            <div className="flex justify-between text-sm mb-1">
              <span className="text-slate-300 truncate">{item.domain}</span>
              <span className="text-slate-500">{item.count}</span>
            </div>
            <div className="h-2 bg-slate-800 rounded-full overflow-hidden">
              <div
                className="h-full bg-gradient-to-r from-violet-500 to-fuchsia-500 rounded-full"
                style={{ width: `${width}%` }}
              />
            </div>
          </div>
        );
      })}
    </div>
  );
}
