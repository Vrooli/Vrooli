import { CalendarDays, FilePlus2, FileText, Moon, Play, Sparkles, Sun, SunMoon } from "lucide-react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { IconButton } from "@vrooli/react-component-library/IconButton/3.1.6";
import { FormField } from "@vrooli/react-component-library/FormField/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { Popover, PopoverParts } from "@vrooli/react-component-library/Popover/1";
import { Textarea } from "@vrooli/react-component-library/Textarea/1";
import type { FormEvent, ReactNode } from "react";

export type TodayAppearanceChoice = "auto" | "day" | "night";
export type TodayTimelineView = "day" | "focus";
export type TodayTimelineItem = { id: string; label: string; detail: string; owner: string; startLabel: string; start: number; width: number; tone: string; end: number; lane: number };

const appearanceOptions: Record<TodayAppearanceChoice, { label: string; icon: ReactNode }> = {
  auto: { label: "Auto", icon: <SunMoon size={16} aria-hidden="true" /> },
  day: { label: "Day", icon: <Sun size={16} aria-hidden="true" /> },
  night: { label: "Night", icon: <Moon size={16} aria-hidden="true" /> },
};
const nextAppearance: Record<TodayAppearanceChoice, TodayAppearanceChoice> = { auto: "day", day: "night", night: "auto" };

export function TodayAppearanceControl({ choice, onChange }: { choice: TodayAppearanceChoice; onChange: (choice: TodayAppearanceChoice) => void }) {
  const current = appearanceOptions[choice];
  const next = nextAppearance[choice];
  return <IconButton type="button" className="appearance-toggle" surface="soft" size="lg" morph="auto" swapIdentity="today-appearance" iconKey={choice} aria-label={`Appearance: ${current.label}. Switch to ${appearanceOptions[next].label}`} onClick={() => onChange(next)}>{current.icon}</IconButton>;
}

export function TodayCaptureLink({ onClick }: { onClick: () => void }) {
  return <Button type="button" variant="ghost" className="capture-link" icon={<FilePlus2 size={16} aria-hidden="true" />} onClick={onClick}>Capture task</Button>;
}

export function TodayCaptureForm({ title, description, minutes, source, pending, error, onTitleChange, onDescriptionChange, onMinutesChange, onSourceChange, onSubmit }: {
  title: string;
  description: string;
  minutes: string;
  source: string;
  pending: boolean;
  error: boolean;
  onTitleChange: (value: string) => void;
  onDescriptionChange: (value: string) => void;
  onMinutesChange: (value: string) => void;
  onSourceChange: (value: string) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  return <form className="capture-panel" onSubmit={onSubmit}>
    <FormField label="Task title" required control={<Input autoFocus aria-label="Task title" value={title} onChange={(event) => onTitleChange(event.target.value)} required />} />
    <FormField label="Why it matters" optionalLabel="(optional)" control={<Textarea aria-label="Why it matters (optional)" value={description} onChange={(event) => onDescriptionChange(event.target.value)} rows={2} />} />
    <div className="capture-fields">
      <FormField label="Minutes" optionalLabel="(optional)" control={<Input aria-label="Minutes (optional)" type="number" min="0" max="1440" step="5" value={minutes} onChange={(event) => onMinutesChange(event.target.value)} />} />
      <FormField label="Source" optionalLabel="(optional)" control={<Input aria-label="Source (optional)" value={source} onChange={(event) => onSourceChange(event.target.value)} />} />
    </div>
    {error && <p role="alert">That task did not save. Nothing was assumed.</p>}
    <Button type="submit" className="primary-action" disabled={pending || !title.trim()} pending={pending} pendingLabel="Saving…">Save task</Button>
  </form>;
}

export function TodayTaskActions({ focusStarted, pending, hasWork, onFocus, onOpenDraft, onCapture, onComplete }: { focusStarted: boolean; pending: boolean; hasWork: boolean; onFocus: () => void; onOpenDraft: () => void; onCapture: () => void; onComplete?: () => void }) {
  return <div className="task-actions"><Button type="button" className="primary-action" icon={<Play size={18} fill="currentColor" />} onClick={onFocus} disabled={pending} pending={pending} pendingLabel="Saving…">{focusStarted ? "Pause focus" : "Start focus"}</Button>{hasWork ? <><Button type="button" className="quiet-action" variant="secondary" icon={<FileText size={18} aria-hidden="true" />} onClick={onOpenDraft}>Open draft</Button>{onComplete && <Button type="button" className="quiet-action" variant="ghost" onClick={onComplete} disabled={pending}>Mark complete</Button>}</> : <Button type="button" className="quiet-action" variant="secondary" icon={<FilePlus2 size={18} aria-hidden="true" />} onClick={onCapture}>Capture task</Button>}</div>;
}

function TodayCloseButton({ label, onClick }: { label: string; onClick: () => void }) {
  return <Button type="button" variant="ghost" size="icon" shape="pill" className="today-close-button" onClick={onClick} aria-label={label}>×</Button>;
}

export function TodayTimelineViewToggle({ view, onChange }: { view: TodayTimelineView; onChange: (view: TodayTimelineView) => void }) {
  return <div className="timeline-view-toggle" role="group" aria-label="Timeline view"><Button type="button" variant="ghost" className={view === "day" ? "selected" : ""} aria-pressed={view === "day"} icon={<CalendarDays size={14} aria-hidden="true" />} onClick={() => onChange("day")}>Day</Button><Button type="button" variant="ghost" className={view === "focus" ? "selected" : ""} aria-pressed={view === "focus"} icon={<Sparkles size={14} aria-hidden="true" />} onClick={() => onChange("focus")}>Focus</Button></div>;
}

export function TodayTimelineBlock({ item, startHour, span, open, onOpen, onClose }: { item: TodayTimelineItem; startHour: number; span: number; open: boolean; onOpen: () => void; onClose: () => void }) {
  const compact = item.width < 1;
  return <Popover open={open} onOpenChange={(next) => next ? onOpen() : onClose()} placement="bottom-start" responsive="auto">
    <PopoverParts.Trigger asChild aria-label={`${item.label}, ${item.startLabel}, ${item.detail}${item.owner ? `, ${item.owner}` : ""}`}>
      <div role="button" tabIndex={0} data-full-title={item.label} title={item.label} className={`time-block ${item.tone}${compact ? " compact" : ""}`} style={{ left: `${((item.start - startHour) / span) * 100}%`, width: `${(item.width / span) * 100}%`, top: `calc(2rem + ${item.lane} * 6.2rem)` }} onKeyDown={(event) => { if (event.key === "Enter" || event.key === " ") { event.preventDefault(); onOpen(); } }}>
        <span className="time-block-time">{item.startLabel}</span><strong>{compact ? item.detail.replace(/\s+min$/, "m") : item.label}</strong><span>{item.detail}</span>{item.owner && <small><span className="source-dot" />{item.owner}</small>}
      </div>
    </PopoverParts.Trigger>
    <PopoverParts.Content className="timeline-detail" aria-label="Timeline item details" initialFocus="first">
      <div><strong>{item.label}</strong><TodayCloseButton label="Close timeline item" onClick={onClose} /></div>
      <p>{item.startLabel} · {item.detail}</p>{item.owner && <small>{item.owner}</small>}
    </PopoverParts.Content>
  </Popover>;
}
