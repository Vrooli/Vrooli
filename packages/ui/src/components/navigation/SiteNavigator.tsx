import { API_CREDITS_MULTIPLIER, LINKS } from "@local/shared";
import { Box, Button, Collapse, Divider, IconButton, List, ListItem, ListItemIcon, ListItemText, Typography, styled, useTheme } from "@mui/material";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { SessionContext } from "../../contexts/session.js";
import { useMenu } from "../../hooks/useMenu.js";
import { useWindowSize } from "../../hooks/useWindowSize.js";
import { Icon, IconCommon, IconInfo, IconText } from "../../icons/Icons.js";
import { useLocation } from "../../route/router.js";
import { ProfileAvatar, ScrollBox } from "../../styles.js";
import { checkIfLoggedIn, getCurrentUser } from "../../utils/authentication/session.js";
import { ELEMENT_IDS } from "../../utils/consts.js";
import { extractImageUrl } from "../../utils/display/imageTools.js";
import { placeholderColor } from "../../utils/display/listTools.js";
import { NAV_ACTION_TAGS, getUserActions } from "../../utils/navigation/userActions.js";
import { MenuPayloads, PubSub } from "../../utils/pubsub.js";
import { PageContainer } from "../Page/Page.js";

/** Threshold for showing low credit balance */
// eslint-disable-next-line no-magic-numbers
const LOW_CREDIT_THRESHOLD = BigInt(5_00) * API_CREDITS_MULTIPLIER;
const AVATAR_SIZE_PX = 50;

const projectItemStyles = {
    selected: { pl: 4, bgcolor: "#3B82F6" },
    default: { pl: 4, bgcolor: "transparent" },
};

const addProjectStyles = { pl: 4 };
const dividerStyles = { my: 1, bgcolor: "#333333" };
const yesterdayLabelStyles = { color: "#9CA3AF" };
const yesterdayItemStyles = {
    noWrap: true,
    sx: { color: "#9CA3AF" },
};
const viewPlansStyles = { color: "#9CA3AF", cursor: "pointer" };

const StyledToolbar = styled(Box)(({ theme }) => ({
    display: "flex",
    alignItems: "center",
    flexDirection: "row",
    justifyContent: "space-between",
    paddingTop: theme.spacing(1.5),
    paddingLeft: theme.spacing(3),
    paddingRight: theme.spacing(1),
}));
const StyledList = styled(List)(() => ({
    minHeight: "100%",
}));
const FooterContainer = styled(Box)(({ theme }) => ({
    alignItems: "center",
    borderTop: `1px solid ${theme.palette.divider}`,
    bottom: 0,
    display: "flex",
    justifyContent: "space-between",
    marginTop: "auto",
    padding: "16px",
    position: "absolute",
    width: "100%",
}));
const CreditStack = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "column",
    alignItems: "flex-start",
    justifyContent: "center",
    gap: theme.spacing(0.5),
}));
const PremiumBox = styled(Box)(({ theme }) => ({
    backgroundColor: `${theme.palette.secondary.main}15`,
    color: theme.palette.secondary.main,
    borderRadius: 1,
    paddingLeft: 1,
    paddingRight: 1,
    paddingTop: 0.25,
    paddingBottom: 0.25,
    display: "flex",
    alignItems: "center",
    gap: 0.5,
    boxShadow: `0 0 0 1px ${theme.palette.secondary.main}40`,
}));
const PremiumTypography = styled(Typography)({
    fontWeight: 500,
    fontSize: "0.75rem",
    lineHeight: 1,
});
const CreditTypography = styled(Typography)(({ theme }) => ({
    color: theme.palette.background.textSecondary,
    fontSize: "0.75rem",
}));

const projects = [
    { text: "Local Vrooli" },
    { text: "Routine graphs", selected: true },
];

const yesterdayItems = [
    "Generate Yup Schema",
    "Align Story at Bottom",
    "MSW unhandled request byp...",
    "Previous 7 Days",
    "Command Palette Search Res...",
];

// Separate component for navigation items
function NavItem({
    iconInfo,
    label,
    link,
    onClick,
}: {
    iconInfo: IconInfo;
    label: string;
    link: string;
    onClick: (url: string) => void;
}) {
    const handleClick = useCallback(() => {
        onClick(link);
    }, [onClick, link]);

    return (
        <ListItem button onClick={handleClick}>
            <ListItemIcon>
                <Icon decorative info={iconInfo} />
            </ListItemIcon>
            <ListItemText primary={label} />
        </ListItem>
    );
}

export function SiteNavigator() {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    // Handler for navigation items
    const handleNavClick = useCallback((url: string) => {
        setLocation(url);
    }, [setLocation]);

    // Get navigation actions as list items
    const navItems = useMemo(() => {
        const actions = getUserActions({ session, exclude: [NAV_ACTION_TAGS.Inbox, NAV_ACTION_TAGS.Pricing, NAV_ACTION_TAGS.LogIn, NAV_ACTION_TAGS.About] });
        return actions.map((action, index) => (
            <NavItem
                key={index}
                iconInfo={action.iconInfo}
                label={action.label}
                link={action.link}
                onClick={handleNavClick}
            />
        ));
    }, [session, handleNavClick]);

    const user = useMemo(() => getCurrentUser(session), [session]);
    const { credits, hasPremium } = user;
    const isLoggedIn = checkIfLoggedIn(session);

    const { creditsAsDollars, showLowCreditBalance } = useMemo(() => {
        let creditsAsDollars = "0";
        let showLowCreditBalance = false;
        try {
            const creditsAsBigInt = credits ? BigInt(credits) : BigInt(0);
            creditsAsDollars = (Number(creditsAsBigInt / API_CREDITS_MULTIPLIER) / 100).toFixed(2);
            showLowCreditBalance = creditsAsBigInt < LOW_CREDIT_THRESHOLD;
        } catch (error) {
            console.error("Error calculating credits", error);
        }
        return { creditsAsDollars, showLowCreditBalance };
    }, [credits]);

    const handleOpenSearch = useCallback(function handleOpenSearchCallback() {
        PubSub.get().publish("menu", { id: ELEMENT_IDS.CommandPalette, isOpen: true });
    }, []);

    // Handle opening and closing
    const onEvent = useCallback(function onEventCallback({ data }: MenuPayloads[typeof ELEMENT_IDS.LeftDrawer]) {
        if (!data) return;
        // Add logic here when data is added to the payload type
    }, []);
    const { close: handleClose } = useMenu({
        id: ELEMENT_IDS.LeftDrawer,
        isMobile,
        onEvent,
    });

    const [projectsOpen, setProjectsOpen] = useState(true);

    function handleProjectsClick() {
        setProjectsOpen(!projectsOpen);
    }

    // Handlers for Projects buttons
    const handleAddProject = useCallback(() => {
        // Add logic for adding a new project
    }, []);

    const handleSeeMoreProjects = useCallback(() => {
        // Add logic for showing more projects
    }, []);

    // Open side menu when profile icon is clicked
    const openUserMenu = useCallback(() => {
        PubSub.get().publish("menu", { id: ELEMENT_IDS.UserMenu, isOpen: true });
    }, []);

    // Always show footer on mobile, or when user has premium or low credit balance
    const showFooter = isMobile || (isLoggedIn && (!hasPremium || showLowCreditBalance));
    const profileColors = placeholderColor(user.id);

    function toLogin() {
        setLocation(LINKS.Login);
    }
    function toBuyCredits() {
        setLocation(LINKS.Pro);
    }

    // Update the projectItemStyle function
    function getProjectItemStyle(selected: boolean) {
        return selected ? projectItemStyles.selected : projectItemStyles.default;
    }

    return (
        <PageContainer contentType="text" size="fullSize">
            <ScrollBox>
                {/* Header */}
                <StyledToolbar>
                    <IconButton
                        aria-label={t("Menu")}
                        color="inherit"
                        edge="start"
                        onClick={handleClose}
                    >
                        <IconText
                            decorative
                            fill="background.textSecondary"
                            name="List"
                        />
                    </IconButton>
                    <Box ml="auto">
                        <IconButton
                            aria-label={t("Search")}
                            color="inherit"
                            onClick={handleOpenSearch}
                        >
                            <IconCommon
                                decorative
                                fill="background.textSecondary"
                                name="Search"
                            />
                        </IconButton>
                    </Box>
                </StyledToolbar>

                {/* Navigation List */}
                <StyledList>
                    {/* Only show nav items on desktop */}
                    {!isMobile && navItems}

                    {/* Projects Section */}
                    <ListItem
                        aria-label={t("Project", { count: 2 })}
                        button
                        onClick={handleProjectsClick}
                    >
                        <ListItemIcon>
                            <IconCommon
                                decorative
                                fill="background.textSecondary"
                                name="Project"
                            />
                        </ListItemIcon>
                        <ListItemText primary={t("Project", { count: 2 })} />
                        {
                            projectsOpen
                                ? <IconCommon
                                    decorative
                                    fill="background.textSecondary"
                                    name="ExpandLess"
                                />
                                : <IconCommon
                                    decorative
                                    fill="background.textSecondary"
                                    name="ExpandMore" />
                        }
                    </ListItem>
                    <Collapse in={projectsOpen} timeout="auto" unmountOnExit>
                        <List component="div" disablePadding>
                            {projects.map((project, index) => (
                                <ListItem
                                    button
                                    key={index}
                                    selected={project.selected}
                                    sx={getProjectItemStyle(project.selected || false)}
                                >
                                    <ListItemText primary={project.text} />
                                </ListItem>
                            ))}
                            {/* Add and See More buttons for Projects */}
                            <ListItem
                                aria-label={t("AddProject")}
                                button onClick={handleAddProject}
                                sx={addProjectStyles}
                            >
                                <ListItemIcon>
                                    <IconCommon
                                        decorative
                                        fill="background.textSecondary"
                                        name="Plus"
                                    />
                                </ListItemIcon>
                                <ListItemText primary={t("AddProject")} />
                            </ListItem>
                            <ListItem
                                aria-label={t("SeeAll")}
                                button
                                onClick={handleSeeMoreProjects}
                                sx={addProjectStyles}
                            >
                                <ListItemText primary={t("SeeAll")} />
                            </ListItem>
                        </List>
                    </Collapse>

                    {/* Yesterday Section */}
                    <Divider sx={dividerStyles} />
                    <ListItem>
                        <ListItemText primary="Yesterday" sx={yesterdayLabelStyles} />
                    </ListItem>
                    {yesterdayItems.map((item, index) => (
                        <ListItem button key={index}>
                            <ListItemText
                                primary={item}
                                primaryTypographyProps={yesterdayItemStyles}
                            />
                        </ListItem>
                    ))}
                </StyledList>
                {showFooter && (
                    <FooterContainer>
                        <Box display="flex" gap={1}>
                            {isMobile && (
                                <ProfileAvatar
                                    id={ELEMENT_IDS.UserMenuProfileIcon}
                                    isBot={false}
                                    src={isLoggedIn ? extractImageUrl(user.profileImage, user.updated_at, AVATAR_SIZE_PX) : undefined}
                                    onClick={openUserMenu}
                                    profileColors={profileColors}
                                >
                                    <IconCommon
                                        decorative
                                        name="Profile"
                                    />
                                </ProfileAvatar>
                            )}
                            <CreditStack>
                                {user.hasPremium && (
                                    <PremiumBox>
                                        <IconCommon
                                            decorative
                                            fill="background.textSecondary"
                                            name="Premium"
                                            size={14}
                                        />
                                        <PremiumTypography variant="body2">
                                            {t("Pro")}
                                        </PremiumTypography>
                                    </PremiumBox>
                                )}
                                <CreditTypography variant="body2">
                                    {t("Credit", { count: 2 })}: ${creditsAsDollars}
                                </CreditTypography>
                            </CreditStack>
                        </Box>
                        {/* Show login prompt when not logged in */}
                        {isMobile && !isLoggedIn && (
                            <Typography variant="body2" sx={viewPlansStyles} onClick={toLogin}>
                                {t("LogInToAccessAllFeatures")}
                            </Typography>
                        )}
                        {/* Show "Buy Credits" button when logged in and low credit balance */}
                        {isLoggedIn && showLowCreditBalance && (
                            <Button variant="outlined" onClick={toBuyCredits}>
                                {t("BuyCredits")}
                            </Button>
                        )}
                    </FooterContainer>
                )}
            </ScrollBox>
        </PageContainer>
    );
}

