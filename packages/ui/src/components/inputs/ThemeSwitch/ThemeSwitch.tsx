import { ProfileUpdateInput, User, endpointsUser } from "@local/shared";
import { Box, BoxProps, Typography, useTheme } from "@mui/material";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../../api/fetchWrapper.js";
import { useLazyFetch } from "../../../hooks/useLazyFetch.js";
import { noSelect } from "../../../styles.js";
import { PubSub } from "../../../utils/pubsub.js";
import { ToggleSwitch } from "../ToggleSwitch/ToggleSwitch.js";

const offIconInfo = { name: "LightMode", type: "Common" } as const;
const onIconInfo = { name: "DarkMode", type: "Common" } as const;

type ThemeSwitchProps = BoxProps & {
    updateServer: boolean;
};

export function ThemeSwitch({
    updateServer,
    ...props
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
            display="flex"
            flexDirection="row"
            gap={1}
            justifyContent="center"
            alignItems="center"
            {...props}
        >
            <Typography variant="body1" sx={noSelect}>
                {palette.mode === "light" ? t("Light") : t("Dark")}
            </Typography>
            <ToggleSwitch
                checked={isDark}
                onChange={handleChange}
                offIconInfo={offIconInfo}
                onIconInfo={onIconInfo}
            />
        </Box>
    );
}
