import { useEffect } from "react";
import { Camera } from "lucide-react";
import { formatBytes } from "../../domain/download";
import { Button } from "../ui/button";
import { SectionTitle, ActionRow } from "../ui/section-helpers";
import { useCapturesStore } from "../../store/capturesStore";

interface CapturesSectionProps {
  scenarioName: string;
}

export function CapturesSection({ scenarioName }: CapturesSectionProps) {
  const summary = useCapturesStore((s) => s.summary);
  const fetchSummary = useCapturesStore((s) => s.fetchSummary);

  useEffect(() => {
    void fetchSummary(scenarioName);
  }, [scenarioName, fetchSummary]);

  const count = summary?.count ?? 0;
  const totalBytes = Number(summary?.totalBytes ?? 0n);

  return (
    <section className="space-y-2">
      <SectionTitle icon={Camera}>Captures</SectionTitle>
      <ActionRow
        icon={Camera}
        title={
          count > 0
            ? `${String(count)} capture${count !== 1 ? "s" : ""}`
            : "No captures yet"
        }
        subtitle={
          count > 0
            ? formatBytes(totalBytes)
            : "Screenshots and recordings from desktop sessions"
        }
      >
        {count > 0 && (
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => {
              useCapturesStore.getState().open(scenarioName);
            }}
          >
            View All
          </Button>
        )}
      </ActionRow>
    </section>
  );
}
