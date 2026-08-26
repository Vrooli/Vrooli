import { useState } from "react";
import { Presence } from "./Presence";

export function TogglePresence({
  args,
}: StoryHarnessProps<{ present: boolean; children: string }>) {
  const [present, setPresent] = useState(args.present);
  return (
    <div className="space-y-space-sm">
      <button
        type="button"
        onClick={() => setPresent((current) => !current)}
        style={{
          appearance: "none",
          background: "var(--color-primary)",
          border: "1px solid var(--color-primary)",
          borderRadius: "999px",
          boxShadow:
            "0 8px 20px color-mix(in srgb, var(--color-primary) 24%, transparent)",
          color: "var(--color-primary-foreground)",
          cursor: "pointer",
          fontSize: "var(--text-label-size)",
          fontWeight: 650,
          padding: "10px 16px",
          transition: "transform 160ms ease, box-shadow 160ms ease",
        }}
      >
        Toggle presence
      </button>
      <Presence
        present={present}
        keepMounted
        initial={false}
        duration="instant"
      >
        {args.children}
      </Presence>
    </div>
  );
}

export function PresenceAnatomy(props: Parameters<typeof TogglePresence>[0]) { return <TogglePresence {...props} />; }
export function PresenceBoundary(props: Parameters<typeof TogglePresence>[0]) { return <TogglePresence {...props} />; }
