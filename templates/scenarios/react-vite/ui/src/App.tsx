import { useQuery } from "@tanstack/react-query";
import { ArrowRight } from "lucide-react";
import { Button } from "./components/ui/button";
import { strings } from "./consts/strings";
import { fetchHealth } from "./lib/api";

export default function App() {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth
  });

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50 flex flex-col items-center justify-center p-6">
      <div className="w-full max-w-xl rounded-2xl border border-white/10 bg-white/5 p-8 shadow-2xl backdrop-blur">
        <p className="text-sm uppercase tracking-[0.2em] text-slate-400">{strings.app.eyebrow}</p>
        <h1 className="mt-3 text-3xl font-semibold">{strings.app.title}</h1>
        <p className="mt-2 text-slate-300">{strings.app.description}</p>

        <div className="mt-6 rounded-xl border border-white/10 bg-black/20 p-4">
          <p className="text-sm font-medium text-slate-400">{strings.health.title}</p>
          {isLoading && <p className="mt-2 text-slate-200">{strings.health.loading}</p>}
          {error && (
            <p className="mt-2 text-red-400">{strings.health.error}</p>
          )}
          {data && (
            <div className="mt-2 text-sm text-slate-200">
              <p>{strings.health.statusLabel} {data.status}</p>
              <p>{strings.health.serviceLabel} {data.service}</p>
              <p>{strings.health.timestampLabel} {new Date(data.timestamp).toLocaleString()}</p>
            </div>
          )}
          <Button className="mt-4" onClick={() => refetch()}>
            {strings.health.refresh}
            <ArrowRight className="ms-2 h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
