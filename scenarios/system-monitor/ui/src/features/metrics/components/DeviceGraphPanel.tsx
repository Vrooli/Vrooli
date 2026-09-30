import { timestampDate } from '@bufbuild/protobuf/wkt';
import type { DeviceGraph, GraphDevice, GraphSubsystem, RungState } from '../../../types';

interface DeviceGraphPanelProps {
  graph: DeviceGraph | null;
  error?: string;
}

const RUNG_LABELS = ['identity', 'telemetry', 'evidence', 'control', 'anticipation'];
const GRADE_LABELS: Record<number, string> = {
  0: 'unspecified',
  1: 'measured',
  2: 'unmeasurable',
  3: 'unavailable',
  4: 'not applicable',
};

function rungLabel(rung: RungState): string {
  return RUNG_LABELS[rung.rung] ?? 'unknown rung';
}

function RungList({ rungs }: { rungs: RungState[] }) {
  return (
    <ul className="device-graph-panel__rungs">
      {rungs.map((rung, index) => (
        <li key={`${rung.rung}-${index}`}>
          <span>{rungLabel(rung)}</span>
          <strong>{GRADE_LABELS[rung.grade] ?? 'unknown'}</strong>
          {rung.reason ? <small>{rung.reason}</small> : null}
        </li>
      ))}
    </ul>
  );
}

function DeviceEntry({ device }: { device: GraphDevice }) {
  const readings = Object.entries(device.readings);
  return (
    <article className="device-graph-panel__device">
      <div>
        <strong>{device.model || device.vendor || device.id || 'Unnamed device'}</strong>
        <span className="text-dim-xs">{device.class || 'device'} · {device.id}</span>
      </div>
      {readings.length > 0 ? (
        <dl className="device-graph-panel__readings">
          {readings.map(([name, value]) => <div key={name}><dt>{name}</dt><dd>{value}</dd></div>)}
        </dl>
      ) : null}
      <RungList rungs={device.rungs} />
    </article>
  );
}

function SubsystemEntry({ subsystem }: { subsystem: GraphSubsystem }) {
  return (
    <article className="device-graph-panel__device">
      <strong>{subsystem.name}</strong>
      <span className="text-dim-xs">graded subsystem</span>
      <RungList rungs={subsystem.rungs} />
    </article>
  );
}

export const DeviceGraphPanel = ({ graph, error }: DeviceGraphPanelProps) => {
  const collectedAt = graph?.collectedAt ? timestampDate(graph.collectedAt).toLocaleTimeString() : undefined;
  return (
    <section className="card device-graph-panel" aria-label="Device graph">
      <div className="device-graph-panel__header">
        <div>
          <span className="eyebrow">Hardware substrate</span>
          <h2>Device graph</h2>
        </div>
        {graph ? <span className="text-dim-xs">{graph.platform || 'unknown platform'} · collected {collectedAt ?? 'unknown'}</span> : null}
      </div>
      {error && !graph ? <p role="status">Device graph unavailable: {error}</p> : null}
      {graph && !graph.available ? <p role="status">Device graph unavailable: {graph.unavailableReason || 'the host did not provide a graph'}.</p> : null}
      {graph?.available ? (
        <div className="device-graph-panel__list">
          {graph.devices.map(device => <DeviceEntry key={device.id} device={device} />)}
          {graph.subsystems.map(subsystem => <SubsystemEntry key={subsystem.name} subsystem={subsystem} />)}
          {graph.devices.length === 0 && graph.subsystems.length === 0 ? <p>No graded devices were observed.</p> : null}
        </div>
      ) : null}
    </section>
  );
};
