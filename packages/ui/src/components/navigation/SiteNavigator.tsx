import { API_CREDITS_MULTIPLIER, ListObject, getObjectUrl } from "@local/shared";
import { Avatar, Box, Collapse, Divider, IconButton, List, ListItem, ListItemIcon, ListItemText, Toolbar, Typography, styled, useTheme } from "@mui/material";
import { useCallback, useContext, useMemo, useState } from "react";
import { SessionContext } from "../../contexts.js";
import { useMenu } from "../../hooks/useMenu.js";
import { useWindowSize } from "../../hooks/useWindowSize.js";
import { ExpandLessIcon, ExpandMoreIcon, ListIcon, PlusIcon, ProfileIcon, SearchIcon } from "../../icons/common.js";
import { useLocation } from "../../route/router.js";
import { ScrollBox } from "../../styles.js";
import { checkIfLoggedIn, getCurrentUser } from "../../utils/authentication/session.js";
import { ELEMENT_IDS } from "../../utils/consts.js";
import { extractImageUrl } from "../../utils/display/imageTools.js";
import { MenuPayloads, PubSub } from "../../utils/pubsub.js";
import { PageContainer } from "../Page/Page.js";

/** Threshold for showing low credit balance */
// eslint-disable-next-line no-magic-numbers
const LOW_CREDIT_THRESHOLD = BigInt(5_00) * API_CREDITS_MULTIPLIER;
const SHORT_TAKE = 10;
const AVATAR_SIZE_PX = 50;

const NoResultsText = styled(Typography)(({ theme }) => ({
    color: theme.palette.background.textSecondary,
    fontStyle: "italic",
    padding: theme.spacing(1),
    textAlign: "center",
}));

const StyledToolbar = styled(Toolbar)(({ theme }) => ({
    backgroundColor: theme.palette.primary.dark,
    padding: "8px 16px",
}));

const FooterContainer = styled("div")(({ theme }) => ({
    marginTop: "auto",
    padding: "16px",
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    borderTop: `1px solid ${theme.palette.divider}`,
}));

const ProfileAvatar = styled(Avatar)(({ theme }) => ({
    background: theme.palette.primary.contrastText,
    width: "40px",
    height: "40px",
    cursor: "pointer",
}));

// Should be both bots and teams
const navItems = [
    { text: "ChatGPT", icon: <Avatar sx={{ bgcolor: "#3B82F6" }}>CG</Avatar> },
    { text: "Sora", icon: <Avatar sx={{ bgcolor: "#10B981" }}>S</Avatar> },
    { text: "DALL-E", icon: <Avatar sx={{ bgcolor: "#F59E0B" }}>D</Avatar> },
    { text: "Vrooli Product Manager", icon: <Avatar sx={{ bgcolor: "#8B5CF6" }}>V</Avatar> },
    { text: "Tweet Responder", icon: <Avatar sx={{ bgcolor: "#EC4899" }}>T</Avatar> },
    { text: "Explore GPTs", icon: <Avatar sx={{ bgcolor: "#6B7280" }}>E</Avatar> },
];

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

export function SiteNavigator() {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

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

    const handleItemClick = useCallback((data: ListObject) => {
        setLocation(getObjectUrl(data));
    }, [setLocation]);

    const [projectsOpen, setProjectsOpen] = useState(true);

    const handleProjectsClick = () => {
        setProjectsOpen(!projectsOpen);
    };

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

    // Define styles to avoid linter errors
    function projectItemStyle(selected: boolean) {
        return {
            pl: 4,
            bgcolor: selected ? "#3B82F6" : "transparent",
        };
    }
    const addProjectStyle = { pl: 4 };
    const dividerStyle = { my: 1, bgcolor: "#333333" };
    const yesterdayLabelStyle = { color: "#9CA3AF" };
    const yesterdayItemStyle = {
        noWrap: true,
        sx: { color: "#9CA3AF" },
    };
    const viewPlansStyle = { color: "#9CA3AF", cursor: "pointer" };
    const captionStyle = { color: "#6B7280" };
    const creditsStyle = { color: "#9CA3AF", cursor: "pointer" };

    return (
        <PageContainer size="fullSize">
            <ScrollBox>
                {/* Header */}
                <StyledToolbar>
                    <IconButton edge="start" color="inherit" aria-label="menu" onClick={handleClose}>
                        <ListIcon fill={palette.primary.contrastText} />
                    </IconButton>
                    <Box ml="auto">
                        <IconButton color="inherit" aria-label="search" onClick={handleOpenSearch}>
                            <SearchIcon fill={palette.primary.contrastText} />
                        </IconButton>
                        <IconButton color="inherit" aria-label="new-chat">
                            <PlusIcon fill={palette.primary.contrastText} />
                        </IconButton>
                    </Box>
                </StyledToolbar>

                {/* Navigation List */}
                <List>
                    {navItems.map((item, index) => (
                        <ListItem button key={index} onClick={() => handleItemClick(item)}>
                            <ListItemIcon>{item.icon}</ListItemIcon>
                            <ListItemText primary={item.text} />
                        </ListItem>
                    ))}

                    {/* Projects Section */}
                    <ListItem button onClick={handleProjectsClick}>
                        <ListItemText primary="Projects" />
                        {projectsOpen ? <ExpandLessIcon /> : <ExpandMoreIcon />}
                    </ListItem>
                    <Collapse in={projectsOpen} timeout="auto" unmountOnExit>
                        <List component="div" disablePadding>
                            {projects.map((project, index) => (
                                <ListItem
                                    button
                                    key={index}
                                    selected={project.selected}
                                    sx={projectItemStyle(project.selected)}
                                >
                                    <ListItemText primary={project.text} />
                                </ListItem>
                            ))}
                            {/* Add and See More buttons for Projects */}
                            <ListItem button onClick={handleAddProject} sx={addProjectStyle}>
                                <ListItemIcon>
                                    <PlusIcon />
                                </ListItemIcon>
                                <ListItemText primary="Add Project" />
                            </ListItem>
                            <ListItem button onClick={handleSeeMoreProjects} sx={addProjectStyle}>
                                <ListItemText primary="See more" />
                            </ListItem>
                        </List>
                    </Collapse>

                    {/* Yesterday Section */}
                    <Divider sx={dividerStyle} />
                    <ListItem>
                        <ListItemText primary="Yesterday" sx={yesterdayLabelStyle} />
                    </ListItem>
                    {yesterdayItems.map((item, index) => (
                        <ListItem button key={index}>
                            <ListItemText
                                primary={item}
                                primaryTypographyProps={yesterdayItemStyle}
                            />
                        </ListItem>
                    ))}
                </List>

                {/* Footer */}
                {showFooter && (
                    <FooterContainer>
                        {/* Always show profile icon on mobile */}
                        {isMobile && (
                            <ProfileAvatar
                                id={ELEMENT_IDS.UserMenuProfileIcon}
                                src={isLoggedIn ? extractImageUrl(user.profileImage, user.updated_at, AVATAR_SIZE_PX) : undefined}
                                onClick={openUserMenu}
                            >
                                <ProfileIcon fill={palette.primary.dark} width="100%" height="100%" />
                            </ProfileAvatar>
                        )}
                        {/* Show credits info only when logged in */}
                        {isLoggedIn && (
                            hasPremium ? (
                                <Typography variant="body2" sx={creditsStyle}>
                                    {`Credits left: $${creditsAsDollars}`}
                                </Typography>
                            ) : (
                                <>
                                    <Typography variant="body2" sx={viewPlansStyle}>
                                        View plans
                                    </Typography>
                                    <Typography variant="caption" sx={captionStyle}>
                                        Unlimited access, team features...
                                    </Typography>
                                </>
                            )
                        )}
                        {/* Show login prompt when not logged in */}
                        {isMobile && !isLoggedIn && (
                            <Typography variant="body2" sx={viewPlansStyle}>
                                Log in to access all features
                            </Typography>
                        )}
                    </FooterContainer>
                )}
            </ScrollBox>
        </PageContainer>
    );
}

