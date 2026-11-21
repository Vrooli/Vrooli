import { useId } from 'react';
import type { DownloadAsset } from '../../../shared/api/types';
import { getPlatformLabel, resolveSigningNotice } from '../services/downloads.service';
import { downloadSystemUi as ui } from './systemUi';
import { ActionLink } from './primitives';
import { SigningNoticeDisclosure } from './SigningNoticeDisclosure';
import type { Action, ResolvedActions } from './types';

export type DownloadState = { status: 'idle' | 'preparing' } | { status: 'error'; message: string } | { status: 'ready'; href: string };
export interface DownloadChooserProps {
  title: string; description: string; options: DownloadAsset[]; selected: string;
  onSelect: (value: string) => void; onPrepare?: () => void; state: DownloadState;
  unavailableReason: string; disabledReason?: string;
  appMetadata?: Record<string, unknown>;
  launchActions?: Action[]; resolvedActions?: ResolvedActions;
  /** Platform detected by the page owner; the view only decorates with it. */
  detectedPlatform?: string;
}

/** Finite platform glyphs from the presentation's own symbol vocabulary. */
const platformGlyphs: Record<string, string> = { windows: '⊞', mac: '⌘', linux: '❯_' };

/** Controlled native view. No auth, transport, navigation or analytics effects. */
export function DownloadChooser({ title, description, options, selected, onSelect, onPrepare, state, unavailableReason, disabledReason, appMetadata, launchActions = [], resolvedActions, detectedPlatform }: DownloadChooserProps) {
  const id = useId();
  const asset = selected === '' ? undefined : options[Number(selected)];
  const notice = resolveSigningNotice(appMetadata, asset?.metadata);
  const platforms = [...new Set(options.map(option => option.platform))];
  return <section className="download-chooser" aria-labelledby={`${id}-title`}>
    <div className="section-intro"><p className="eyebrow"><span />{ui.title}</p><h1 id={`${id}-title`}>{title}</h1><p className="hero-description">{description}</p></div>
    {launchActions.length > 0 && <div className="commerce-actions download-launch">{launchActions.map((action, index) => <ActionLink key={index} action={action} resolvedActions={resolvedActions} reason={unavailableReason} className={options.length > 0 ? 'button-text' : 'button-primary'} />)}</div>}
    {options.length === 0 ? <div className="download-empty">
      <span className="download-empty-glyph" aria-hidden="true"><svg viewBox="0 0 24 24" fill="none"><path d="M12 4v11m0 0 5-5m-5 5-5-5M5 20h14" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" /></svg></span>
      <p role="status">{ui.noInstallers}</p>
      <p className="download-empty-detail">{ui.noInstallersDetail}</p>
      {launchActions.length > 0 && <p className="download-empty-detail">{ui.launchNote}</p>}
    </div> : <div className="download-panel">
      <div className="download-configure">
        {platforms.length > 1 && <div className="platform-picker" role="group" aria-label={ui.platform}>
          {platforms.map(platform => {
            const index = options.findIndex(option => option.platform === platform);
            const active = asset ? asset.platform === platform : false;
            return <button key={platform} type="button" className="platform-card" aria-pressed={active} onClick={() => { onSelect(String(index)); }}>
              <span className="platform-glyph" aria-hidden="true">{platformGlyphs[platform] ?? '◇'}</span>
              <span className="platform-name">{getPlatformLabel(platform)}</span>
              {platform === detectedPlatform && <span className="platform-detected">{ui.detected}</span>}
            </button>;
          })}
        </div>}
        <label htmlFor={`${id}-choice`}>{ui.choose}</label>
        <select id={`${id}-choice`} value={selected} onChange={event => { onSelect(event.target.value); }}>
          <option value="">{ui.choose}</option>
          {platforms.map(platform => <optgroup key={platform} label={getPlatformLabel(platform)}>{options.map((option, index) => option.platform === platform && <option key={index} value={String(index)}>{getPlatformLabel(platform)} · {option.release_version}{option.artifact_filename ? ` · ${option.artifact_filename}` : ''}</option>)}</optgroup>)}
        </select>
        {asset && <dl className="download-facts"><div><dt>{ui.platform}</dt><dd>{getPlatformLabel(asset.platform)}</dd></div><div><dt>{ui.release}</dt><dd>{asset.release_version}</dd></div>{asset.artifact_filename && <div><dt>{ui.file}</dt><dd>{asset.artifact_filename}</dd></div>}{asset.checksum && <div><dt>{ui.checksum}</dt><dd><code>{asset.checksum}</code></dd></div>}</dl>}
        {asset?.release_notes && <details><summary>{ui.notes}</summary><p className="artifact-text">{asset.release_notes}</p></details>}
      </div>
      <div className="download-next">
        <p className="exhibit-kicker">{ui.nextTitle}</p>
        <ol className="download-steps">{ui.steps.map((step, index) => <li key={index}><span aria-hidden="true">{String(index + 1).padStart(2, '0')}</span>{step}</li>)}</ol>
        <p className="download-access">{ui.access}</p>
        {notice && <SigningNoticeDisclosure notice={notice} />}
        <button type="button" className="button button-primary" disabled={!asset || !onPrepare || Boolean(disabledReason) || state.status === 'preparing'} onClick={onPrepare} aria-describedby={`${id}-status`}>{state.status === 'preparing' ? ui.preparing : ui.prepare}</button>
        <div id={`${id}-status`} className="download-status" role="status" aria-live="polite">{disabledReason || (state.status === 'error' ? state.message : state.status === 'ready' ? ui.ready : '')}</div>
        {state.status === 'ready' && !disabledReason && <a className="button button-primary" href={state.href} rel="noreferrer" referrerPolicy="no-referrer">{ui.open}</a>}
      </div>
    </div>}
  </section>;
}
