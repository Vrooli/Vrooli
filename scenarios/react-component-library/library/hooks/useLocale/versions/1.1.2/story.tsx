import { useLocale } from "./useLocale";
export function Default() {
  return (
    <div data-testid="hooks.use-locale" role="status">
      {useLocale()}
    </div>
  );
}
