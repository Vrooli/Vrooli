import { useState } from "react";
import { createPortal } from "react-dom";
import { Cpu, Feather, Info, Shield, Sparkles, X } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "./ui/card";
import { Button } from "./ui/button";
import { Badge } from "./ui/badge";

type FrameworkId = "electron" | "tauri" | "neutralino";

const FRAMEWORKS: Array<{
  id: FrameworkId;
  name: string;
  status: "available" | "coming_soon";
  summary: string;
  strengths: string[];
  tradeoffs: string[];
  bestFor: string[];
}> = [
  {
    id: "electron",
    name: "Electron",
    status: "available",
    summary: "Chromium + Node.js, huge ecosystem, most compatibility.",
    strengths: ["Mature tooling", "Full Node access", "Largest community"],
    tradeoffs: ["Heavier bundles", "Higher memory footprint"],
    bestFor: ["Feature-rich apps", "Deep OS integration", "Fastest path to ship"]
  },
  {
    id: "tauri",
    name: "Tauri",
    status: "coming_soon",
    summary: "Rust backend + system webview for lighter, more secure apps.",
    strengths: ["Smaller binaries", "Lower memory use", "Security-focused APIs"],
    tradeoffs: ["Rust toolchain", "Webview differences across OSes"],
    bestFor: ["Performance-sensitive apps", "Smaller installs", "Security-first deployments"]
  },
  {
    id: "neutralino",
    name: "Neutralino",
    status: "coming_soon",
    summary: "Minimal wrapper using the OS webview with a tiny runtime.",
    strengths: ["Very small footprint", "Simple build output"],
    tradeoffs: ["Smaller ecosystem", "Limited native APIs"],
    bestFor: ["Lightweight utilities", "Simple dashboards", "Minimal surface area"]
  }
];

const FRAMEWORK_VISUALS: Record<
  FrameworkId,
  {
    icon: typeof Cpu;
    gradient: string;
    tagline: string;
  }
> = {
  electron: {
    icon: Cpu,
    gradient: "from-blue-700/80 via-indigo-600/70 to-slate-700/70",
    tagline: "Most compatible today"
  },
  tauri: {
    icon: Shield,
    gradient: "from-emerald-700/80 via-teal-600/70 to-slate-700/70",
    tagline: "Secure + lightweight"
  },
  neutralino: {
    icon: Feather,
    gradient: "from-amber-700/80 via-orange-600/70 to-slate-700/70",
    tagline: "Tiny runtime"
  }
};

interface FrameworkModalProps {
  open: boolean;
  selectedFramework: string;
  onSelect: (framework: FrameworkId) => void;
  onClose: () => void;
}

export function FrameworkModal({
  open,
  selectedFramework,
  onSelect,
  onClose
}: FrameworkModalProps) {
  const [showHelper, setShowHelper] = useState(false);

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
      <Card className="w-full max-w-5xl border-slate-800 bg-slate-950/90 shadow-xl">
        <CardHeader className="flex flex-row items-start justify-between gap-4">
          <div className="space-y-1">
            <CardTitle className="text-lg text-slate-100">Choose a framework</CardTitle>
            <p className="text-sm text-slate-400">
              Electron is supported today. Tauri and Neutralino are coming soon.
            </p>
          </div>
          <Button type="button" variant="outline" size="sm" onClick={onClose} className="h-8 w-8 p-0">
            <X className="h-4 w-4" />
          </Button>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-start gap-3 rounded-lg border border-slate-800/80 bg-slate-950/60 p-3 text-sm text-slate-200">
            <Info className="mt-0.5 h-5 w-5 text-blue-300" />
            <div className="space-y-1">
              <p className="font-semibold text-slate-100">Framework choice affects build tooling and runtime size.</p>
              <p className="text-slate-300">
                We will expand template support when Tauri and Neutralino scaffolds are ready.
              </p>
            </div>
          </div>

          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="text-sm text-slate-300">
              Current selection: <span className="font-semibold text-slate-100">{selectedFramework}</span>
            </p>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setShowHelper((prev) => !prev)}
              className="gap-2"
            >
              <Sparkles className="h-4 w-4" />
              Help me decide
            </Button>
          </div>

          {showHelper && (
            <div className="rounded-lg border border-blue-800/70 bg-blue-950/30 p-4 space-y-3">
              <p className="text-sm font-semibold text-slate-100">Quick guidance</p>
              <p className="text-sm text-slate-200">
                If you need maximum compatibility, OS integrations, or a proven pipeline, choose Electron. If you want
                smaller bundles and are comfortable with Rust tooling, Tauri is the future fit. If you want the smallest
                possible footprint for a simple utility, Neutralino is the lightest option.
              </p>
            </div>
          )}

          <div className="grid gap-4 md:grid-cols-3">
            {FRAMEWORKS.map((framework) => {
              const isSelected = selectedFramework === framework.id;
              const isAvailable = framework.status === "available";
              const visual = FRAMEWORK_VISUALS[framework.id];
              return (
                <div
                  key={framework.id}
                  className={`rounded-lg border border-slate-800/80 bg-slate-950/60 p-4 transition-all ${
                    isSelected ? "shadow-[inset_0_0_0_2px_rgba(59,130,246,0.9)]" : ""
                  } ${isAvailable ? "hover:scale-[1.01]" : "opacity-70"}`}
                >
                  <div className={`mb-3 flex items-center gap-3 rounded-md bg-gradient-to-r ${visual.gradient} p-3`}>
                    <visual.icon className="h-6 w-6 text-white/90" />
                    <div>
                      <p className="text-sm font-semibold text-white">{framework.name}</p>
                      <p className="text-xs text-slate-100/90">{visual.tagline}</p>
                    </div>
                  </div>

                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <p className="text-xs text-slate-400">{framework.summary}</p>
                    </div>
                    {framework.status === "coming_soon" && (
                      <Badge variant="outline" className="text-xs text-slate-300">
                        Coming soon
                      </Badge>
                    )}
                  </div>

                  <div className="mt-3 space-y-2 text-xs text-slate-300">
                    <div>
                      <p className="text-slate-200">Strengths</p>
                      <p>{framework.strengths.join(" · ")}</p>
                    </div>
                    <div>
                      <p className="text-slate-200">Tradeoffs</p>
                      <p>{framework.tradeoffs.join(" · ")}</p>
                    </div>
                    <div>
                      <p className="text-slate-200">Best for</p>
                      <p>{framework.bestFor.join(" · ")}</p>
                    </div>
                  </div>

                  <Button
                    type="button"
                    size="sm"
                    className="mt-4 w-full"
                    disabled={!isAvailable}
                    onClick={() => onSelect(framework.id)}
                  >
                    {isAvailable ? "Select" : "Coming soon"}
                  </Button>
                </div>
              );
            })}
          </div>
        </CardContent>
      </Card>
    </div>,
    document.body
  );
}
