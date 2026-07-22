/** Preview-only contract available to version-local story.tsx harnesses. */
interface StoryHarnessProps<TArgs extends Record<string, unknown> = Record<string, unknown>> {
  args: TArgs;
  log: (name: string, ...args: unknown[]) => void;
}
