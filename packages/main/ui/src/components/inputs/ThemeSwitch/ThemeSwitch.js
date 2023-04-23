import { jsxs as _jsxs, jsx as _jsx } from "react/jsx-runtime";
import { DarkModeIcon, LightModeIcon } from "@local/icons";
import { Stack, Typography, useTheme } from "@mui/material";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { noSelect } from "../../../styles";
import { PubSub } from "../../../utils/pubsub";
import { ToggleSwitch } from "../ToggleSwitch/ToggleSwitch";
export function ThemeSwitch() {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const handleChange = useCallback(() => {
        const newTheme = palette.mode === "light" ? "dark" : "light";
        PubSub.get().publishTheme(newTheme);
    }, [palette.mode]);
    const isDark = useMemo(() => palette.mode === "dark", [palette.mode]);
    const Icon = useMemo(() => isDark ? DarkModeIcon : LightModeIcon, [isDark]);
    const trackColor = useMemo(() => isDark ? "#2F3A45" : "#BFC7CF", [isDark]);
    return (_jsxs(Stack, { direction: "row", spacing: 1, justifyContent: "center", alignItems: "center", children: [_jsxs(Typography, { variant: "body1", sx: {
                    ...noSelect,
                    marginRight: "auto",
                }, children: [t("Theme"), ": ", palette.mode === "light" ? t("Light") : t("Dark")] }), _jsx(ToggleSwitch, { checked: isDark, onChange: handleChange, OffIcon: LightModeIcon, OnIcon: DarkModeIcon })] }));
}
//# sourceMappingURL=ThemeSwitch.js.map