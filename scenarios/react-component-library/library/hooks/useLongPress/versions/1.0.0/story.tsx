import { useLongPress } from "./useLongPress";
export function UseLongPressStory({ className }: { className?: string }) { const { longPressProps } = useLongPress({ onLongPress: () => {} }); return <button className={className} type="button" {...longPressProps}>Press and hold</button>; }
