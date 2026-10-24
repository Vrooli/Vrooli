import { CardShell } from "./CardShell";

export function CardShellStory({ args }: StoryHarnessProps<{ children?: string }>) {
  const children = args.children ?? "Scenario-owned card content";
  return (
    <CardShell selection={{ selectionMode: true, selected: false }}>
      <span data-testid="card-shell-story-content">{children}</span>
    </CardShell>
  );
}
