import { Stack } from "@mui/material";
import { useTranslation } from "react-i18next";
import { pagePaddingBottom } from "styles";
import { TopBar } from "../TopBar/TopBar";
import { SettingsTopBarProps } from "../types";

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
            titleBehaviorDesktop="ShowIn"
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
