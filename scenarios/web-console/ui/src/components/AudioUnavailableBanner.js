import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { AlertCircle } from "lucide-react";
const REASON_MESSAGES = {
    discovery_failed: "Could not reach the audio-tools service. Voice input and synthesis are disabled.",
    scenario_not_running: "The audio-tools scenario is not running. Start it to enable voice features.",
    env_misconfigured: "audio-tools URL is not configured. Set AUDIO_TOOLS_URL or start the audio-tools scenario.",
    resolver_not_configured: "Audio-tools discovery is not wired in this build of web-console.",
};
export function AudioUnavailableBanner({ reason, className }) {
    if (!reason)
        return null;
    const message = REASON_MESSAGES[reason] ?? `Audio-tools is unavailable (${reason}).`;
    return (_jsxs("div", { role: "status", "aria-live": "polite", "data-audio-state": "unavailable", className: [
            "flex items-start gap-2 rounded-control border border-app-warning/40 bg-app-warning/10 px-3 py-2 text-sm text-app-warning-foreground",
            className ?? "",
        ].join(" ").trim(), children: [_jsx(AlertCircle, { className: "mt-0.5 h-4 w-4 flex-shrink-0", "aria-hidden": "true" }), _jsx("span", { children: message })] }));
}
