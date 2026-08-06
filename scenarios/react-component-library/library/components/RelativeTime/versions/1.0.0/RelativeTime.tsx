/** @vrooliComponentSource materialized.relativetime */
const muted = { color: "var(--color-muted-foreground, #64748b)" };
export function RelativeTime({ value = "just now" }: { value?: string }) { return <time dateTime={value} style={muted}>{value}</time>; }
