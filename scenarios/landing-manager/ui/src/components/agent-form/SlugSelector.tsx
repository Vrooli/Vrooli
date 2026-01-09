import { HelpCircle } from 'lucide-react';
import { Tooltip } from '../Tooltip';
import { slugify } from '../../lib/utils';
import { type GeneratedScenario } from '../../lib/api';

interface SlugSelectorProps {
  value: string;
  generated: GeneratedScenario[];
  onChange: (value: string) => void;
  onSelectScenario: (slug: string) => void;
}

export function SlugSelector({ value, generated, onChange, onSelectScenario }: SlugSelectorProps) {
  const hasScenarios = generated.length > 0;

  return (
    <div>
      <label htmlFor="customize-slug-input" className="block text-sm font-medium text-slate-300 mb-2">
        Scenario Slug
        <Tooltip content="The slug of the scenario you want to customize. Must be a generated scenario from the list above.">
          <HelpCircle className="inline h-3.5 w-3.5 ml-1 text-slate-500 hover:text-slate-400" />
        </Tooltip>
      </label>
      {hasScenarios ? (
        <select
          id="customize-slug-input"
          data-testid="customize-slug-input"
          value={value}
          onChange={(e) => {
            onChange(e.target.value);
            onSelectScenario(e.target.value);
          }}
          className="w-full rounded-lg border border-white/10 bg-slate-900 px-3 py-2.5 text-sm text-slate-100 font-mono focus:border-emerald-400 focus:ring-2 focus:ring-emerald-500/20 outline-none transition-colors"
          aria-describedby="customize-slug-helper"
        >
          <option value="">Select a scenario...</option>
          {generated.map((scenario) => (
            <option key={scenario.scenario_id} value={scenario.scenario_id}>
              {scenario.name} ({scenario.scenario_id})
            </option>
          ))}
        </select>
      ) : (
        <input
          id="customize-slug-input"
          data-testid="customize-slug-input"
          value={value}
          onChange={(e) => onChange(slugify(e.target.value))}
          className="w-full rounded-lg border border-white/10 bg-slate-900 px-3 py-2.5 text-sm text-slate-100 placeholder-slate-500 font-mono focus:border-emerald-400 focus:ring-2 focus:ring-emerald-500/20 outline-none transition-colors"
          placeholder="my-landing-page"
          aria-describedby="customize-slug-helper"
          disabled
        />
      )}
      <p id="customize-slug-helper" className="mt-1.5 text-xs text-slate-400">
        {hasScenarios ? 'Choose from your generated scenarios' : 'Generate a scenario first to customize it'}
      </p>
    </div>
  );
}
