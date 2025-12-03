import { ArrowRight } from "lucide-react";
import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";

interface GuidedFlowItem {
  key: string;
  step: string;
  title: string;
  description: string;
  stat: string;
  statLabel: string;
  action: string;
  onClick: () => void;
}

interface GuidedFlowsProps {
  items: GuidedFlowItem[];
}

export function GuidedFlows({ items }: GuidedFlowsProps) {
  return (
    <section
      className="rounded-2xl border border-white/10 bg-white/[0.02] p-6"
      data-testid={selectors.dashboard.guidedFlows}
    >
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Workflows</p>
          <h2 className="mt-2 text-2xl font-semibold">Common tasks</h2>
          <p className="mt-2 text-sm text-slate-300">
            Quick shortcuts to the most common testing workflows.
          </p>
        </div>
      </div>
      <div className="mt-5 grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        {items.map((item) => (
          <article
            key={item.key}
            className="flex flex-col rounded-2xl border border-white/5 bg-black/30 p-5"
          >
            <p className="text-[0.65rem] uppercase tracking-[0.4em] text-slate-400">
              {item.step}
            </p>
            <h3 className="mt-2 text-xl font-semibold">{item.title}</h3>
            <p className="mt-2 text-sm text-slate-300">{item.description}</p>
            <div className="mt-4">
              <p className="text-xs uppercase tracking-wide text-slate-400">{item.statLabel}</p>
              <p className="mt-1 text-lg font-semibold">{item.stat}</p>
            </div>
            <div className="mt-4">
              <Button variant="outline" size="sm" onClick={item.onClick}>
                {item.action}
                <ArrowRight className="ml-2 h-4 w-4" />
              </Button>
            </div>
          </article>
        ))}
      </div>
    </section>
  );
}
