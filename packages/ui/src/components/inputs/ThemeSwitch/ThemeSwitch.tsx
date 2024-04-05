import { ProfileUpdateInput, User, endpointPutProfile } from "@local/shared";
import { Stack, Typography, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { useLazyFetch } from "hooks/useLazyFetch";
import { DarkModeIcon, LightModeIcon } from "icons";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { noSelect } from "styles";
import { PubSub } from "utils/pubsub";
import { ToggleSwitch } from "../ToggleSwitch/ToggleSwitch";

type ThemeSwitchProps = {
    updateServer: boolean;
};

export function ThemeSwitch({
    updateServer,
}: ThemeSwitchProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [fetch, { loading: isUpdating }] = useLazyFetch<ProfileUpdateInput, User>(endpointPutProfile);
    const handleChange = useCallback(() => {
        if (isUpdating) return;
        const newTheme = palette.mode === "light" ? "dark" : "light";
        PubSub.get().publish("theme", newTheme);
        if (updateServer) {
            fetchLazyWrapper<ProfileUpdateInput, User>({
                fetch,
                inputs: { theme: newTheme },
            });
        }
    }, [fetch, isUpdating, palette.mode, updateServer]);

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
