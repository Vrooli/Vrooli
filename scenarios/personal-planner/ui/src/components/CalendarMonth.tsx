import { useMemo, useRef, useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { FormField } from "@vrooli/react-component-library/FormField/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { Select } from "@vrooli/react-component-library/Select/1";
import { Textarea } from "@vrooli/react-component-library/Textarea/1";
import { createCalendarEvent, fetchCalendarEvent, fetchCalendarEvents, lookupCalendarEventByIdempotencyKey, updateCalendarEvent, type CalendarEvent, type CalendarEventInput } from "../api/calendar";

type Draft = Omit<CalendarEventInput, "created_at" | "updated_at">;
const EMPTY_DRAFT: Draft = { title: "", subject: "", notes: "", availability: "busy", timezone: "America/New_York", all_day: true, start_date: "", end_date_exclusive: "", start_at: "", end_at: "" };

function dateString(date: Date) { return `${date.getUTCFullYear()}-${String(date.getUTCMonth() + 1).padStart(2, "0")}-${String(date.getUTCDate()).padStart(2, "0")}`; }
function shiftDate(value: string, days: number) {
  const [year = 0, month = 1, day = 1] = value.split("-").map(Number);
  return dateString(new Date(Date.UTC(year, month - 1, day + days, 12)));
}
function addMonths(value: string, amount: number) {
  const [year = 0, month = 1] = value.split("-").map(Number);
  return `${new Date(Date.UTC(year, month - 1 + amount, 1)).getUTCFullYear()}-${String(new Date(Date.UTC(year, month - 1 + amount, 1)).getUTCMonth() + 1).padStart(2, "0")}`;
}
function dateFromMonth(value: string, day = 1) {
  const [year = 0, month = 1] = value.split("-").map(Number);
  return dateString(new Date(Date.UTC(year, month - 1, day, 12)));
}
function eventTouchesDate(event: CalendarEvent, date: string) {
  if (event.all_day) return event.start_date <= date && date < event.end_date_exclusive;
  const start = new Date(event.start_at);
  const end = new Date(event.end_at);
  end.setMilliseconds(end.getMilliseconds() - 1);
  const localDate = (value: Date) => {
    const parts = new Intl.DateTimeFormat("en-CA", { timeZone: event.timezone, year: "numeric", month: "2-digit", day: "2-digit" }).formatToParts(value);
    const part = (type: string) => parts.find((entry) => entry.type === type)?.value ?? "";
    return `${part("year")}-${part("month")}-${part("day")}`;
  };
  return localDate(start) <= date && date <= localDate(end);
}
function dateLabel(value: string) {
  const [year = 0, month = 1, day = 1] = value.split("-").map(Number);
  return new Intl.DateTimeFormat(undefined, { weekday: "long", month: "long", day: "numeric" }).format(new Date(Date.UTC(year, month - 1, day, 12)));
}
function monthLabel(value: string) {
  const [year = 0, month = 1] = value.split("-").map(Number);
  return new Intl.DateTimeFormat(undefined, { month: "long", year: "numeric", timeZone: "UTC" }).format(new Date(Date.UTC(year, month - 1, 1)));
}
function makeIdempotencyKey() {
  return typeof crypto !== "undefined" && "randomUUID" in crypto ? crypto.randomUUID() : `calendar-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

export function CalendarMonth() {
  const queryClient = useQueryClient();
  const [month, setMonth] = useState(() => {
    const now = new Date();
    return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}`;
  });
  const rangeStart = dateFromMonth(month);
  const [year = 0, monthNumber = 1] = month.split("-").map(Number);
  const rangeEnd = dateFromMonth(month, new Date(Date.UTC(year, monthNumber, 0)).getUTCDate());
  const eventsQuery = useQuery({ queryKey: ["calendar-events", rangeStart, rangeEnd], queryFn: () => fetchCalendarEvents(rangeStart, rangeEnd) });
  const [draft, setDraft] = useState<Draft>(EMPTY_DRAFT);
  const [selectedDate, setSelectedDate] = useState("");
  const [editingID, setEditingID] = useState("");
  const pendingCreateKeyRef = useRef("");
  const [uncertainCreate, setUncertainCreate] = useState(false);
  const [checkingStatus, setCheckingStatus] = useState(false);
  const [message, setMessage] = useState("");
  const editingQuery = useQuery({ queryKey: ["calendar-event", editingID], queryFn: () => fetchCalendarEvent(editingID), enabled: Boolean(editingID) });
  const editingEvent = editingQuery.data;

  const days = useMemo(() => {
    const firstWeekday = new Date(Date.UTC(year, monthNumber - 1, 1, 12)).getUTCDay();
    const mondayOffset = (firstWeekday + 6) % 7;
    const count = new Date(Date.UTC(year, monthNumber, 0)).getUTCDate();
    const first = shiftDate(rangeStart, -mondayOffset);
    const cells = Math.ceil((mondayOffset + count) / 7) * 7;
    return Array.from({ length: cells }, (_, index) => {
      const date = shiftDate(first, index);
      return { date, inMonth: date.slice(0, 7) === month };
    });
  }, [month, monthNumber, rangeStart, year]);

  const createMutation = useMutation({
    mutationFn: () => {
      if (!pendingCreateKeyRef.current) pendingCreateKeyRef.current = makeIdempotencyKey();
      return createCalendarEvent(draft, pendingCreateKeyRef.current);
    },
    onSuccess: async (event) => {
      pendingCreateKeyRef.current = "";
      setUncertainCreate(false);
      setMessage(`Saved ${event.title} as event ${event.id}, revision ${event.revision}.`);
      setEditingID("");
      setSelectedDate("");
      setDraft(EMPTY_DRAFT);
      await queryClient.invalidateQueries({ queryKey: ["calendar-events"] });
    },
    onError: () => {
      setUncertainCreate(true);
      setMessage("The save outcome could not be confirmed. Check its idempotency key before retrying.");
    },
  });
  const updateMutation = useMutation({
    mutationFn: ({ event, value }: { event: CalendarEvent; value: Draft }) => updateCalendarEvent(event.id, value, event.revision),
    onSuccess: async (event) => {
      setMessage(`Updated ${event.title} to revision ${event.revision}.`);
      setEditingID("");
      setDraft(EMPTY_DRAFT);
      await queryClient.invalidateQueries({ queryKey: ["calendar-events"] });
      await queryClient.invalidateQueries({ queryKey: ["calendar-event", event.id] });
    },
    onError: () => setMessage("This event changed elsewhere or the save could not be confirmed. Your draft is still here; reopen the event to load its current revision before retrying."),
  });

  function startNewEvent(date: string) {
    setSelectedDate(date);
    setEditingID("");
    setDraft({ ...EMPTY_DRAFT, start_date: date, end_date_exclusive: shiftDate(date, 1) });
    pendingCreateKeyRef.current = "";
    setUncertainCreate(false);
    setMessage("");
  }
  function openEvent(event: CalendarEvent) {
    setSelectedDate("");
    setEditingID(event.id);
    setDraft({ title: event.title, subject: event.subject, notes: event.notes, availability: event.availability, timezone: event.timezone, all_day: event.all_day, start_date: event.start_date, end_date_exclusive: event.end_date_exclusive, start_at: event.start_at, end_at: event.end_at });
    pendingCreateKeyRef.current = "";
    setUncertainCreate(false);
    setMessage("");
  }
  function changeDraft<K extends keyof Draft>(key: K, value: Draft[K]) {
    pendingCreateKeyRef.current = "";
    if (key === "all_day") {
      const allDay = value as boolean;
      setDraft((current) => ({ ...current, all_day: allDay, start_date: allDay ? current.start_date : "", end_date_exclusive: allDay ? current.end_date_exclusive : "", start_at: allDay ? "" : current.start_at, end_at: allDay ? "" : current.end_at }));
      return;
    }
    setDraft((current) => ({ ...current, [key]: value }));
  }
  function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMessage("");
    if (editingID && editingEvent) updateMutation.mutate({ event: editingEvent, value: draft });
    else {
      createMutation.mutate();
    }
  }

  async function checkCreateStatus() {
    if (!pendingCreateKeyRef.current) return;
    setCheckingStatus(true);
    try {
      const saved = await lookupCalendarEventByIdempotencyKey(pendingCreateKeyRef.current);
      if (saved) {
        setMessage(`The earlier save succeeded: ${saved.title}, event ${saved.id}, revision ${saved.revision}.`);
        setEditingID("");
        setSelectedDate("");
        setDraft(EMPTY_DRAFT);
        pendingCreateKeyRef.current = "";
        setUncertainCreate(false);
        await queryClient.invalidateQueries({ queryKey: ["calendar-events"] });
      } else {
        setMessage("No event exists for this key yet. Retry the unchanged event; the same key will prevent a duplicate.");
      }
    } catch {
      setMessage("Event status is still unavailable. Keep this draft unchanged and check again before retrying.");
    } finally {
      setCheckingStatus(false);
    }
  }

  const events = eventsQuery.data ?? [];
  const active = editingID ? editingEvent : null;
  const pending = createMutation.isPending || updateMutation.isPending;
  return <section className="calendar-month" aria-label="Native calendar month">
    <header className="calendar-month-header">
      <div><span className="card-kicker">PERSONAL CALENDAR</span><h2>{monthLabel(month)}</h2></div>
      <nav aria-label="Calendar month navigation">
        <Button type="button" variant="secondary" onClick={() => setMonth((value) => addMonths(value, -1))} aria-label="Previous month" data-testid="calendar-month-previous">‹</Button>
        <Button type="button" variant="secondary" onClick={() => setMonth(addMonths(`${new Date().getFullYear()}-${String(new Date().getMonth() + 1).padStart(2, "0")}`, 0))}>Today</Button>
        <Button type="button" variant="secondary" onClick={() => setMonth((value) => addMonths(value, 1))} aria-label="Next month" data-testid="calendar-month-next">›</Button>
      </nav>
    </header>
    {eventsQuery.isLoading && <p role="status">Loading events for {monthLabel(month)}…</p>}
    {eventsQuery.isError && <p role="alert">Events for this month are unavailable. The month grid remains available while you retry.</p>}
    <div className="calendar-month-weekdays" aria-hidden="true">{["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"].map((day) => <span key={day}>{day}</span>)}</div>
    <div className="calendar-month-grid" data-testid="calendar-month-grid" aria-label={`${monthLabel(month)} dates`}>
      {days.map(({ date, inMonth }) => {
        const dayEvents = events.filter((event) => eventTouchesDate(event, date));
        return <section key={date} className={`calendar-day${inMonth ? "" : " outside-month"}${dayEvents.length ? " has-events" : ""}`} aria-label={dateLabel(date)}>
          <header><time dateTime={date}>{Number(date.slice(-2))}</time><Button type="button" variant="ghost" size="icon" shape="pill" onClick={() => startNewEvent(date)} aria-label={`Add event on ${dateLabel(date)}`} data-testid={date === rangeStart ? "calendar-add-first-in-month" : undefined}>+</Button></header>
          {dayEvents.map((event) => <button type="button" className={`calendar-event ${event.availability}`} data-testid={event.all_day ? "calendar-event" : "calendar-timed-event"} data-event-title={event.title} key={event.id} onClick={() => openEvent(event)} aria-label={`Edit ${event.title}, event ${event.id}, ${event.subject || "no subject"}, revision ${event.revision}`}>
            <strong>{event.title}</strong><small>{event.subject || event.availability}{event.all_day ? " · all day" : ` · ${event.timezone}`}</small>{event.provider_event_id && <small>Source {event.provider_event_id}</small>}
          </button>)}
          {!dayEvents.length && <small className="calendar-day-clear">Clear</small>}
        </section>;
      })}
    </div>
    {events.length === 0 && !eventsQuery.isLoading && !eventsQuery.isError && <p className="calendar-empty-note" data-testid="calendar-empty-state">No events this month. The date grid stays open so you can add one.</p>}
    {(selectedDate || editingID) && <form className="calendar-event-editor" data-testid="calendar-event-editor" onSubmit={save} aria-label={editingID ? "Edit calendar event" : "Create calendar event"}>
      <header><div><span className="card-kicker" data-testid="calendar-event-revision">{editingID ? `EVENT REVISION ${active?.revision ?? "…"}` : "NEW EVENT"}</span><h3>{editingID ? "Edit calendar event" : `Add event · ${selectedDate}`}</h3></div><Button type="button" variant="ghost" data-testid="calendar-event-close" disabled={pending || (uncertainCreate && !editingID)} onClick={() => { setSelectedDate(""); setEditingID(""); setMessage(""); }}>Close</Button></header>
      {editingID && editingQuery.isLoading && <p role="status">Reopening event…</p>}
      {editingQuery.isError && <p role="alert">Could not reopen this event. Try opening it again from the month grid.</p>}
      <FormField label="Title" control={<Input aria-label="Event title" data-testid="calendar-event-title" required disabled={uncertainCreate} value={draft.title} onChange={(event) => changeDraft("title", event.target.value)} />} />
      <FormField label="Context subject" control={<Input aria-label="Context subject" data-testid="calendar-event-subject" disabled={uncertainCreate} value={draft.subject} onChange={(event) => changeDraft("subject", event.target.value)} />} /><small>For example, owner, girlfriend awareness, or family awareness.</small>
      <FormField label="Availability" control={<Select aria-label="Event availability" disabled={uncertainCreate} value={draft.availability} onChange={(event) => changeDraft("availability", event.target.value as Draft["availability"])} options={[{ value: "busy", label: "Busy — reserves attention" }, { value: "free", label: "Free — awareness only" }]} />} />
      <FormField label="Timezone" control={<Input aria-label="Event timezone" required disabled={uncertainCreate} value={draft.timezone} onChange={(event) => changeDraft("timezone", event.target.value)} />} />
      <label className="calendar-all-day"><input type="checkbox" data-testid="calendar-event-all-day" disabled={uncertainCreate} checked={draft.all_day} onChange={(event) => changeDraft("all_day", event.target.checked)} /> All day, using civil dates</label>
      {draft.all_day ? <div className="calendar-event-dates"><FormField label="Start date" control={<Input aria-label="Event start date" required disabled={uncertainCreate} value={draft.start_date} onChange={(event) => changeDraft("start_date", event.target.value)} />} /><div><FormField label="End date, exclusive" control={<Input aria-label="Exclusive event end date" data-testid="calendar-event-end-exclusive" type="date" required disabled={uncertainCreate} value={draft.end_date_exclusive} onChange={(event) => changeDraft("end_date_exclusive", event.target.value)} />} /><small>The event includes dates before this day.</small></div></div> : <div className="calendar-event-dates"><div><FormField label="Start instant" control={<Input aria-label="Event start instant" data-testid="calendar-event-start-instant" required disabled={uncertainCreate} placeholder="2026-11-01T01:30:00-04:00" value={draft.start_at} onChange={(event) => changeDraft("start_at", event.target.value)} />} /><small>Use RFC3339 with an explicit offset to distinguish repeated DST times.</small></div><FormField label="End instant" control={<Input aria-label="Event end instant" data-testid="calendar-event-end-instant" required disabled={uncertainCreate} placeholder="2026-11-01T01:30:00-05:00" value={draft.end_at} onChange={(event) => changeDraft("end_at", event.target.value)} />} /></div>}
      <FormField label="Private notes" control={<Textarea aria-label="Event notes" rows={2} disabled={uncertainCreate} value={draft.notes} onChange={(event) => changeDraft("notes", event.target.value)} />} />
      {message && <p role="status" className="calendar-event-message" data-testid="calendar-event-message">{message}</p>}
      <footer>{uncertainCreate && !editingID && <Button type="button" data-testid="calendar-event-check-status" variant="secondary" pending={checkingStatus} pendingLabel="Checking…" onClick={() => void checkCreateStatus()}>Check save status</Button>}<Button type="submit" data-testid={editingID ? "calendar-event-save" : "calendar-event-create"} variant="primary" pending={pending} pendingLabel="Saving…" disabled={pending || checkingStatus || (Boolean(editingID) && !active)}>{editingID ? "Save event" : uncertainCreate ? "Retry same event" : "Create event"}</Button></footer>
    </form>}
    {!selectedDate && !editingID && message && <p role="status" className="calendar-event-message" data-testid="calendar-event-message">{message}</p>}
  </section>;
}
