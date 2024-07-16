import { LINKS } from "@local/shared";
import { IconButton, useTheme } from "@mui/material";
import { ArrowLeftIcon } from "icons";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { TopBar } from "../TopBar/TopBar";
import { SettingsTopBarProps } from "../types";

/**
 * Generates an app bar for both pages and dialogs
 */
export function SettingsTopBar({
    display,
    onClose,
    ...rest
}: SettingsTopBarProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();


    return (
        <TopBar
            {...rest}
            display={display}
            onClose={onClose}
            startComponent={<IconButton
                aria-label="Back"
                onClick={() => {
                    const hasPreviousPage = Boolean(sessionStorage.getItem("lastPath"));
                    if (hasPreviousPage) window.history.back();
                    else setLocation(LINKS.Settings, { replace: true });
                }}
                sx={{
                    width: "48px",
                    height: "48px",
                    marginLeft: 1,
                    marginRight: 1,
                    cursor: "pointer",
                }}
            >
                <ArrowLeftIcon fill={palette.primary.contrastText} width="100%" height="100%" />
            </IconButton>}
            title={t("Settings")}
            titleBehaviorDesktop="ShowIn"
        />
    );
}
