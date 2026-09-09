import { useMediaQuery } from "./useMediaQuery";
export function Default() {
  return (
    <div data-testid="hooks.use-media-query" role="status">
      {useMediaQuery("(min-width: 0px)") ? "matches" : "does-not-match"}
    </div>
  );
}
