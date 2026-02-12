import { useEffect, useState } from 'react';
import { Info, CheckCircle, XCircle, Loader2, AlertTriangle } from 'lucide-react';
import './RulesPanel.css';

interface RuleDef {
  id: string;
  name: string;
  description: string;
  why: string;
  category: string;
  severity: string;
  slot: string;
  slot_file: string;
  tech_stack: string[];
  recommendation: string;
  good_example?: string;
  bad_example?: string;
  standard?: string;
  enabled: boolean;
}

interface RulesApiResponse {
  success?: boolean;
  data?: RuleDef[];
}

interface SlotGroup {
  slot: string;
  slotFile: string;
  rules: RuleDef[];
}

function RuleCard({ rule }: { rule: RuleDef }) {
  const hasExamples = rule.good_example || rule.bad_example;

  return (
    <div className={`rules-panel__rule rules-panel__rule--${rule.severity}`}>
      <div className="rules-panel__rule-header">
        <span className="rules-panel__rule-name">{rule.name}</span>
        <span className={`rules-panel__severity rules-panel__severity--${rule.severity}`}>
          {rule.severity}
        </span>
      </div>

      <p className="rules-panel__why">{rule.why || rule.description}</p>

      <div className="rules-panel__recommendation">
        <Info size={14} aria-hidden />
        <span>{rule.recommendation}</span>
      </div>

      {hasExamples && (
        <div className="rules-panel__examples">
          {rule.good_example && (
            <div className="rules-panel__example rules-panel__example--good">
              <span className="rules-panel__example-label">
                <CheckCircle size={12} aria-hidden />
                Good
              </span>
              <pre><code>{rule.good_example}</code></pre>
            </div>
          )}
          {rule.bad_example && (
            <div className="rules-panel__example rules-panel__example--bad">
              <span className="rules-panel__example-label">
                <XCircle size={12} aria-hidden />
                Bad
              </span>
              <pre><code>{rule.bad_example}</code></pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function groupBySlot(rules: RuleDef[]): SlotGroup[] {
  const groups = new Map<string, SlotGroup>();
  for (const rule of rules) {
    const slot = rule.slot || 'Other';
    const existing = groups.get(slot);
    if (existing) {
      existing.rules.push(rule);
    } else {
      groups.set(slot, { slot, slotFile: rule.slot_file || '', rules: [rule] });
    }
  }
  // Sort groups by slot label
  return [...groups.values()].sort((a, b) => a.slot.localeCompare(b.slot));
}

async function fetchRuleDefs(): Promise<RuleDef[]> {
  const res = await fetch('/api/v1/rules');
  if (!res.ok) throw new Error(`Failed to fetch rules: ${res.status}`);
  const json = (await res.json()) as RulesApiResponse;
  return json.data ?? [];
}

export default function RulesPanel() {
  const [rules, setRules] = useState<RuleDef[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    fetchRuleDefs()
      .then((data) => {
        if (!cancelled) {
          setRules(data);
          setLoading(false);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to load rules');
          setLoading(false);
        }
      });
    return () => { cancelled = true; };
  }, []);

  if (loading) {
    return (
      <div className="rules-panel rules-panel--loading">
        <Loader2 className="rules-panel__spinner" size={24} />
        <span>Loading rules&hellip;</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rules-panel rules-panel--error">
        <AlertTriangle size={20} />
        <span>{error}</span>
      </div>
    );
  }

  const slotGroups = groupBySlot(rules);

  return (
    <div className="rules-panel">
      <p className="rules-panel__intro">
        These rules ensure scenario UIs work reliably across all Vrooli deployment
        contexts &mdash; localhost, Cloudflare tunnel, and app-monitor proxy/iframe.
        Each rule maps to a specific file slot in the canonical layout.
      </p>

      {slotGroups.map((group) => (
        <section key={group.slot} className="rules-panel__slot-group">
          <h3 className="rules-panel__slot-header">
            <span className="rules-panel__slot-badge">{group.slot}</span>
            <code className="rules-panel__slot-file">{group.slotFile}</code>
          </h3>

          {group.rules.map((rule) => (
            <RuleCard key={rule.id} rule={rule} />
          ))}
        </section>
      ))}
    </div>
  );
}
