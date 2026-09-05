import { useTypeahead } from "./useTypeahead";
export function Default({ log }: StoryHarnessProps) {
  return (
    <input data-testid="hooks.use-typeahead"
      aria-label="Typeahead"
      onKeyDown={useTypeahead((query) => log("match", query))}
    />
  );
}
