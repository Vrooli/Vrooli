import { HelpCircle } from 'lucide-react';
import { Tooltip } from '../Tooltip';

interface BriefTextareaProps {
  value: string;
  onChange: (value: string) => void;
  hasError: boolean;
}

export function BriefTextarea({ value, onChange, hasError }: BriefTextareaProps) {
  return (
    <div>
      <label htmlFor="customize-brief-input" className="block text-sm font-medium text-slate-300 mb-2">
        Customization Brief
        <Tooltip content="Describe your goals, target audience, value proposition, desired tone, call-to-action, and success metrics. The more specific, the better the results.">
          <HelpCircle className="inline h-3.5 w-3.5 ml-1 text-slate-500 hover:text-slate-400" />
        </Tooltip>
      </label>
      <textarea
        id="customize-brief-input"
        data-testid="customize-brief-input"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full rounded-lg border border-white/10 bg-slate-900 px-3 py-2.5 text-sm text-slate-100 placeholder-slate-500 focus:border-emerald-400 focus:ring-2 focus:ring-emerald-500/20 outline-none transition-colors min-h-[120px] resize-y"
        placeholder="Example: Target SaaS founders looking for rapid launch. Professional but friendly tone. Emphasize time savings and built-in features. CTA: Start Free Trial. Success metric: 10% conversion rate."
        required
        aria-required="true"
        aria-invalid={hasError ? 'true' : 'false'}
        aria-describedby="customize-brief-helper"
      />
      <div className="flex items-center justify-between mt-1.5">
        <p id="customize-brief-helper" className="text-xs text-slate-400">
          Be specific about goals, audience, tone, and desired outcomes
        </p>
        <p className="text-xs text-slate-500">
          {value.length} characters
        </p>
      </div>
    </div>
  );
}
