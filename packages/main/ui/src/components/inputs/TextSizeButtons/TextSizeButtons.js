import { jsxs as _jsxs, jsx as _jsx } from "react/jsx-runtime";
import { BumpModerateIcon } from "@local/icons";
import { Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { noSelect } from "../../../styles";
import { getCookieFontSize } from "../../../utils/cookies";
import { PubSub } from "../../../utils/pubsub";
import { ColorIconButton } from "../../buttons/ColorIconButton/ColorIconButton";
const smallestFontSize = 10;
const largestFontSize = 20;
export function TextSizeButtons() {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [size, setSize] = useState(getCookieFontSize(14));
    const handleShrink = useCallback(() => {
        if (size > smallestFontSize) {
            setSize(size - 1);
            PubSub.get().publishFontSize(size - 1);
        }
    }, [size]);
    const handleGrow = useCallback(() => {
        if (size < largestFontSize) {
            setSize(size + 1);
            PubSub.get().publishFontSize(size + 1);
        }
    }, [size]);
    return (_jsxs(Stack, { direction: "row", spacing: 1, justifyContent: "center", alignItems: "center", children: [_jsxs(Typography, { variant: "body1", sx: {
                    ...noSelect,
                    marginRight: "auto",
                }, children: [t("TextSize"), ": ", size] }), _jsxs(Stack, { direction: "row", spacing: 0, sx: { paddingTop: 1, paddingBottom: 1 }, children: [_jsx(Tooltip, { placement: "top", title: "Smaller", children: _jsx(ColorIconButton, { "aria-label": 'shrink text', background: palette.secondary.main, onClick: handleShrink, sx: {
                                borderRadius: "12px 0 0 12px",
                                borderRight: `1px solid ${palette.secondary.contrastText}`,
                                height: "48px",
                            }, children: _jsx(BumpModerateIcon, { style: { transform: "rotate(180deg)" } }) }) }), _jsx(Tooltip, { placement: "top", title: "Larger", children: _jsx(ColorIconButton, { "aria-label": 'grow text', background: palette.secondary.main, onClick: handleGrow, sx: {
                                borderRadius: "0 12px 12px 0",
                                height: "48px",
                            }, children: _jsx(BumpModerateIcon, {}) }) })] })] }));
}
//# sourceMappingURL=TextSizeButtons.js.map