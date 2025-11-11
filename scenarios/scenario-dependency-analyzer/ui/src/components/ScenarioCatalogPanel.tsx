import { useMemo, useState } from "react";
import { RefreshCw, Search } from "lucide-react";
import type { ScenarioSummary } from "../types";
import { Button } from "./ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "./ui/card";
import { Input } from "./ui/input";

interface ScenarioCatalogPanelProps {
  scenarios: ScenarioSummary[];
  selected?: string | null;
  loading: boolean;
  onSelect: (name: string) => void;
  onRefresh: () => void;
}

export function ScenarioCatalogPanel({ scenarios, selected, loading, onSelect, onRefresh }: ScenarioCatalogPanelProps) {
  const [query, setQuery] = useState("");

  const filtered = useMemo(() => {
    if (!query) return scenarios;
    const lowered = query.toLowerCase();
    return scenarios.filter((scenario) =>
      scenario.display_name?.toLowerCase().includes(lowered) || scenario.name.toLowerCase().includes(lowered)
    );
  }, [query, scenarios]);

  return (
    <Card className="glass border border-border/40">
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="text-base">Scenario Catalog</CardTitle>
        <Button variant="ghost" size="icon" onClick={onRefresh} disabled={loading} aria-label="Refresh scenario list">
          <RefreshCw className={loading ? "h-4 w-4 animate-spin" : "h-4 w-4"} />
        </Button>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Search scenarios"
            className="pl-9"
          />
        </div>
        <div className="max-h-[360px] space-y-2 overflow-y-auto pr-1">
          {filtered.map((scenario) => (
            <button
              key={scenario.name}
              onClick={() => onSelect(scenario.name)}
              className={`w-full rounded-xl border px-3 py-2 text-left text-sm transition hover:border-primary/60 hover:bg-primary/5 ${
                selected === scenario.name ? "border-primary bg-primary/10" : "border-border/60"
              }`}
            >
              <div className="font-medium text-foreground">{scenario.display_name || scenario.name}</div>
              <div className="text-xs text-muted-foreground">
                {scenario.last_scanned ? new Date(scenario.last_scanned).toLocaleString() : "Not scanned yet"}
              </div>
            </button>
          ))}
          {!filtered.length && !loading ? (
            <div className="rounded-lg border border-dashed border-border/60 px-3 py-6 text-center text-xs text-muted-foreground">
              No scenarios match your search.
            </div>
          ) : null}
        </div>
      </CardContent>
    </Card>
  );
}
