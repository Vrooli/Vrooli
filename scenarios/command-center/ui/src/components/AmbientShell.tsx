import { useEffect, useState, type ReactNode } from "react";
import { ExperienceSurface } from "@vrooli/react-component-library/ExperienceSurface/1.0.3";
import { formatClock } from "../lib/format";
import { INK_LABELS, InkSwatch } from "@vrooli/react-component-library/ProvenanceInk/0.1.1";
import { useBoardController, type SamplesMode } from "../lib/boardContext";

interface AmbientShellProps {
  theme: string;
  /** Room title in the eyebrow. */
  title: string;
  /** Position line under the title: "ROOM 2 OF 6" or a surface name. */
  position: string;
  /** Right-hand eyebrow content: source availability dots. */
  status?: ReactNode;
  /** Shown bottom-right whenever illustrative readings are on screen in mark mode. */
  legend?: boolean;
  children: ReactNode;
}

/**
 * Full-bleed, zero idle chrome. The only persistent elements are the cycle
 * rail at the top edge and the eyebrow; controls reveal on input and fade.
 */
export function AmbientShell({ theme, title, position, status, legend = false, children }: AmbientShellProps) {
  const board = useBoardController();
  const [clock, setClock] = useState(() => formatClock(new Date()));
  useEffect(() => {
    const timer = window.setInterval(() => setClock(formatClock(new Date())), 1000);
    return () => window.clearInterval(timer);
  }, []);
  useEffect(() => {
    const root = document.documentElement;
    root.setAttribute("data-theme", theme);
    return () => root.removeAttribute("data-theme");
  }, [theme]);
  return (
    <div className="cc-shell" data-theme={theme} data-testid="ambient-shell">
      <div className="cc-cycle-rail" data-testid="cycle-rail" data-experience-surface="cycle-rail" data-experience-state="static" role="progressbar" aria-label={board.paused ? "Cycle paused" : "Cycle running"} aria-valuenow={Math.round(board.progress * 100)} aria-valuemin={0} aria-valuemax={100} data-paused={board.paused || undefined}>
        <span data-testid="room-cycle-rail" style={{ transform: `scaleX(${board.progress.toFixed(3)})` }} />
      </div>
      <header className="cc-eyebrow">
        <div className="cc-eyebrow-identity">
          <h1 className="cc-room-title">{title}</h1>
          <span className="cc-eyebrow-line">{position}{board.paused ? " · PAUSED" : ` · CYCLE ${board.cycleSeconds}S`}</span>
        </div>
        <div className="cc-eyebrow-status">
          {status}
          <time className="cc-clock" dateTime={new Date().toISOString()}>{clock}</time>
        </div>
      </header>
      {children}
      {legend && board.samples === "mark" ? (
        <ExperienceSurface surfaceId="legend" as="div" data-testid="room-legend" className="cc-legend" state="static" aria-label="Provenance legend">
          {(["solid", "dimmed", "hollow", "dotted"] as const).map((ink) => (
            <span key={ink} className="cc-legend-item"><InkSwatch ink={ink} />{INK_LABELS[ink]}</span>
          ))}
        </ExperienceSurface>
      ) : null}
      {board.samples !== "mark" ? <span className="cc-mode-stamp" data-testid="samples-mode-stamp">samples {board.samples}</span> : null}
      <ControlBar />
      {board.helpVisible ? <HelpOverlay /> : null}
      <div className={board.transitioning ? "cc-veil cc-veil-on" : "cc-veil"} aria-hidden="true" />
    </div>
  );
}

function ControlBar() {
  const board = useBoardController();
  const visible = board.controlsVisible || board.helpVisible;
  const current = board.rooms.findIndex((room) => window.location.pathname.endsWith(`/${room.id}`));
  return (
    <>
      {!visible ? <div className="cc-visually-hidden" data-testid="control-bar-controls" data-experience-surface="controls" data-experience-state="static" aria-hidden="true" /> : null}
      <nav className={visible ? "cc-controls cc-controls-visible" : "cc-controls"} aria-label="Board controls" data-testid={visible ? "control-bar-controls" : undefined} data-experience-surface={visible ? "controls" : undefined} data-experience-state={visible ? "ready" : undefined} aria-hidden={!visible}>
        <button type="button" className="cc-control" onClick={() => board.dispatch("page-prev")} aria-label="Previous room">‹</button>
        <button type="button" className="cc-control" onClick={() => board.dispatch("pause-cycle")} aria-label={board.paused ? "Resume cycle" : "Pause cycle"}>{board.paused ? "▶" : "❚❚"}</button>
        <button type="button" className="cc-control" onClick={() => board.dispatch("page-next")} aria-label="Next room">›</button>
        <span className="cc-room-dots" role="list" aria-label="Rooms">
          {board.rooms.map((room, i) => (
            <button key={room.id} type="button" role="listitem" className={i === current ? "cc-room-dot cc-room-dot-current" : "cc-room-dot"} onClick={() => board.goTo(`/${room.id}`)} aria-label={room.title} aria-current={i === current ? "page" : undefined} />
          ))}
        </span>
        <button type="button" className="cc-control cc-control-text" onClick={() => board.goTo("/focus")}>Focus</button>
        <button type="button" className="cc-control cc-control-text" onClick={() => board.goTo("/open-loop")}>Open loop</button>
        <label className="cc-control cc-control-select">
          <span>Samples</span>
          <select value={board.samples} onChange={(event) => board.setSamples(event.target.value as SamplesMode)} aria-label="Sample visibility">
            <option value="mark">mark</option>
            <option value="full">full</option>
            <option value="hide">hide</option>
          </select>
        </label>
        <button type="button" className="cc-control" onClick={() => board.dispatch("toggle-fullscreen")} aria-label="Toggle fullscreen">⛶</button>
        <button type="button" className="cc-control" onClick={() => board.dispatch("show-help")} aria-label="Help">?</button>
        <output className="cc-ack" data-testid={visible ? "control-bar-acknowledgement" : undefined} data-experience-surface={visible ? "acknowledgement" : undefined} data-experience-state={visible ? "ready" : undefined} aria-live="polite">{board.acknowledgement}</output>
      </nav>
      {!visible ? <output className="cc-visually-hidden" data-testid="control-bar-acknowledgement" data-experience-surface="acknowledgement" data-experience-state="ready" aria-live="polite">{board.acknowledgement}</output> : null}
    </>
  );
}

function HelpOverlay() {
  const board = useBoardController();
  const rows: Array<[string, string, string, string]> = [
    ["Next / previous room", "← → · 1–9", "D-pad ◀ ▶", "swipe"],
    ["Pause / resume cycle", "space", "A", "long-press"],
    ["Fullscreen", "F", "menu (hold)", "control bar"],
    ["Show controls", "any key", "Y", "tap"],
    ["This help", "?", "view", "control bar"],
    ["Hide", "esc", "B", "tap away"],
  ];
  return (
    <div className="cc-help" role="dialog" aria-label="Board help" data-testid="shortcut-help" data-experience-surface="shortcut-help" data-experience-state="ready" onClick={() => board.dispatch("back")}>
      <div className="cc-help-panel" onClick={(event) => event.stopPropagation()}>
        <h2>Command Center</h2>
        <p>The board cycles rooms on its own. Any input pauses it; twenty seconds of quiet resumes it. Nothing here writes anywhere.</p>
        <table>
          <thead><tr><th>Intent</th><th>Keyboard</th><th>Gamepad</th><th>Touch</th></tr></thead>
          <tbody>{rows.map((row) => <tr key={row[0]}>{row.map((cell, i) => <td key={i}>{cell}</td>)}</tr>)}</tbody>
        </table>
        <p className="cc-help-inks">
          {(["solid", "dimmed", "hollow", "dotted"] as const).map((ink) => (
            <span key={ink} className="cc-legend-item"><InkSwatch ink={ink} />{INK_LABELS[ink]}</span>
          ))}
        </p>
        <p className="cc-help-url">?room=forge&amp;cycle=45&amp;samples=mark&amp;fullscreen=1 boots a configured kiosk.</p>
      </div>
    </div>
  );
}
