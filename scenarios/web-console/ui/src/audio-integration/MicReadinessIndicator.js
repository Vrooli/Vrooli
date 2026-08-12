import { jsx as _jsx } from "react/jsx-runtime";
const DEFAULT_LABELS = {
    granted: "Microphone ready",
    denied: "Microphone denied",
    prompt: "Microphone permission required",
    unknown: "Microphone status unknown",
};
export function MicReadinessIndicator(props) {
    const label = props.labels?.[props.state] ?? DEFAULT_LABELS[props.state];
    return (_jsx("span", { role: "status", "aria-live": "polite", "data-state": props.state, className: "audio-tools-embed-mic-readiness", children: label }));
}
