import { useLocale } from "./useLocale";
export function Default() { return <div role="status">{useLocale()}</div>; }
