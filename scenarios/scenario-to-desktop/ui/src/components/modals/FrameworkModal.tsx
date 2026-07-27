import { createPortal } from "react-dom";
import { Cpu, X } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import { Button } from "../ui/button";

type FrameworkId = "electron";

const FRAMEWORKS: Array<{
  id: FrameworkId;
  name: string;
  summary: string;
  strengths: string[];
  tradeoffs: string[];
  bestFor: string[];
}> = [
  {
    id: "electron",
    name: "Electron",
    summary: "Chromium + Node.js, huge ecosystem, most compatibility.",
    strengths: ["Mature tooling", "Full Node access", "Largest community"],
    tradeoffs: ["Heavier bundles", "Higher memory footprint"],
    bestFor: [
      "Feature-rich apps",
      "Deep OS integration",
      "Fastest path to ship",
    ],
  },
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
    tagline: "Most compatible today",
  },
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
  onClose,
}: FrameworkModalProps) {
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
            <CardTitle className="text-lg text-slate-100">
              Choose a framework
            </CardTitle>
            <p className="text-sm text-slate-400">
              Electron is the supported desktop framework.
            </p>
          </div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={onClose}
            className="h-8 w-8 p-0"
            aria-label="Close framework chooser"
          >
            <X className="h-4 w-4" />
          </Button>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="text-sm text-slate-300">
              Current selection:{" "}
              <span className="font-semibold text-slate-100">
                {selectedFramework}
              </span>
            </p>
          </div>

          <div className="grid gap-4">
            {FRAMEWORKS.map((framework) => {
              const isSelected = selectedFramework === framework.id;
              const visual = FRAMEWORK_VISUALS[framework.id];
              return (
                <div
                  key={framework.id}
                  className={`rounded-lg border border-slate-800/80 bg-slate-950/60 p-4 transition-all ${
                    isSelected
                      ? "shadow-[inset_0_0_0_2px_rgba(59,130,246,0.9)]"
                      : ""
                  } hover:scale-[1.01]`}
                >
                  <div
                    className={`mb-3 flex items-center gap-3 rounded-md bg-gradient-to-r ${visual.gradient} p-3`}
                  >
                    <visual.icon className="h-6 w-6 text-white/90" />
                    <div>
                      <p className="text-sm font-semibold text-white">
                        {framework.name}
                      </p>
                      <p className="text-xs text-slate-100/90">
                        {visual.tagline}
                      </p>
                    </div>
                  </div>

                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <p className="text-xs text-slate-400">
                        {framework.summary}
                      </p>
                    </div>
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
                    onClick={() => {
                      onSelect(framework.id);
                    }}
                  >
                    Select
                  </Button>
                </div>
              );
            })}
          </div>
        </CardContent>
      </Card>
    </div>,
    document.body,
  );
}
