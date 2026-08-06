import { useAnnounce } from "./useAnnounce";
export function Default({ log }: StoryHarnessProps) { const announce = useAnnounce(); return <button type="button" onClick={() => { announce("Updated"); log("announced"); }}>Announce</button>; }
