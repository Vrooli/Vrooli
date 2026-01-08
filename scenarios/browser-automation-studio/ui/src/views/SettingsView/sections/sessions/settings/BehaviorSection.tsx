import type { BehaviorSettings, MouseMovementStyle, ScrollStyle } from '@/domains/recording/types/types';
import {
  FormNumberInput,
  FormSelect,
  FormCheckbox,
  FormFieldGroup,
  FormGrid,
} from '@/components/form';

interface BehaviorSectionProps {
  behavior: BehaviorSettings;
  onChange: <K extends keyof BehaviorSettings>(key: K, value: BehaviorSettings[K]) => void;
}

const mouseMovementOptions = [
  { value: 'linear' as MouseMovementStyle, label: 'Linear (Fast)' },
  { value: 'bezier' as MouseMovementStyle, label: 'Bezier (Smooth)' },
  { value: 'natural' as MouseMovementStyle, label: 'Natural (Human-like)' },
];

const scrollStyleOptions = [
  { value: 'smooth' as ScrollStyle, label: 'Smooth (Continuous)' },
  { value: 'stepped' as ScrollStyle, label: 'Stepped (Human-like)' },
];

export function BehaviorSection({ behavior, onChange }: BehaviorSectionProps) {
  return (
    <div className="space-y-6">
      {/* Typing Speed */}
      <FormFieldGroup title="Typing Speed" description="Delay between each keystroke">
        <FormGrid cols={2}>
          <FormNumberInput
            value={behavior.typing_delay_min}
            onChange={(v) => onChange('typing_delay_min', v)}
            label="Min Delay (ms)"
            placeholder="50"
          />
          <FormNumberInput
            value={behavior.typing_delay_max}
            onChange={(v) => onChange('typing_delay_max', v)}
            label="Max Delay (ms)"
            placeholder="150"
          />
        </FormGrid>
      </FormFieldGroup>

      {/* Pre-Typing Delay */}
      <FormFieldGroup title="Pre-Typing Delay" description="Pause before starting to type (simulates human thinking)">
        <FormGrid cols={2}>
          <FormNumberInput
            value={behavior.typing_start_delay_min}
            onChange={(v) => onChange('typing_start_delay_min', v)}
            label="Min Delay (ms)"
            placeholder="100"
          />
          <FormNumberInput
            value={behavior.typing_start_delay_max}
            onChange={(v) => onChange('typing_start_delay_max', v)}
            label="Max Delay (ms)"
            placeholder="300"
          />
        </FormGrid>
      </FormFieldGroup>

      {/* Paste Threshold */}
      <FormFieldGroup
        title="Paste Threshold"
        description="Long text is pasted instead of typed (0 = always type, -1 = always paste)"
      >
        <FormNumberInput
          value={behavior.typing_paste_threshold}
          onChange={(v) => onChange('typing_paste_threshold', v)}
          label="Character Limit"
          placeholder="200"
        />
      </FormFieldGroup>

      {/* Typing Variance */}
      <FormFieldGroup
        title="Enhanced Typing Variance"
        description="Simulate human typing patterns (faster for common pairs, slower for capitals)"
      >
        <FormCheckbox
          checked={behavior.typing_variance_enabled ?? false}
          onChange={(v) => onChange('typing_variance_enabled', v)}
          label="Enable character-aware typing variance"
        />
      </FormFieldGroup>

      {/* Mouse Movement */}
      <FormFieldGroup title="Mouse Movement">
        <FormGrid cols={2}>
          <FormSelect
            value={behavior.mouse_movement_style ?? 'linear'}
            onChange={(v) => onChange('mouse_movement_style', v)}
            label="Movement Style"
            options={mouseMovementOptions}
          />
          <FormNumberInput
            value={behavior.mouse_jitter_amount}
            onChange={(v) => onChange('mouse_jitter_amount', v)}
            label="Jitter Amount (px)"
            placeholder="2"
            step={0.5}
          />
        </FormGrid>
      </FormFieldGroup>

      {/* Click Behavior */}
      <FormFieldGroup title="Click Behavior">
        <FormGrid cols={2}>
          <FormNumberInput
            value={behavior.click_delay_min}
            onChange={(v) => onChange('click_delay_min', v)}
            label="Min Delay (ms)"
            placeholder="100"
          />
          <FormNumberInput
            value={behavior.click_delay_max}
            onChange={(v) => onChange('click_delay_max', v)}
            label="Max Delay (ms)"
            placeholder="300"
          />
        </FormGrid>
      </FormFieldGroup>

      {/* Scroll Behavior */}
      <FormFieldGroup title="Scroll Behavior">
        <FormSelect
          value={behavior.scroll_style ?? 'smooth'}
          onChange={(v) => onChange('scroll_style', v)}
          label="Scroll Style"
          options={scrollStyleOptions}
        />
        <FormGrid cols={2} className="mt-3">
          <FormNumberInput
            value={behavior.scroll_speed_min}
            onChange={(v) => onChange('scroll_speed_min', v)}
            label="Min Speed (px/step)"
            placeholder="50"
          />
          <FormNumberInput
            value={behavior.scroll_speed_max}
            onChange={(v) => onChange('scroll_speed_max', v)}
            label="Max Speed (px/step)"
            placeholder="200"
          />
        </FormGrid>
      </FormFieldGroup>

      {/* Micro-pauses */}
      <FormFieldGroup title="Micro-Pauses">
        <FormCheckbox
          checked={behavior.micro_pause_enabled ?? false}
          onChange={(v) => onChange('micro_pause_enabled', v)}
          label="Enable random micro-pauses"
          className="mb-3"
        />
        {behavior.micro_pause_enabled && (
          <FormGrid cols={3}>
            <FormNumberInput
              value={behavior.micro_pause_min_ms}
              onChange={(v) => onChange('micro_pause_min_ms', v)}
              label="Min (ms)"
              placeholder="500"
            />
            <FormNumberInput
              value={behavior.micro_pause_max_ms}
              onChange={(v) => onChange('micro_pause_max_ms', v)}
              label="Max (ms)"
              placeholder="2000"
            />
            <FormNumberInput
              value={behavior.micro_pause_frequency}
              onChange={(v) => onChange('micro_pause_frequency', v)}
              label="Frequency"
              placeholder="0.15"
              step={0.01}
            />
          </FormGrid>
        )}
      </FormFieldGroup>
    </div>
  );
}
