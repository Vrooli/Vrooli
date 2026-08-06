import { useState } from "react";
import { useEscapeKey } from "./useEscapeKey";

export function Default({ args }: StoryHarnessProps<{ active: boolean }>) {
  const [escaped, setEscaped] = useState(false);
  useEscapeKey(args.active, () => setEscaped(true));
  return <div role="status">{escaped ? "escaped" : "waiting"}</div>;
}
