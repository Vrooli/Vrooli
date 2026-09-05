import { useState, type ComponentType, type ReactNode } from "react";

import { PreviewShowcase } from "../../../showcase/PreviewShowcase";

export type ControlledStateConfig = {
  valueProp?: string;
  changeProp?: string;
  initialValue?: unknown;
  title?: string;
  detail?: string;
  status?: string;
};

export type PreviewHarnessProps = {
  subject: ComponentType<Record<string, unknown>>;
  args?: Record<string, unknown>;
  config?: ControlledStateConfig;
  children?: ReactNode;
  label?: string;
  description?: string;
  log?: (event: { kind: string; [key: string]: unknown }) => void;
};

/**
 * Generic controlled-state adapter. The prop names are explicit configuration
 * because the harness must not guess a component API. If no value prop is
 * configured, it remains a presentation shell and does not fabricate state.
 */
export function ControlledState({
  subject: Subject,
  args = {},
  config = {},
  children,
  label = "Controlled state",
  description = "A focused state with its inputs visible for inspection.",
  log,
}: PreviewHarnessProps) {
  const [value, setValue] = useState(config.initialValue);
  const subjectArgs = { ...args };
  if (config.valueProp) {
    subjectArgs[config.valueProp] = value;
  }
  if (config.changeProp) {
    subjectArgs[config.changeProp] = (next: unknown) => {
      setValue(next);
      log?.({
        kind: "controlled-change",
        prop: config.changeProp,
        value: next,
      });
    };
  }
  return (
    <PreviewShowcase
      subject={Subject}
      args={subjectArgs}
      config={config}
      family="controlled-state"
      label={label}
      description={description}
    >
      {children}
    </PreviewShowcase>
  );
}
