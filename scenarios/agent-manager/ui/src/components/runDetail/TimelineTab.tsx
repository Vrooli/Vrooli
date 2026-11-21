import type { Run, RunEvent } from "../../types";
import { RunTimeline } from "../RunTimeline";
import { FallbackTimeline } from "../runs/FallbackTimeline";

interface TimelineTabProps {
  run: Run;
  events: RunEvent[];
  eventsLoading: boolean;
  focusEventId?: string;
  focusSequence?: string;
  onContinue: (message: string, attachmentIds?: string[]) => Promise<void>;
  onDeleteMessage: (eventId: string) => Promise<void>;
}

export function TimelineTab({ run, events, eventsLoading, focusEventId, focusSequence, onContinue, onDeleteMessage }: TimelineTabProps) {
  return (
    <div data-testid="run-detail-timeline-layout" className="flex h-full min-h-0 flex-col gap-3">
      <div className="shrink-0">
        <FallbackTimeline runId={run.id} />
      </div>
      <div className="min-h-0 flex-1">
        <RunTimeline
          run={run}
          events={events}
          eventsLoading={eventsLoading}
          focusEventId={focusEventId}
          focusSequence={focusSequence}
          onContinue={onContinue}
          onDeleteMessage={onDeleteMessage}
        />
      </div>
    </div>
  );
}
