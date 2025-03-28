import { LINKS, SOCIALS } from "@local/shared";
import { BottomNavigation, BottomNavigationAction, Box, IconButton, Tooltip, Typography, useTheme } from "@mui/material";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Icon, IconInfo } from "../../icons/Icons.js";
import { openLink } from "../../route/openLink.js";
import { useLocation } from "../../route/router.js";
import { noSelect } from "../../styles.js";
import { CopyrightBreadcrumbs } from "../breadcrumbs/CopyrightBreadcrumbs.js";

type NavActionListData = [string, string, string, IconInfo]

export function ContactInfo() {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { additionalInfo, contactInfo } = useMemo(() => {
        return {
            additionalInfo: [
                [t("AboutUs"), t("About"), LINKS.About, { name: "Info", type: "Common" }],
                [t("DocumentationShort"), t("DocumentationShort"), "https://docs.vrooli.com", { name: "Article", type: "Common" }],
                [t("StatisticsShort"), t("StatisticsShort"), LINKS.Stats, { name: "Stats", type: "Common" }],
            ] as NavActionListData[],
            contactInfo: [
                [t("ContactHelpX"), t("X"), SOCIALS.X, { name: "X", type: "Service" }],
                // [t("ContactHelpDiscord"), t("Discord"), SOCIALS.Discord, DiscordIcon],
                [t("ContactHelpCode"), t("Code"), SOCIALS.GitHub, { name: "GitHub", type: "Service" }],
            ] as NavActionListData[],
        };
    }, [t]);

    function handleLink(e: React.MouseEvent<any>, link: string) {
        e.preventDefault();
        openLink(setLocation, link);
    }

    return (
        <Box minWidth="fit-content" height="fit-content" padding={1}>
            <Typography variant="h6" textAlign="center" color={palette.background.textPrimary} sx={{ ...noSelect }}>{t("FindUsOn")}</Typography>
            <BottomNavigation
                showLabels
                sx={{
                    alignItems: "baseline",
                    background: "transparent",
                    height: "fit-content",
                    padding: 1,
                    marginBottom: 2,
                }}>
                {contactInfo.map(([tooltip, label, link, iconInfo], index: number) => {
                    function handleClick(event: React.MouseEvent<HTMLElement>) {
                        event.preventDefault();
                        handleLink(event, link);
                    }

                    return (
                        <Tooltip key={`contact-info-button-${index}`} title={tooltip} placement="top">
                            <BottomNavigationAction
                                label={label}
                                onClick={handleClick}
                                href={link}
                                icon={
                                    <IconButton sx={{ background: palette.secondary.main }} >
                                        <Icon
                                            decorative
                                            fill={palette.secondary.contrastText}
                                            info={iconInfo}
                                        />
                                    </IconButton>
                                }
                                sx={{
                                    alignItems: "center",
                                    color: palette.background.textPrimary,
                                    overflowWrap: "anywhere",
                                }}
                            />
                        </Tooltip>
                    );
                })}
            </BottomNavigation>
            <Typography variant="h6" textAlign="center" color={palette.background.textPrimary} sx={{ ...noSelect }}>{t("AdditionalResources")}</Typography>
            <BottomNavigation
                showLabels
                sx={{
                    alignItems: "baseline",
                    background: "transparent",
                    height: "fit-content",
                    padding: 1,
                }}>
                {additionalInfo.map(([tooltip, label, link, iconInfo], index: number) => {
                    function handleClick(event: React.MouseEvent<HTMLElement>) {
                        event.preventDefault();
                        handleLink(event, link);
                    }

                    return (
                        <Tooltip key={`additional-info-button-${index}`} title={tooltip} placement="top">
                            <BottomNavigationAction
                                label={label}
                                onClick={handleClick}
                                href={link}
                                icon={
                                    <IconButton sx={{ background: palette.secondary.main }}>
                                        <Icon
                                            decorative
                                            fill={palette.secondary.contrastText}
                                            info={iconInfo}
                                        />
                                    </IconButton>
                                }
                                sx={{
                                    alignItems: "center",
                                    color: palette.background.textPrimary,
                                    overflowWrap: "anywhere",
                                }}
                            />
                        </Tooltip>
                    );
                })}
            </BottomNavigation>
            <CopyrightBreadcrumbs sx={{ color: palette.background.textPrimary }} />
        </Box>
    );
}
