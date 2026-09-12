/** Preview-only contract available to version-local story.tsx harnesses. */
interface PreviewEnvironment {
  [key: string]: unknown;
}

interface PreviewFixture {
  key?: string;
  adapter?: string;
  value?: unknown;
}

interface PreviewEvent {
  kind: string;
  [key: string]: unknown;
}

interface StoryHarnessProps<TArgs extends Record<string, unknown> = Record<string, unknown>> {
  args: TArgs;
  log: (name: string, ...args: unknown[]) => void;
  environment?: PreviewEnvironment;
  fixtures?: Record<string, PreviewFixture>;
}

/** Typed contract for versioned injected-subject composition harnesses. */
interface CompositionHarnessProps<
  TArgs extends Record<string, unknown> = Record<string, unknown>,
  TConfig extends Record<string, unknown> = Record<string, unknown>,
> {
  subject: React.ComponentType<TArgs>;
  args?: TArgs;
  config?: TConfig;
  environment?: PreviewEnvironment;
  fixtures?: Record<string, PreviewFixture>;
  log?: (event: PreviewEvent) => void;
  children?: React.ReactNode;
}
