import { memo } from 'react';
import { AlertCircle, Calendar, FileText, FolderOpen, Link, PanelRightClose, Sparkles } from 'lucide-react';
import { type GeneratedScenario, type PreviewLinks } from '../lib/api';
import { AdminCredentialsHint } from './AdminCredentialsHint';
import { type LifecycleControlConfig } from './types';

interface ScenarioInfoPanelProps {
  open: boolean;
  onClose: () => void;
  scenario: GeneratedScenario;
  previewLinks?: PreviewLinks;
  lifecycleControls?: LifecycleControlConfig | null;
}

export const ScenarioInfoPanel = memo(function ScenarioInfoPanel({
  open,
  onClose,
  scenario,
  previewLinks,
  lifecycleControls,
}: ScenarioInfoPanelProps) {
  if (!open) {
    return null;
  }

  const links = previewLinks?.links ?? {};

  return (
    <>
      <div className="scenario-info-panel__overlay" onClick={onClose} />
      <aside className="scenario-info-panel" aria-label="Scenario information panel">
        <div className="scenario-info-panel__header">
          <div>
            <p className="scenario-info-panel__eyebrow">Scenario</p>
            <h2 className="scenario-info-panel__title">{scenario.name}</h2>
            <p className="scenario-info-panel__slug">{scenario.scenario_id}</p>
          </div>
          <button
            type="button"
            className="scenario-info-panel__close-btn"
            onClick={onClose}
            aria-label="Close scenario information"
          >
            <PanelRightClose size={20} />
          </button>
        </div>

        <div className="scenario-info-panel__content">
          <section className="scenario-info-panel__section">
            <h3>Details</h3>
            <dl>
              {scenario.template_id && (
                <div>
                  <dt>Template</dt>
                  <dd>{scenario.template_id}{scenario.template_version ? ` · v${scenario.template_version}` : ''}</dd>
                </div>
              )}
              <div>
                <dt>Location</dt>
                <dd>
                  <code>{scenario.path}</code>
                </dd>
              </div>
              {scenario.generated_at && (
                <div>
                  <dt>Generated</dt>
                  <dd className="scenario-info-panel__meta">
                    <Calendar size={14} />
                    {new Date(scenario.generated_at).toLocaleString()}
                  </dd>
                </div>
              )}
            </dl>
          </section>

          <section className="scenario-info-panel__section">
            <h3>Preview Links</h3>
            {links.public || links.admin || links.admin_login ? (
              <ul className="scenario-info-panel__links">
                {links.public && (
                  <li>
                    <a href={links.public} target="_blank" rel="noreferrer">
                      <GlobeIcon />
                      Public Landing
                      <Link size={14} />
                    </a>
                  </li>
                )}
                {links.admin && (
                  <li>
                    <a href={links.admin} target="_blank" rel="noreferrer">
                      <FileText size={14} />
                      Admin Portal
                      <Link size={14} />
                    </a>
                  </li>
                )}
                {!links.admin && links.admin_login && (
                  <li>
                    <a href={links.admin_login} target="_blank" rel="noreferrer">
                      <FolderOpen size={14} />
                      Admin Login
                      <Link size={14} />
                    </a>
                  </li>
                )}
              </ul>
            ) : (
              <p className="scenario-info-panel__empty">
                <AlertCircle size={14} />
                Preview links will appear once the scenario is running.
              </p>
            )}
          </section>

          <section className="scenario-info-panel__section">
            <h3>Credentials</h3>
            <AdminCredentialsHint />
          </section>
        </div>

        {lifecycleControls && (
          <div className="scenario-info-panel__footer">
            <div className="scenario-info-panel__status">
              <Sparkles size={16} />
              <span>{lifecycleControls.running ? 'Running' : 'Stopped'}</span>
            </div>
            <div className="scenario-info-panel__footer-actions">
              <button
                type="button"
                onClick={lifecycleControls.onStart}
                disabled={lifecycleControls.loading || lifecycleControls.running}
              >
                Start
              </button>
              <button
                type="button"
                onClick={lifecycleControls.onStop}
                disabled={lifecycleControls.loading || !lifecycleControls.onStop || !lifecycleControls.running}
              >
                Stop
              </button>
              <button
                type="button"
                onClick={lifecycleControls.onRestart}
                disabled={lifecycleControls.loading || !lifecycleControls.onRestart || !lifecycleControls.running}
              >
                Restart
              </button>
            </div>
          </div>
        )}
      </aside>
    </>
  );
});

function GlobeIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" className="scenario-info-panel__inline-icon" aria-hidden="true">
      <circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="1.5" fill="none" />
      <ellipse cx="12" cy="12" rx="5" ry="10" stroke="currentColor" strokeWidth="1.5" fill="none" />
      <line x1="2" y1="12" x2="22" y2="12" stroke="currentColor" strokeWidth="1.5" />
    </svg>
  );
}

export default ScenarioInfoPanel;
