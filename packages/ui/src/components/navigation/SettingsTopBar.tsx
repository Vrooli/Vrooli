import { Stack } from "@mui/material";
import { useTranslation } from "react-i18next";
import { pagePaddingBottom } from "../../styles.js";
import { TopBar } from "./TopBar.js";
import { SettingsTopBarProps } from "./types.js";

export function SettingsTopBar({
    display,
    onClose,
    ...rest
}: SettingsTopBarProps) {
    const { t } = useTranslation();

    return (
        <TopBar
            {...rest}
            display={display}
            onClose={onClose}
            title={t("Settings")}
        />
    );
}

const settingsContentStackStyle = { paddingBottom: pagePaddingBottom } as const;

export function SettingsContent({ children }: { children: React.ReactNode }) {
    return (
        <Stack id="settings-page" direction="row" mt={2} sx={settingsContentStackStyle}>
            {children}
        </Stack>
    );
}
