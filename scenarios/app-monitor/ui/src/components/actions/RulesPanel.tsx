import { Info, CheckCircle, XCircle } from 'lucide-react';
import { INTEROP_SLOT_GROUPS } from './interopRulesData';
import type { InteropRuleDef } from './interopRulesData';
import './RulesPanel.css';

function RuleCard({ rule }: { rule: InteropRuleDef }) {
  const hasExamples = rule.goodExample || rule.badExample;

  return (
    <div className={`rules-panel__rule rules-panel__rule--${rule.severity}`}>
      <div className="rules-panel__rule-header">
        <span className="rules-panel__rule-name">{rule.name}</span>
        <span className={`rules-panel__severity rules-panel__severity--${rule.severity}`}>
          {rule.severity}
        </span>
      </div>

      <p className="rules-panel__why">{rule.why}</p>

      <div className="rules-panel__recommendation">
        <Info size={14} aria-hidden />
        <span>{rule.recommendation}</span>
      </div>

      {hasExamples && (
        <div className="rules-panel__examples">
          {rule.goodExample && (
            <div className="rules-panel__example rules-panel__example--good">
              <span className="rules-panel__example-label">
                <CheckCircle size={12} aria-hidden />
                Good
              </span>
              <pre><code>{rule.goodExample}</code></pre>
            </div>
          )}
          {rule.badExample && (
            <div className="rules-panel__example rules-panel__example--bad">
              <span className="rules-panel__example-label">
                <XCircle size={12} aria-hidden />
                Bad
              </span>
              <pre><code>{rule.badExample}</code></pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default function RulesPanel() {
  return (
    <div className="rules-panel">
      <p className="rules-panel__intro">
        These rules ensure scenario UIs work reliably across all Vrooli deployment
        contexts &mdash; localhost, Cloudflare tunnel, and app-monitor proxy/iframe.
        Each rule maps to a specific file slot in the canonical layout.
      </p>

      {INTEROP_SLOT_GROUPS.map((group) => (
        <section key={group.slot} className="rules-panel__slot-group">
          <h3 className="rules-panel__slot-header">
            <span className="rules-panel__slot-badge">{group.slot}</span>
            <code className="rules-panel__slot-file">{group.file}</code>
            <span className="rules-panel__slot-desc">&mdash; {group.description}</span>
          </h3>

          {group.rules.map((rule) => (
            <RuleCard key={rule.id} rule={rule} />
          ))}
        </section>
      ))}
    </div>
  );
}
