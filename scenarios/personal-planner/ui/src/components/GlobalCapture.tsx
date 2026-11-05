import { useCallback, useState } from "react";
import { Command, NotebookPen, Sparkles } from "lucide-react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { IconButton } from "@vrooli/react-component-library/IconButton/3.1.6";
import { Input } from "@vrooli/react-component-library/Input/1";
import { createWorkItem } from "../api/work";
import { applyAllocationProposal, previewAllocation } from "../api/calendar";
import { useKeyboardShortcut } from "../hooks/useKeyboardShortcut";
import { PlannerDialog } from "./PlannerDialog";

export function parseNaturalCapture(value: string) {
  const raw = value.trim();
  const duration = raw.match(/\b(\d+)\s*(m|min|mins|h|hr|hrs|hour|hours)\b/i);
  const minutes = duration ? Number(duration[1]) * (/h|hr|hrs|hour|hours/i.test(duration[2] ?? "") ? 60 : 1) : 0;
  const time = raw.match(/\b([01]?\d|2[0-3])(?::([0-5]\d))?\s*(am|pm)?\b/i);
  const source = raw.match(/\b(mon|tue|wed|thu|fri|sat|sun)(?:day)?\b/i)?.[0] ?? "quick capture";
  const title = raw
    .replace(duration?.[0] ?? "", "")
    .replace(time?.[0] ?? "", "")
    .replace(/\b(mon|tue|wed|thu|fri|sat|sun)(?:day)?\b/i, "")
    .replace(/\s+/g, " ")
    .trim();
  return { title: title || raw, minutes, source, time: time?.[0] ?? "" };
}

export function localDateForSource(source: string): string {
  const now = new Date();
  const weekdays = ["sun", "mon", "tue", "wed", "thu", "fri", "sat"];
  const target = weekdays.indexOf(source.slice(0, 3).toLowerCase());
  if (target >= 0) {
    const daysAhead = (target - now.getDay() + 7) % 7;
    now.setDate(now.getDate() + daysAhead);
  }
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-${String(now.getDate()).padStart(2, "0")}`;
}

export function startMinutesForTime(value: string): number {
  const match = value.trim().match(/^(\d{1,2})(?::(\d{2}))?\s*(am|pm)?$/i);
  if (!match) return 9 * 60;
  let hour = Number(match[1]);
  const minute = Number(match[2] ?? 0);
  const meridiem = match[3]?.toLowerCase();
  if (meridiem === "pm" && hour < 12) hour += 12;
  if (meridiem === "am" && hour === 12) hour = 0;
  return hour * 60 + minute;
}

export function GlobalCapture() {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [value, setValue] = useState("");
  const mutation = useMutation({
    mutationFn: async () => {
      const parsed = parseNaturalCapture(value);
      const item = await createWorkItem({ title: parsed.title, remainingMinutes: parsed.minutes, sourceLabel: parsed.time ? `${parsed.source} · ${parsed.time}` : parsed.source });
      if (item.id && parsed.minutes > 0 && parsed.time) {
        const proposal = await previewAllocation({ workItemId: item.id, localDate: localDateForSource(parsed.source), startMinutes: startMinutesForTime(parsed.time), durationMinutes: parsed.minutes });
        if (proposal.state !== "feasible") throw new Error(proposal.reason || "That placement is not feasible");
        await applyAllocationProposal({ proposalId: proposal.id, expectedRevision: proposal.baseRevision, idempotencyKey: `${proposal.id}:capture` });
      }
      return item;
    },
    onSuccess: async () => { setValue(""); setOpen(false); await queryClient.invalidateQueries({ queryKey: ["work-items"] }); await queryClient.invalidateQueries({ queryKey: ["today-allocations"] }); await queryClient.invalidateQueries({ queryKey: ["calendar-range"] }); },
  });
  const openCapture = useCallback(() => setOpen(true), []);
  useKeyboardShortcut("k", openCapture);
  const parsed = value ? parseNaturalCapture(value) : null;
  return <>
    <Button className="global-capture-trigger" type="button" variant="ghost" onClick={openCapture} icon={<Command size={15} aria-hidden="true" />}>Capture <kbd>⌘K</kbd></Button>
    <IconButton className="global-capture-fab" type="button" surface="solid" shape="circle" size="lg" aria-label="Capture task" onClick={openCapture}><NotebookPen size={21} aria-hidden="true" /></IconButton>
    <PlannerDialog open={open} title="Capture a useful next step" description="Write it as you would say it. For example: lunch with Sam tue 1pm 1h" onClose={() => { setOpen(false); mutation.reset(); }} closeLabel="Close capture" contentClassName="global-capture-dialog">
      <form onSubmit={(event) => { event.preventDefault(); if (value.trim()) mutation.mutate(); }}>
        <label className="global-capture-input"><span>What should become true?</span><Input autoFocus value={value} onChange={(event) => setValue(event.target.value)} placeholder="lunch with Sam tue 1pm 1h" /></label>
        {parsed && <p className="global-capture-preview"><Sparkles size={15} aria-hidden="true" /> {parsed.title}{parsed.minutes ? ` · ${parsed.minutes} min` : " · time estimate not set"}{parsed.time ? ` · ${parsed.time}` : ""}</p>}
        {mutation.isError && <p role="alert">That capture did not save. Your words are still here.</p>}
        <Button type="submit" variant="primary" disabled={!value.trim() || mutation.isPending} pending={mutation.isPending} pendingLabel="Capturing…">Capture task</Button>
      </form>
    </PlannerDialog>
  </>;
}
