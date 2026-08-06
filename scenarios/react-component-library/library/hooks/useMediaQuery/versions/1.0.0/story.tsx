import { useMediaQuery } from "./useMediaQuery";
export function Default() {
  return (
    <div role="status">
      {useMediaQuery("(min-width: 0px)") ? "matches" : "does-not-match"}
    </div>
  );
}
