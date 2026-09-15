import { useId } from 'react';
import type { DownloadAsset } from '../../../shared/api/types';
import { getPlatformLabel } from '../services/downloads.service';
import { downloadSystemUi as ui } from './systemUi';
import { ActionLink } from './primitives';
import type { Action, ResolvedActions } from './types';

export type DownloadState = { status: 'idle' | 'preparing' } | { status: 'error'; message: string } | { status: 'ready'; href: string };
export interface DownloadChooserProps {
  title: string; description: string; options: DownloadAsset[]; selected: string;
  onSelect: (value: string) => void; onPrepare?: () => void; state: DownloadState;
  unavailableReason: string; disabledReason?: string;
  launchActions?: Action[]; resolvedActions?: ResolvedActions;
}

/** Controlled native view. No auth, transport, navigation or analytics effects. */
export function DownloadChooser({ title, description, options, selected, onSelect, onPrepare, state, unavailableReason, disabledReason, launchActions = [], resolvedActions }: DownloadChooserProps) {
  const id = useId();
  const asset = selected === '' ? undefined : options[Number(selected)];
  const platforms = [...new Set(options.map(option => option.platform))];
  return <section className="download-chooser" aria-labelledby={`${id}-title`}>
    <div className="section-intro"><p className="eyebrow">{ui.title}</p><h1 id={`${id}-title`}>{title}</h1><p>{description}</p></div>
    {launchActions.length > 0 && <div className="commerce-actions">{launchActions.map((action, index) => <ActionLink key={index} action={action} resolvedActions={resolvedActions} reason={unavailableReason} />)}</div>}
    {options.length === 0 ? <p className="commerce-note" role="status">{ui.noInstallers}</p> : <div className="download-panel">
      <div><label htmlFor={`${id}-choice`}>{ui.choose}</label>
        <select id={`${id}-choice`} value={selected} onChange={event => { onSelect(event.target.value); }}>
          <option value="">{ui.choose}</option>
          {platforms.map(platform => <optgroup key={platform} label={getPlatformLabel(platform)}>{options.map((option, index) => option.platform === platform && <option key={index} value={String(index)}>{getPlatformLabel(platform)} · {option.release_version}{option.artifact_filename ? ` · ${option.artifact_filename}` : ''}</option>)}</optgroup>)}
        </select>
        {asset && <dl className="download-facts"><div><dt>{ui.platform}</dt><dd>{getPlatformLabel(asset.platform)}</dd></div><div><dt>{ui.release}</dt><dd>{asset.release_version}</dd></div>{asset.artifact_filename && <div><dt>{ui.file}</dt><dd>{asset.artifact_filename}</dd></div>}{asset.checksum && <div><dt>{ui.checksum}</dt><dd><code>{asset.checksum}</code></dd></div>}</dl>}
        {asset?.release_notes && <details><summary>{ui.notes}</summary><p className="artifact-text">{asset.release_notes}</p></details>}
      </div>
      <div className="download-next"><p>{ui.access}</p>
        <button type="button" className="button button-primary" disabled={!asset || !onPrepare || Boolean(disabledReason) || state.status === 'preparing'} onClick={onPrepare} aria-describedby={`${id}-status`}>{state.status === 'preparing' ? ui.preparing : ui.prepare}</button>
        <div id={`${id}-status`} role="status" aria-live="polite">{disabledReason || (state.status === 'error' ? state.message : state.status === 'ready' ? ui.ready : '')}</div>
        {state.status === 'ready' && !disabledReason && <a className="button button-primary" href={state.href} rel="noreferrer" referrerPolicy="no-referrer">{ui.open}</a>}
      </div>
    </div>}
  </section>;
}
