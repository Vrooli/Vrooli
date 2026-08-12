import { jsx as _jsx } from "react/jsx-runtime";
import { Check, Pipette, Plus, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import ColorPicker from "./ColorPicker";
import { HEADER_COLORS } from "../../consts/config";
import { strings } from "../../consts/strings";
import { parsePaneColor, serializePaneColor } from "../../lib/paneColor";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
/**
 * Scenario seam for the adopted ColorPicker. The library owns presentation,
 * controlled input timing, and accessibility; web-console owns workspace
 * recents, translations, palette selection, and its pane-color encoding.
 */
export default function HeaderColorPicker({ currentColor, onSelectColor, testIdPrefix = "appearance" }) {
    const { t } = useTranslation();
    const recentColors = useWorkspaceStore((state) => state.recentHeaderColors) ?? [];
    const recordRecent = useWorkspaceStore((state) => state.addRecentHeaderColor);
    const normalizedValue = serializePaneColor(parsePaneColor(currentColor).colors);
    return _jsx(ColorPicker, { palette: HEADER_COLORS, value: normalizedValue, onChange: (next) => onSelectColor(serializePaneColor(parsePaneColor(next).colors)), recentColors: recentColors, onRecordRecent: recordRecent, allowGradient: true, testIdPrefix: `${testIdPrefix}-header-color`, icons: { check: Check, custom: Pipette, add: Plus, remove: X }, labels: {
            heading: t(strings.appearance.headerColorHeading),
            transparent: t(strings.appearance.noColorTitle),
            custom: t(strings.appearance.customColorTitle),
            recents: t(strings.appearance.recentColorsHeading),
            addGradient: t(strings.appearance.gradientLabel),
            removeGradient: t(strings.appearance.removeSecondaryColor),
        } });
}
