import { useTypeahead } from "./useTypeahead";
export function Default({ log }: StoryHarnessProps) {
  return (
    <input
      aria-label="Typeahead"
      onKeyDown={useTypeahead((query) => log("match", query))}
    />
  );
}
