import { useState } from 'react';
import type { ArtStudiesFixture, PresentationResources, WorkspaceFixture } from './resources';
import { AssetImage, ProductMark } from './primitives';

export function Workspace({ fixture: w, interactive = true }: { fixture: WorkspaceFixture; interactive?: boolean }) {
  const [view, setView] = useState<'terminal' | 'messages'>('terminal');
  return <div className="workspace" data-capture-landmark="product">
    <div className="window-bar"><span className="traffic" aria-hidden="true"><i /><i /><i /></span><span className="window-title"><ProductMark kind={w.mark} />{w.title}</span><span className="window-meta">{w.group}</span><span className="window-controls" aria-hidden="true">⌘ K</span></div>
    <div className="workspace-body">
      <aside className="workspace-sidebar"><div className="workspace-side-title">{w.group_label}<span aria-hidden="true">+</span></div>
        {w.groups.map((group, index) => <div key={index} className={`workspace-group ${index === 0 ? 'selected' : ''}`}><span aria-hidden="true">{index === 0 ? '◈' : '◇'}</span>{group}</div>)}
        <div className="workspace-side-title sessions-title">{w.sessions_label}</div>
        {w.sessions.map((session, index) => <div key={index} className="workspace-session"><i className={`session-dot d${String(index)}`} aria-hidden="true" />{session}</div>)}
        <div className="sidebar-bottom"><span className="avatar">{w.avatar}</span><span>{w.today}</span><span aria-hidden="true">⌘</span></div>
      </aside>
      <div className="workspace-main"><div className="pane-head"><div><i className="session-dot" aria-hidden="true" /><b>{w.role}</b><span>{w.model}</span></div>
        <div className="demo-tabs" role={interactive ? 'group' : undefined} aria-label={interactive ? w.tabs_label : undefined}>
          {(['terminal', 'messages'] as const).map(tab => interactive
            ? <button type="button" key={tab} onClick={() => { setView(tab); }} aria-pressed={view === tab}>{tab === 'terminal' ? w.terminal_label : w.messages_label}</button>
            : <span key={tab} className="static-control" data-selected={tab === 'terminal'}>{tab === 'terminal' ? w.terminal_label : w.messages_label}</span>)}
        </div>
      </div>
      <div className="terminal-view" hidden={view !== 'terminal'}><div className="terminal-heading"><span className="terminal-spark" aria-hidden="true">✳</span><div><b>{w.model}</b><small>{w.branch}</small></div></div>
        <div className="terminal-prompt"><span aria-hidden="true">›</span>{w.prompt}</div><p className="terminal-answer">{w.answer}</p>
        <div className="change-list"><small>{w.file_label}</small>{w.files.map((file, index) => <div key={index}><span className="file-symbol" aria-hidden="true">‹›</span><span>{file}</span><span className="file-diff">{w.file_changes[file]}</span></div>)}</div>
        <div className="terminal-command">{w.command}</div><div className="terminal-checks">{w.checks.map((check, index) => <div key={index}>{check}</div>)}</div>
      </div>
      <div className="messages-view" hidden={view !== 'messages'}><small>{w.message_label}</small><p className="user-bubble">{w.prompt}</p><small>{w.reply_label}</small><p>{w.answer}</p><div className="message-changes">{w.diff.map((line, index) => <div key={index}>{line}</div>)}</div><p className="ready-note">{w.ready}</p></div>
      <div className="composer"><span>{w.composer}</span><span className="send-icon" aria-hidden="true">↑</span></div></div>
      <aside className="review-pane"><div className="pane-head"><div><i className="session-dot d1" aria-hidden="true" /><b>{w.reviewer}</b><span>{w.reviewer_model}</span></div></div><div className="review-content"><div className="review-mark" aria-hidden="true">◎</div><small>{w.return_label}</small><p className="review-title">{w.return_title}</p><p>{w.review_message}</p>{w.files[0] && <div className="review-file"><span aria-hidden="true">↳</span> {w.files[0]}</div>}<div className="review-lines" aria-hidden="true"><i /><i /><i /><i /><i /></div></div></aside>
    </div><div className="workspace-status"><span><i className="session-dot" aria-hidden="true" />{w.status}</span><span>{w.branch}</span></div>
  </div>;
}

export function ArtStudies({ fixture, resources, interactive = true, eager = false }: {
  fixture: ArtStudiesFixture; resources: PresentationResources; interactive?: boolean; eager?: boolean;
}) {
  const [selected, setSelected] = useState(fixture.selected);
  const index = fixture.styles.indexOf(selected);
  const asset = fixture.asset_refs[index];
  return <div className="backdrop-app"><div className="backdrop-bar"><ProductMark kind={fixture.mark} /><b>{fixture.title}</b><span>{fixture.surface}</span></div>
    <div className="backdrop-canvas">{asset && <AssetImage assetRef={asset} resources={resources} eager={eager} />}<span className="canvas-label">{fixture.badge}</span><div className="canvas-caption"><span>{String(index + 1).padStart(2, '0')}—{String(fixture.asset_refs.length).padStart(2, '0')}</span><span>{selected}</span></div></div>
    <div className="backdrop-bottom"><div className="art-thumbnails">{fixture.asset_refs.map((asset_ref, index) => interactive
      ? <button key={asset_ref} type="button" aria-label={fixture.styles[index]} aria-pressed={selected === fixture.styles[index]} onClick={() => { setSelected(fixture.styles[index] ?? fixture.selected); }}><AssetImage assetRef={asset_ref} resources={resources} alt="" /></button>
      : <span key={asset_ref} className="static-control" data-selected={fixture.styles[index] === fixture.selected}><AssetImage assetRef={asset_ref} resources={resources} alt="" /></span>)}</div><div className="palette"><span>{fixture.palette}</span><i /><i /><i /></div></div>
  </div>;
}

export function Visual({ visualRef, resources, interactive = true, eager = false }: {
  visualRef: string; resources: PresentationResources; interactive?: boolean; eager?: boolean;
}) {
  const fixture = resources.visuals[visualRef];
  if (!fixture) return <AssetImage assetRef={visualRef} resources={resources} eager={eager} />;
  switch (fixture.kind) {
    case 'workflow': return <div className="workflow-exhibit"><p className="eyebrow">{fixture.label}</p><h3>{fixture.title}</h3><ol>{fixture.steps.map(step => <li key={step.number}><span>{step.number}</span><h4>{step.title}</h4><p>{step.description}</p></li>)}</ol><div className="workflow-browser"><h4>{fixture.browser_title}</h4>{fixture.browser_rows.map((row, index) => <p key={index}>{row}</p>)}</div><p>{fixture.note}</p></div>;
    case 'workspace': return <Workspace fixture={fixture} interactive={interactive} />;
    case 'backdrop': return <ArtStudies fixture={fixture} resources={resources} interactive={interactive} eager={eager} />;
  }
}

export function Phone({ fixture: w }: { fixture: WorkspaceFixture }) {
  return <div className="phone-stage"><div className="phone-orbit" aria-hidden="true" /><div className="phone">
    <div className="phone-top"><span>{w.time}</span><i /><span aria-hidden="true">▮▮ ◕</span></div>
    <div className="phone-header"><ProductMark kind={w.mark} /><b>{w.title}</b><span aria-hidden="true">☰</span></div>
    <div className="phone-group">{w.group}<span aria-hidden="true">⌄</span></div>
    <div className="phone-chat"><small>{w.message_label}</small><p className="phone-user">{w.prompt}</p><small><i className="session-dot" aria-hidden="true" />{w.reply_label}</small><p>{w.answer}</p>
      {w.files[0] && <div className="phone-file"><span aria-hidden="true">‹›</span><div>{w.files[0]}<small>{w.ready}</small></div><span aria-hidden="true">↗</span></div>}
    </div><div className="phone-composer">{w.composer}<span aria-hidden="true">↑</span></div>
    <div className="phone-keys">{w.keyboard.map((key, index) => <span key={index}>{key}</span>)}</div><div className="phone-home" aria-hidden="true" />
  </div></div>;
}
