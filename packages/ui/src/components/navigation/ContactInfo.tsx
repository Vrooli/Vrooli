import Box from "@mui/material/Box";
import Divider from "@mui/material/Divider";
import Link from "@mui/material/Link";
import { Tooltip } from "../Tooltip/Tooltip.js";
import Typography from "@mui/material/Typography";
import { styled } from "@mui/material";
import { LINKS, SOCIALS, type TranslationKeyCommon } from "@vrooli/shared";
import { useTranslation } from "react-i18next";
import { Icon, type IconInfo } from "../../icons/Icons.js";
import { noSelect } from "../../styles.js";

type ContactInfoItem = {
    tooltip: TranslationKeyCommon | "";
    label: string;
    link: string;
    iconInfo: IconInfo;
}

type AdditionalInfoItem = Omit<ContactInfoItem, "iconInfo">;

const contactInfo: ContactInfoItem[] = [
    { tooltip: "ContactHelpX", label: "X/Twitter", link: SOCIALS.X, iconInfo: { name: "X", type: "Service" } },
    { tooltip: "ContactHelpCode", label: "Code", link: SOCIALS.GitHub, iconInfo: { name: "GitHub", type: "Service" } },
];

const additionalInfo: AdditionalInfoItem[] = [
    { tooltip: "", label: "About", link: LINKS.About },
    { tooltip: "", label: "Stats", link: LINKS.Stats },
    { tooltip: "", label: "Privacy", link: LINKS.Privacy },
    { tooltip: "", label: "Terms", link: LINKS.Terms },
];

const LinkStyled = styled(Link)(({ theme }) => ({
    display: "flex",
    flexDirection: "row",
    justifyContent: "center",
    alignItems: "center",
    gap: theme.spacing(1),
    color: theme.palette.background.textPrimary,
    textDecoration: "none",
}));
const DividerStyled = styled(Divider)(({ theme }) => ({
    background: theme.palette.background.textSecondary,
    width: "100%",
    marginTop: theme.spacing(1),
}));
const SectionTitle = styled(Typography)(({ theme }) => ({
    color: theme.palette.background.textPrimary,
    paddingLeft: theme.spacing(2),
    paddingRight: theme.spacing(2),
    paddingTop: theme.spacing(1),
    paddingBottom: theme.spacing(1),
    ...noSelect,
}));
const SectionBox = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "row",
    gap: theme.spacing(2),
    justifyContent: "center",
    alignItems: "center",
    paddingLeft: theme.spacing(2),
    paddingRight: theme.spacing(2),
    paddingBottom: theme.spacing(1),
}));

export function ContactInfo() {
    const { t } = useTranslation();

    return (
        <Box
            display="flex"
            flexDirection="column"
            alignItems="flex-start"
        >
            <Box>
                <SectionTitle
                    variant="body1"
                    component="h6"
                >
                    {t("ConnectWithUs")}
                </SectionTitle>
                <SectionBox>
                    {contactInfo.map(({ tooltip, label, link, iconInfo }) => (
                        <Tooltip key={`contact-info-button-${label}`} title={tooltip.length > 0 ? t(tooltip) : ""} placement="top">
                            <LinkStyled href={link}>
                                <Icon fill="background.textPrimary" info={iconInfo} size={20} />
                                <Typography variant="caption">
                                    {label}
                                </Typography>
                            </LinkStyled>
                        </Tooltip>
                    ))}
                </SectionBox>
            </Box>
            <DividerStyled />
            <Box>
                <SectionTitle
                    variant="body1"
                    component="h6"
                >
                    {t("AdditionalResources")}
                </SectionTitle>
                <SectionBox>
                    {additionalInfo.map(({ tooltip, label, link }) => (
                        <Tooltip key={`additional-info-button-${label}`} title={tooltip.length > 0 ? t(tooltip) : ""} placement="top">
                            <LinkStyled href={link}>
                                <Typography variant="caption">
                                    {label}
                                </Typography>
                            </LinkStyled>
                        </Tooltip>
                    ))}
                </SectionBox>
            </Box>
        </Box>
    );
}
