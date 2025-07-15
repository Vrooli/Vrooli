import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import type { BoxProps } from "@mui/material";
import { useTheme } from "@mui/material";
import { endpointsUser, type ProfileUpdateInput, type User } from "@vrooli/shared";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../../api/fetchWrapper.js";
import { useLazyFetch } from "../../../hooks/useFetch.js";
import { noSelect } from "../../../styles.js";
import { PubSub } from "../../../utils/pubsub.js";
import { Switch } from "../Switch/Switch.js";

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
    const handleChange = useCallback((checked: boolean) => {
        if (isUpdating) return;
        const newTheme = checked ? "dark" : "light";
        PubSub.get().publish("theme", newTheme);
        if (updateServer) {
            fetchLazyWrapper<ProfileUpdateInput, User>({
                fetch,
                inputs: { theme: newTheme },
            });
        }
    }, [fetch, isUpdating, updateServer]);

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
            <Switch
                checked={isDark}
                onChange={handleChange}
                offIcon={offIconInfo}
                onIcon={onIconInfo}
                size="md"
                labelPosition="none"
            />
        </Box>
    );
}
