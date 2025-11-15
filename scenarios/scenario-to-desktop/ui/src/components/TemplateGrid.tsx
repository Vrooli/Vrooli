import { useQuery } from "@tanstack/react-query";
import { fetchTemplates, type TemplateInfo } from "../lib/api";
import { Badge } from "./ui/badge";

interface TemplateGridProps {
  selectedTemplate: string;
  onSelect: (type: string) => void;
}

export function TemplateGrid({ selectedTemplate, onSelect }: TemplateGridProps) {
  const { data, isLoading, error } = useQuery({
    queryKey: ["templates"],
    queryFn: fetchTemplates
  });

  if (isLoading) {
    return <div className="text-slate-400">Loading templates...</div>;
  }

  if (error) {
    return <div className="text-red-400">Failed to load templates</div>;
  }

  return (
    <div className="grid gap-4 sm:grid-cols-2">
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
  return (
    <button
      type="button"
      onClick={onSelect}
      className={`rounded-lg border p-4 text-left transition-all hover:scale-[1.02] ${
        selected
          ? "border-blue-500 bg-blue-900/20"
          : "border-white/10 bg-white/5 hover:border-white/30"
      }`}
    >
      <h3 className="mb-2 text-lg font-semibold text-slate-50">{template.name}</h3>
      <p className="mb-3 text-sm text-slate-400">{template.description}</p>
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
