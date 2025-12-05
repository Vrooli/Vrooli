import { useMemo, useState, type ComponentType } from "react";
import { useQuery } from "@tanstack/react-query";
import { Aperture, LayoutTemplate, PanelsTopLeft, ShieldCheck, Sparkles } from "lucide-react";
import { fetchTemplates, type TemplateInfo } from "../lib/api";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";

interface TemplateGridProps {
  selectedTemplate: string;
  onSelect: (type: string) => void;
}

type HelperAnswers = {
  needsTray: "yes" | "no" | null;
  needsMultiWindow: "yes" | "no" | null;
  kiosk: "yes" | "no" | null;
  lockedHardware: "yes" | "no" | null;
};

const visuals: Record<
  string,
  {
    icon: ComponentType<{ className?: string }>;
    gradient: string;
    tagline: string;
  }
> = {
  basic: {
    icon: Aperture,
    gradient: "from-blue-700/80 via-blue-600/60 to-indigo-600/80",
    tagline: "Balanced single window wrapper"
  },
  advanced: {
    icon: Sparkles,
    gradient: "from-emerald-600/80 via-teal-600/70 to-blue-600/80",
    tagline: "Tray, shortcuts, deep OS touches"
  },
  multi_window: {
    icon: PanelsTopLeft,
    gradient: "from-purple-700/80 via-indigo-700/70 to-blue-700/70",
    tagline: "Multiple coordinated windows"
  },
  kiosk: {
    icon: ShieldCheck,
    gradient: "from-amber-700/80 via-orange-600/80 to-red-600/80",
    tagline: "Locked-down fullscreen kiosk"
  }
};

export function TemplateGrid({ selectedTemplate, onSelect }: TemplateGridProps) {
  const [showHelper, setShowHelper] = useState(false);
  const [answers, setAnswers] = useState<HelperAnswers>({
    needsTray: null,
    needsMultiWindow: null,
    kiosk: null,
    lockedHardware: null
  });
  const { data, isLoading, error } = useQuery({
    queryKey: ["templates"],
    queryFn: fetchTemplates
  });

  const recommended = useMemo(() => {
    if (answers.kiosk === "yes" || answers.lockedHardware === "yes") return "kiosk";
    if (answers.needsMultiWindow === "yes") return "multi_window";
    if (answers.needsTray === "yes") return "advanced";
    if (answers.kiosk === "no" && answers.needsMultiWindow === "no" && answers.needsTray === "no") return "basic";
    return null;
  }, [answers]);

  const recommendedLabel = useMemo(() => {
    if (!recommended) return null;
    return data?.templates.find((t) => t.type === recommended)?.name || "Recommended";
  }, [data?.templates, recommended]);

  if (isLoading) {
    return <div className="text-slate-400">Loading templates...</div>;
  }

  if (error) {
    return <div className="text-red-400">Failed to load templates</div>;
  }

  return (
    <div className="grid gap-4 sm:grid-cols-2">
        <div className="sm:col-span-2 flex items-center justify-between gap-3 rounded-lg border border-slate-800/80 bg-slate-950/60 p-3">
          <div className="flex items-center gap-2 text-sm text-slate-200">
            <LayoutTemplate className="h-4 w-4 text-blue-300" />
            <span>Choose a template, or let us suggest one.</span>
          </div>
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
        <div className="sm:col-span-2 rounded-lg border border-blue-800/70 bg-blue-950/30 p-4 space-y-4">
          <div className="flex items-start justify-between gap-3">
            <div className="space-y-1">
              <p className="text-sm font-semibold text-slate-100">Quick questionnaire</p>
              <p className="text-sm text-slate-200">
                Answer a few questions to get a recommendation. You can override anytime.
              </p>
            </div>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() =>
                setAnswers({
                  needsTray: null,
                  needsMultiWindow: null,
                  kiosk: null,
                  lockedHardware: null
                })
              }
            >
              Reset
            </Button>
          </div>

          <div className="grid gap-3 md:grid-cols-2">
            <Question
              prompt="Do you need a system tray icon or global shortcuts?"
              value={answers.needsTray}
              onChange={(value) => setAnswers((prev) => ({ ...prev, needsTray: value }))}
            />
            <Question
              prompt="Will users open multiple windows or panels at once?"
              value={answers.needsMultiWindow}
              onChange={(value) => setAnswers((prev) => ({ ...prev, needsMultiWindow: value }))}
            />
            <Question
              prompt="Is this a locked-down kiosk or unattended terminal?"
              value={answers.kiosk}
              onChange={(value) => setAnswers((prev) => ({ ...prev, kiosk: value }))}
            />
            <Question
              prompt="Are you targeting controlled hardware (retail, signage, factory)?"
              value={answers.lockedHardware}
              onChange={(value) => setAnswers((prev) => ({ ...prev, lockedHardware: value }))}
            />
          </div>

          <div className="flex flex-wrap items-center gap-3">
            <div className="text-sm text-slate-100">
              {recommended ? (
                <span>
                  Recommended: <span className="font-semibold">{recommendedLabel}</span>
                  {" "}
                  ({recommended === "basic"
                    ? "single window"
                    : recommended === "advanced"
                      ? "tray + shortcuts"
                      : recommended === "multi_window"
                        ? "multi-surface"
                        : "locked-down kiosk"}
                  )
                </span>
              ) : (
                "Answer the questions to see a recommendation."
              )}
            </div>
            <Button
              type="button"
              size="sm"
              disabled={!recommended}
              onClick={() => recommended && onSelect(recommended)}
            >
              Apply recommendation
            </Button>
          </div>
        </div>
      )}

      {data?.templates.map((template) => (
        <TemplateCard
          key={template.type}
          template={template}
          selected={selectedTemplate === template.type}
          onSelect={() => onSelect(template.type)}
        />
      ))}
    </div>
  );
}

interface TemplateCardProps {
  template: TemplateInfo;
  selected: boolean;
  onSelect: () => void;
}

function TemplateCard({ template, selected, onSelect }: TemplateCardProps) {
  const visual = visuals[template.type] || visuals.basic;
  const tagline = visual.tagline || template.use_cases?.[0];

  return (
    <button
      type="button"
      onClick={onSelect}
      className={`rounded-lg border p-4 text-left transition-all hover:scale-[1.02] ${
        selected
          ? "border-blue-500 bg-blue-900/20 shadow-lg shadow-blue-900/40"
          : "border-white/10 bg-white/5 hover:border-white/30"
      }`}
    >
      <div className={`mb-3 flex items-center gap-3 rounded-md bg-gradient-to-r ${visual.gradient} p-3`}>
        <visual.icon className="h-6 w-6 text-white/90" />
        <div>
          <p className="text-sm font-semibold text-white">{template.name}</p>
          <p className="text-xs text-slate-100/90">{tagline}</p>
        </div>
      </div>

      <h3 className="mb-2 text-lg font-semibold text-slate-50">{template.name}</h3>
      <p className="mb-3 text-sm text-slate-400">{template.description}</p>
      {template.use_cases && template.use_cases.length > 0 && (
        <p className="mb-3 text-xs text-slate-300">
          Best for: {template.use_cases.slice(0, 2).join(" · ")}
        </p>
      )}
      <div className="flex flex-wrap gap-1.5">
        {template.features.slice(0, 4).map((feature) => (
          <Badge key={feature} variant="default" className="text-xs">
            {feature}
          </Badge>
        ))}
        {template.features.length > 4 && (
          <Badge variant="info" className="text-xs">
            +{template.features.length - 4} more
          </Badge>
        )}
      </div>
    </button>
  );
}

interface QuestionProps {
  prompt: string;
  value: "yes" | "no" | null;
  onChange: (value: "yes" | "no") => void;
}

function Question({ prompt, value, onChange }: QuestionProps) {
  return (
    <div className="rounded-lg border border-slate-800 bg-slate-950/60 p-3">
      <p className="text-sm text-slate-100">{prompt}</p>
      <div className="mt-2 flex gap-2">
        <Button
          type="button"
          variant={value === "yes" ? "default" : "outline"}
          size="sm"
          onClick={() => onChange("yes")}
        >
          Yes
        </Button>
        <Button
          type="button"
          variant={value === "no" ? "default" : "outline"}
          size="sm"
          onClick={() => onChange("no")}
        >
          No
        </Button>
      </div>
    </div>
  );
}
