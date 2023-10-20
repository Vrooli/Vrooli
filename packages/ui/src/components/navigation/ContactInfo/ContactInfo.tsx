import { LINKS, SOCIALS } from "@local/shared";
import { BottomNavigation, BottomNavigationAction, Box, IconButton, Tooltip, Typography, useTheme } from "@mui/material";
import { CopyrightBreadcrumbs } from "components/breadcrumbs/CopyrightBreadcrumbs/CopyrightBreadcrumbs";
import { ArticleIcon, GitHubIcon, InfoIcon, StatsIcon, TwitterIcon } from "icons";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { openLink, useLocation } from "route";
import { noSelect } from "styles";
import { SvgComponent } from "types";
import { ContactInfoProps } from "../types";

type NavActionListData = [string, string, string, SvgComponent]

export const ContactInfo = ({
    sx,
    ...props
}: ContactInfoProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { additionalInfo, contactInfo } = useMemo(() => {
        return {
            additionalInfo: [
                [t("AboutUs"), t("About"), LINKS.About, InfoIcon],
                [t("DocumentationShort"), t("DocumentationShort"), "https://docs.vrooli.com", ArticleIcon],
                [t("StatisticsShort"), t("StatisticsShort"), LINKS.Stats, StatsIcon],
            ] as NavActionListData[],
            contactInfo: [
                [t("ContactHelpTwitter"), t("Twitter"), SOCIALS.Twitter, TwitterIcon],
                // [t("ContactHelpDiscord"), t("Discord"), SOCIALS.Discord, DiscordIcon],
                [t("ContactHelpCode"), t("Code"), SOCIALS.GitHub, GitHubIcon],
            ] as NavActionListData[],
        };
    }, [t]);

    const handleLink = (e: React.MouseEvent<any>, link: string) => {
        e.preventDefault();
        openLink(setLocation, link);
    };

    return (
        <Box sx={{
            minWidth: "fit-content",
            height: "fit-content",
            background: palette.background.default,
            padding: 1,
            ...(sx ?? {}),
        }} {...props}>
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
                {contactInfo.map(([tooltip, label, link, Icon], index: number) => (
                    <Tooltip key={`contact-info-button-${index}`} title={tooltip} placement="top">
                        <BottomNavigationAction
                            label={label}
                            onClick={(e) => { e.preventDefault(); handleLink(e, link); }}
                            href={link}
                            icon={
                                <IconButton sx={{ background: palette.secondary.main }} >
                                    <Icon fill={palette.secondary.contrastText} />
                                </IconButton>
                            }
                            sx={{
                                alignItems: "center",
                                color: palette.background.textPrimary,
                                overflowWrap: "anywhere",
                            }}
                        />
                    </Tooltip>
                ))}
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
                {additionalInfo.map(([tooltip, label, link, Icon], index: number) => (
                    <Tooltip key={`additional-info-button-${index}`} title={tooltip} placement="top">
                        <BottomNavigationAction
                            label={label}
                            onClick={(e) => { e.preventDefault(); handleLink(e, link); }}
                            href={link}
                            icon={
                                <IconButton sx={{ background: palette.secondary.main }}>
                                    <Icon fill={palette.secondary.contrastText} />
                                </IconButton>
                            }
                            sx={{
                                alignItems: "center",
                                color: palette.background.textPrimary,
                                overflowWrap: "anywhere",
                            }}
                        />
                    </Tooltip>
                ))}
            </BottomNavigation>
            <CopyrightBreadcrumbs sx={{ color: palette.background.textPrimary }} />
        </Box>
    );
};
