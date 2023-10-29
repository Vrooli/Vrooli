import { Stack, Typography, useTheme } from "@mui/material";
import { DarkModeIcon, LightModeIcon } from "icons";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { noSelect } from "styles";
import { PubSub } from "utils/pubsub";
import { ToggleSwitch } from "../ToggleSwitch/ToggleSwitch";

export function ThemeSwitch() {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const handleChange = useCallback(() => {
        const newTheme = palette.mode === "light" ? "dark" : "light";
        PubSub.get().publishTheme(newTheme);
    }, [palette.mode]);

    const isDark = useMemo(() => palette.mode === "dark", [palette.mode]);

    return (
        <Stack direction="row" spacing={1} justifyContent="center" alignItems="center">
            <Typography variant="body1" sx={{
                ...noSelect,
                marginRight: "auto",
            }}>
                {palette.mode === "light" ? t("Light") : t("Dark")}
            </Typography>
            <ToggleSwitch
                checked={isDark}
                onChange={handleChange}
                OffIcon={LightModeIcon}
                OnIcon={DarkModeIcon}
            />
        </Stack>
    );
}
