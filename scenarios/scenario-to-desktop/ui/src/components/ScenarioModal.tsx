import { useEffect, useMemo, useState } from "react";
import { createPortal } from "react-dom";
import { Clock, Search, X } from "lucide-react";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "./ui/card";
import { Input } from "./ui/input";
import type { ScenarioDesktopStatus } from "./scenario-inventory/types";

const RECENTS_KEY = "scenario-to-desktop:recents";
const MAX_RECENTS = 6;

type ScenarioCardData = {
  name: string;
  displayName?: string;
  hasDesktop?: boolean;
  isKnown?: boolean;
};

function readRecents(): string[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = window.localStorage.getItem(RECENTS_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((entry) => typeof entry === "string");
  } catch {
    return [];
  }
}

function writeRecents(recents: string[]) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(RECENTS_KEY, JSON.stringify(recents.slice(0, MAX_RECENTS)));
}

function mergeRecents(recents: string[], name: string): string[] {
  const cleaned = name.trim();
  if (!cleaned) return recents;
  return [cleaned, ...recents.filter((entry) => entry !== cleaned)].slice(0, MAX_RECENTS);
}

interface ScenarioModalProps {
  open: boolean;
  loading: boolean;
  scenarios: ScenarioDesktopStatus[];
  selectedScenarioName: string;
  onSelect: (name: string) => void;
  onClose: () => void;
}

export function ScenarioModal({
  open,
  loading,
  scenarios,
  selectedScenarioName,
  onSelect,
  onClose
}: ScenarioModalProps) {
  const [search, setSearch] = useState("");
  const [recents, setRecents] = useState<string[]>([]);

  useEffect(() => {
    if (!open) return;
    setRecents(readRecents());
  }, [open]);

  useEffect(() => {
    if (!open) {
      setSearch("");
    }
  }, [open]);

  const normalized = search.trim().toLowerCase();
  const filteredScenarios = useMemo(() => {
    if (!normalized) return scenarios;
    return scenarios.filter((scenario) => {
      const nameMatch = scenario.name.toLowerCase().includes(normalized);
      const displayMatch = scenario.display_name?.toLowerCase().includes(normalized);
      return nameMatch || displayMatch;
    });
  }, [normalized, scenarios]);

  const recentsData = useMemo<ScenarioCardData[]>(() => {
    if (recents.length === 0) return [];
    return recents.map((name) => {
      const scenario = scenarios.find((item) => item.name === name);
      if (!scenario) {
        return { name, displayName: name, isKnown: false };
      }
      return {
        name: scenario.name,
        displayName: scenario.display_name,
        hasDesktop: scenario.has_desktop,
        isKnown: true
      };
    });
  }, [recents, scenarios]);

  const hasExactMatch = normalized
    ? scenarios.some((scenario) => scenario.name.toLowerCase() === normalized)
    : false;

  const handleSelect = (name: string) => {
    const nextRecents = mergeRecents(recents, name);
    setRecents(nextRecents);
    writeRecents(nextRecents);
    onSelect(name);
  };

  if (!open) return null;

  return createPortal(
    <div
      className="fixed inset-0 z-[99999] flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
      onClick={(event) => {
        if (event.target === event.currentTarget) {
          onClose();
        }
      }}
    >
      <Card className="w-full max-w-5xl border-slate-800 bg-slate-950/90 shadow-xl max-h-[90vh] flex flex-col">
        <CardHeader className="flex flex-row items-start justify-between gap-4">
          <div className="space-y-1">
            <CardTitle className="text-lg text-slate-100">Choose a scenario</CardTitle>
            <p className="text-sm text-slate-400">
              Search by name or slug. Recent scenarios are highlighted for faster access.
            </p>
          </div>
          <Button type="button" variant="outline" size="sm" onClick={onClose} className="h-8 w-8 p-0">
            <X className="h-4 w-4" />
          </Button>
        </CardHeader>
        <CardContent className="space-y-5 overflow-y-auto">
          <div className="flex flex-wrap items-center gap-3 rounded-lg border border-slate-800/80 bg-slate-950/60 p-3">
            <div className="flex flex-1 items-center gap-2 text-slate-400">
              <Search className="h-4 w-4" />
              <Input
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder="Search scenarios..."
                className="h-9 border-0 bg-transparent p-0 text-sm text-slate-100 focus-visible:ring-0"
              />
            </div>
            {normalized && !hasExactMatch && (
              <Button type="button" size="sm" onClick={() => handleSelect(search.trim())}>
                Use "{search.trim()}" slug
              </Button>
            )}
          </div>

          {recentsData.length > 0 && (
            <div className="space-y-3">
              <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-slate-400">
                <Clock className="h-3.5 w-3.5" />
                Recents
              </div>
              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                {recentsData.map((scenario) => (
                  <ScenarioCard
                    key={`recent-${scenario.name}`}
                    scenario={scenario}
                    selected={selectedScenarioName === scenario.name}
                    onSelect={() => handleSelect(scenario.name)}
                  />
                ))}
              </div>
            </div>
          )}

          <div className="space-y-3">
            <div className="flex items-center justify-between text-xs uppercase tracking-wide text-slate-400">
              <span>All scenarios</span>
              {!loading && (
                <span>{filteredScenarios.length} total</span>
              )}
            </div>
            {loading ? (
              <div className="rounded-lg border border-slate-800 bg-slate-950/60 p-6 text-center text-sm text-slate-400">
                Loading scenarios...
              </div>
            ) : filteredScenarios.length === 0 ? (
              <div className="rounded-lg border border-slate-800 bg-slate-950/60 p-6 text-center text-sm text-slate-400">
                No scenarios match that search.
              </div>
            ) : (
              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                {filteredScenarios.map((scenario) => (
                  <ScenarioCard
                    key={scenario.name}
                    scenario={{
                      name: scenario.name,
                      displayName: scenario.display_name,
                      hasDesktop: scenario.has_desktop,
                      isKnown: true
                    }}
                    selected={selectedScenarioName === scenario.name}
                    onSelect={() => handleSelect(scenario.name)}
                  />
                ))}
              </div>
            )}
          </div>
        </CardContent>
      </Card>
    </div>,
    document.body
  );
}

interface ScenarioCardProps {
  scenario: ScenarioCardData;
  selected: boolean;
  onSelect: () => void;
}

function ScenarioCard({ scenario, selected, onSelect }: ScenarioCardProps) {
  return (
    <button
      type="button"
      onClick={onSelect}
      className={`rounded-lg border border-white/10 p-4 text-left transition-all hover:scale-[1.01] ${
        selected
          ? "bg-blue-900/20 shadow-[inset_0_0_0_2px_rgba(59,130,246,0.9)] shadow-blue-900/40"
          : "bg-white/5 hover:shadow-[inset_0_0_0_1px_rgba(255,255,255,0.35)]"
      }`}
    >
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-sm font-semibold text-slate-100">
            {scenario.displayName || scenario.name}
          </p>
          <p className="text-xs text-slate-400">Slug: {scenario.name}</p>
        </div>
        {scenario.hasDesktop && (
          <Badge variant="info" className="text-[10px]">
            Desktop ready
          </Badge>
        )}
        {scenario.isKnown === false && (
          <Badge className="text-[10px]">
            Recent
          </Badge>
        )}
      </div>
    </button>
  );
}
