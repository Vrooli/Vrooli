import { ProfileUpdateInput, User, endpointsUser } from "@local/shared";
import { Box, Typography, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api/fetchWrapper";
import { useLazyFetch } from "hooks/useLazyFetch";
import { DarkModeIcon, LightModeIcon } from "icons";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { noSelect } from "styles";
import { SxType } from "types";
import { PubSub } from "utils/pubsub";
import { ToggleSwitch } from "../ToggleSwitch/ToggleSwitch";

type ThemeSwitchProps = {
    updateServer: boolean;
    sx?: SxType;
};

export function ThemeSwitch({
    updateServer,
    sx,
}: ThemeSwitchProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [fetch, { loading: isUpdating }] = useLazyFetch<ProfileUpdateInput, User>(endpointsUser.profileUpdate);
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
        <Box
            sx={{
                display: "flex",
                flexDirection: "row",
                gap: 1,
                justifyContent: "center",
                alignItems: "center",
                ...sx,
            }}
        >
            <Typography variant="body1" sx={noSelect}>
                {palette.mode === "light" ? t("Light") : t("Dark")}
            </Typography>
            <ToggleSwitch
                checked={isDark}
                onChange={handleChange}
                OffIcon={LightModeIcon}
                OnIcon={DarkModeIcon}
            />
        </Box>
    );
}
