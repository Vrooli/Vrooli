import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useCallback, useState } from "react";
import { Loader2, Volume2, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";
/**
 * Persistent affordance shown when auto-TTS playback was rejected by the
 * browser's autoplay policy and no qualifying user gesture has unlocked the
 * audio element yet. Clicking "Enable voice" runs a silent play() from the
 * click's gesture stack, activating the media element for the rest of the
 * session.
 */
export default function EnableAudioBanner({ onEnable, onDismiss }) {
    const { t } = useTranslation();
    const [enabling, setEnabling] = useState(false);
    const handleEnable = useCallback(async () => {
        if (enabling)
            return;
        setEnabling(true);
        try {
            await onEnable();
        }
        finally {
            setEnabling(false);
        }
    }, [enabling, onEnable]);
    const handleDismiss = useCallback(() => {
        if (enabling)
            return;
        onDismiss();
    }, [enabling, onDismiss]);
    return (_jsxs("div", { "data-testid": "enable-audio-banner", "data-audio-state": "enable-audio", className: "wc-stable-theme flex items-start gap-2 border-b border-amber-500/30 bg-amber-500/10 py-2 ps-[max(0.75rem,var(--wc-safe-left,0px))] pe-[max(0.75rem,var(--wc-safe-right,0px))] text-xs text-amber-200", role: "status", children: [_jsx(Volume2, { className: "mt-0.5 h-3.5 w-3.5 shrink-0", "aria-hidden": true }), _jsxs("div", { className: "flex-1 min-w-0", children: [_jsx("div", { className: "font-medium", children: t(strings.enableAudioBanner.title) }), _jsx("div", { className: "mt-0.5 break-words text-amber-200/80", children: t(strings.enableAudioBanner.description) })] }), _jsxs("button", { type: "button", "data-testid": "enable-audio-banner-enable", onClick: handleEnable, disabled: enabling, className: "shrink-0 inline-flex items-center gap-1 rounded border border-amber-400/40 bg-amber-500/20 px-2 py-1 font-medium text-amber-100 transition active:bg-amber-500/30 disabled:cursor-not-allowed disabled:opacity-60", title: t(strings.enableAudioBanner.enableTitle), "aria-label": t(strings.enableAudioBanner.enable), children: [enabling ? (_jsx(Loader2, { className: "h-3.5 w-3.5 animate-spin", "aria-hidden": true })) : (_jsx(Volume2, { className: "h-3.5 w-3.5", "aria-hidden": true })), _jsx("span", { children: enabling ? t(strings.enableAudioBanner.enabling) : t(strings.enableAudioBanner.enable) })] }), _jsx("button", { type: "button", "data-testid": "enable-audio-banner-dismiss", onClick: handleDismiss, disabled: enabling, className: "shrink-0 rounded border border-wc-default bg-wc-surface-input p-1 text-wc-text-secondary transition active:bg-wc-accent-active disabled:cursor-not-allowed disabled:opacity-60", title: t(strings.enableAudioBanner.dismiss), "aria-label": t(strings.enableAudioBanner.dismiss), children: _jsx(X, { className: "h-3.5 w-3.5" }) })] }));
}
