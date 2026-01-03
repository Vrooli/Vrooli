import type { BehaviorSettings, MouseMovementStyle, ScrollStyle } from '@/domains/recording/types/types';

interface BehaviorSectionProps {
  behavior: BehaviorSettings;
  onChange: <K extends keyof BehaviorSettings>(key: K, value: BehaviorSettings[K]) => void;
}

export function BehaviorSection({ behavior, onChange }: BehaviorSectionProps) {
  return (
    <div className="space-y-6">
      {/* Typing Delays - Inter-keystroke */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-1">Typing Speed</h4>
        <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">Delay between each keystroke</p>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Min Delay (ms)</label>
            <input
              type="number"
              value={behavior.typing_delay_min ?? ''}
              onChange={(e) => onChange('typing_delay_min', e.target.value ? parseInt(e.target.value) : undefined)}
              placeholder="50"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Max Delay (ms)</label>
            <input
              type="number"
              value={behavior.typing_delay_max ?? ''}
              onChange={(e) => onChange('typing_delay_max', e.target.value ? parseInt(e.target.value) : undefined)}
              placeholder="150"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
          </div>
        </div>
      </div>

      {/* Typing Start Delay */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-1">Pre-Typing Delay</h4>
        <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">Pause before starting to type (simulates human thinking)</p>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Min Delay (ms)</label>
            <input
              type="number"
              value={behavior.typing_start_delay_min ?? ''}
              onChange={(e) => onChange('typing_start_delay_min', e.target.value ? parseInt(e.target.value) : undefined)}
              placeholder="100"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Max Delay (ms)</label>
            <input
              type="number"
              value={behavior.typing_start_delay_max ?? ''}
              onChange={(e) => onChange('typing_start_delay_max', e.target.value ? parseInt(e.target.value) : undefined)}
              placeholder="300"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
          </div>
        </div>
      </div>

      {/* Typing Paste Threshold */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-1">Paste Threshold</h4>
        <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">
          Long text is pasted instead of typed (0 = always type, -1 = always paste)
        </p>
        <div>
          <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Character Limit</label>
          <input
            type="number"
            value={behavior.typing_paste_threshold ?? ''}
            onChange={(e) => onChange('typing_paste_threshold', e.target.value ? parseInt(e.target.value) : undefined)}
            placeholder="200"
            className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
          />
        </div>
      </div>

      {/* Typing Variance */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-1">Enhanced Typing Variance</h4>
        <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">
          Simulate human typing patterns (faster for common pairs, slower for capitals)
        </p>
        <label className="flex items-center gap-2">
          <input
            type="checkbox"
            checked={behavior.typing_variance_enabled ?? false}
            onChange={(e) => onChange('typing_variance_enabled', e.target.checked)}
            className="rounded border-gray-300 dark:border-gray-600"
          />
          <span className="text-sm text-gray-700 dark:text-gray-300">Enable character-aware typing variance</span>
        </label>
      </div>

      {/* Mouse Movement */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">Mouse Movement</h4>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Movement Style</label>
            <select
              value={behavior.mouse_movement_style ?? 'linear'}
              onChange={(e) => onChange('mouse_movement_style', e.target.value as MouseMovementStyle)}
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            >
              <option value="linear">Linear (Fast)</option>
              <option value="bezier">Bezier (Smooth)</option>
              <option value="natural">Natural (Human-like)</option>
            </select>
          </div>
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Jitter Amount (px)</label>
            <input
              type="number"
              step="0.5"
              value={behavior.mouse_jitter_amount ?? ''}
              onChange={(e) => onChange('mouse_jitter_amount', e.target.value ? parseFloat(e.target.value) : undefined)}
              placeholder="2"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
          </div>
        </div>
      </div>

      {/* Click Delays */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">Click Behavior</h4>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Min Delay (ms)</label>
            <input
              type="number"
              value={behavior.click_delay_min ?? ''}
              onChange={(e) => onChange('click_delay_min', e.target.value ? parseInt(e.target.value) : undefined)}
              placeholder="100"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Max Delay (ms)</label>
            <input
              type="number"
              value={behavior.click_delay_max ?? ''}
              onChange={(e) => onChange('click_delay_max', e.target.value ? parseInt(e.target.value) : undefined)}
              placeholder="300"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
          </div>
        </div>
      </div>

      {/* Scroll Style */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">Scroll Behavior</h4>
        <div>
          <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Scroll Style</label>
          <select
            value={behavior.scroll_style ?? 'smooth'}
            onChange={(e) => onChange('scroll_style', e.target.value as ScrollStyle)}
            className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
          >
            <option value="smooth">Smooth (Continuous)</option>
            <option value="stepped">Stepped (Human-like)</option>
          </select>
        </div>
        <div className="grid grid-cols-2 gap-4 mt-3">
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Min Speed (px/step)</label>
            <input
              type="number"
              value={behavior.scroll_speed_min ?? ''}
              onChange={(e) => onChange('scroll_speed_min', e.target.value ? parseInt(e.target.value) : undefined)}
              placeholder="50"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Max Speed (px/step)</label>
            <input
              type="number"
              value={behavior.scroll_speed_max ?? ''}
              onChange={(e) => onChange('scroll_speed_max', e.target.value ? parseInt(e.target.value) : undefined)}
              placeholder="200"
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
            />
          </div>
        </div>
      </div>

      {/* Micro-pauses */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">Micro-Pauses</h4>
        <label className="flex items-center gap-2 mb-3">
          <input
            type="checkbox"
            checked={behavior.micro_pause_enabled ?? false}
            onChange={(e) => onChange('micro_pause_enabled', e.target.checked)}
            className="rounded border-gray-300 dark:border-gray-600"
          />
          <span className="text-sm text-gray-700 dark:text-gray-300">Enable random micro-pauses</span>
        </label>
        {behavior.micro_pause_enabled && (
          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Min (ms)</label>
              <input
                type="number"
                value={behavior.micro_pause_min_ms ?? ''}
                onChange={(e) => onChange('micro_pause_min_ms', e.target.value ? parseInt(e.target.value) : undefined)}
                placeholder="500"
                className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
              />
            </div>
            <div>
              <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Max (ms)</label>
              <input
                type="number"
                value={behavior.micro_pause_max_ms ?? ''}
                onChange={(e) => onChange('micro_pause_max_ms', e.target.value ? parseInt(e.target.value) : undefined)}
                placeholder="2000"
                className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
              />
            </div>
            <div>
              <label className="block text-xs text-gray-600 dark:text-gray-400 mb-1">Frequency</label>
              <input
                type="number"
                step="0.01"
                value={behavior.micro_pause_frequency ?? ''}
                onChange={(e) =>
                  onChange('micro_pause_frequency', e.target.value ? parseFloat(e.target.value) : undefined)
                }
                placeholder="0.15"
                className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100"
              />
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
