import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Card from "@mui/material/Card";
import CardContent from "@mui/material/CardContent";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import Typography from "@mui/material/Typography";
import { styled, useTheme } from "@mui/material/styles";
import { LINKS, type ProjectVersion } from "@vrooli/shared";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { PageContainer } from "../../components/Page/Page.js";
import { Navbar } from "../../components/navigation/Navbar.js";
import { SessionContext } from "../../contexts/session.js";
import { useFindMany } from "../../hooks/useFindMany.js";
import { Icon, IconCommon } from "../../icons/Icons.js";
import { useLocation } from "../../route/router.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { type MyStuffViewProps } from "./types.js";

const StyledCard = styled(Card)(({ theme }) => ({
    height: "100%",
    display: "flex",
    flexDirection: "column",
    transition: "transform 0.2s ease-in-out, box-shadow 0.2s ease-in-out",
    cursor: "pointer",
    "&:hover": {
        transform: "translateY(-4px)",
        boxShadow: theme.shadows[8],
    },
}));

const IconWrapper = styled(Box)(({ theme }) => ({
    width: 64,
    height: 64,
    borderRadius: theme.shape.borderRadius,
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    marginBottom: theme.spacing(2),
}));

const EmptyStateContainer = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "column",
    alignItems: "center",
    justifyContent: "center",
    padding: theme.spacing(8, 2),
    textAlign: "center",
    minHeight: "60vh",
}));

const ProjectCard = styled(Card)(({ theme }) => ({
    marginBottom: theme.spacing(2),
    cursor: "pointer",
    transition: "all 0.2s ease-in-out",
    "&:hover": {
        transform: "translateX(4px)",
        boxShadow: theme.shadows[4],
    },
}));

/**
 * MyStuffView - Transitional implementation that provides links to SearchView
 * with appropriate permissions filters while preparing for project-centric redesign
 */
export function MyStuffView({
    display,
}: MyStuffViewProps) {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();
    const currentUser = useMemo(() => getCurrentUser(session), [session]);

    // Fetch user's projects for the project-centric view
    const projectsData = useFindMany<ProjectVersion>({
        searchType: "ProjectVersion",
        take: 10,
        where: {
            root: {
                owner: {
                    User: { id: currentUser.id },
                },
            },
            isLatest: true,
        },
    });

    // Navigation functions for different search views
    const navigateToSearch = useCallback((filter: "Own" | "Team" | "Public") => {
        // Navigate to search view with the appropriate permissions filter
        // We'll need to pass the filter through URL params
        setLocation(`${LINKS.Search}?permissionsFilter=${filter}`);
    }, [setLocation]);

    const navigateToProject = useCallback((project: ProjectVersion) => {
        setLocation(`/project/${project.root?.handle ?? project.id}`);
    }, [setLocation]);

    // Quick access cards data
    const quickAccessCards = [
        {
            title: "MyItems",
            description: "ViewEditPersonalContent",
            icon: "User",
            color: palette.primary.main,
            onClick: () => navigateToSearch("Own"),
        },
        {
            title: "TeamItems",
            description: "CollaborativeContent",
            icon: "Team",
            color: palette.secondary.main,
            onClick: () => navigateToSearch("Team"),
        },
        {
            title: "Bookmarks",
            description: "SavedForLater",
            icon: "BookmarkFilled",
            color: palette.info.main,
            onClick: () => setLocation(LINKS.Bookmarks),
        },
        {
            title: "History",
            description: "RecentlyViewed",
            icon: "History",
            color: palette.warning.main,
            onClick: () => setLocation(LINKS.History),
        },
    ];

    return (
        <PageContainer>
            <Navbar title={t("MyStuff")} />
            
            {/* Header Section */}
            <Box sx={{ p: 3 }}>
                <Typography variant="h4" gutterBottom>
                    {t("Welcome")}, {currentUser.name || t("User")}!
                </Typography>
                <Typography variant="body1" color="text.secondary" sx={{ mb: 4 }}>
                    {t("MyStuffTransitionMessage")}
                </Typography>

                {/* Quick Access Cards */}
                <Typography variant="h5" sx={{ mb: 2 }}>
                    {t("QuickAccess")}
                </Typography>
                <Grid container spacing={3} sx={{ mb: 4 }}>
                    {quickAccessCards.map((card) => (
                        <Grid item xs={12} sm={6} md={3} key={card.title}>
                            <StyledCard onClick={card.onClick}>
                                <CardContent sx={{ 
                                    display: "flex", 
                                    flexDirection: "column", 
                                    alignItems: "center",
                                    textAlign: "center",
                                    height: "100%",
                                }}>
                                    <IconWrapper sx={{ backgroundColor: `${card.color}20` }}>
                                        <Icon
                                            decorative
                                            info={{ name: card.icon, fill: card.color }}
                                            size={32}
                                        />
                                    </IconWrapper>
                                    <Typography variant="h6" gutterBottom>
                                        {t(card.title)}
                                    </Typography>
                                    <Typography variant="body2" color="text.secondary">
                                        {t(card.description)}
                                    </Typography>
                                </CardContent>
                            </StyledCard>
                        </Grid>
                    ))}
                </Grid>

                {/* Projects Section */}
                <Box sx={{ mb: 3, display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                    <Typography variant="h5">
                        {t("YourProjects")}
                    </Typography>
                    <Button
                        startIcon={<IconCommon name="Add" decorative />}
                        onClick={() => setLocation("/project/add")}
                        variant="contained"
                    >
                        {t("NewProject")}
                    </Button>
                </Box>

                {/* Projects List */}
                {projectsData.loading ? (
                    <Typography>{t("Loading")}...</Typography>
                ) : projectsData.allData.length === 0 ? (
                    <EmptyStateContainer>
                        <Icon
                            decorative
                            info={{ name: "Project", fill: palette.text.secondary }}
                            size={80}
                            sx={{ mb: 2, opacity: 0.5 }}
                        />
                        <Typography variant="h6" gutterBottom>
                            {t("NoProjectsYet")}
                        </Typography>
                        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
                            {t("ProjectsOrganizeWork")}
                        </Typography>
                        <Button
                            variant="contained"
                            startIcon={<IconCommon name="Add" decorative />}
                            onClick={() => setLocation("/project/add")}
                        >
                            {t("CreateFirstProject")}
                        </Button>
                    </EmptyStateContainer>
                ) : (
                    <Box>
                        {projectsData.allData.map((project) => (
                            <ProjectCard key={project.id} onClick={() => navigateToProject(project)}>
                                <CardContent sx={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
                                    <Box>
                                        <Typography variant="h6">
                                            {project.translations?.[0]?.name || project.root?.handle || t("Untitled")}
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary">
                                            {project.translations?.[0]?.description || t("NoDescription")}
                                        </Typography>
                                    </Box>
                                    <IconButton>
                                        <IconCommon name="ChevronRight" decorative />
                                    </IconButton>
                                </CardContent>
                            </ProjectCard>
                        ))}
                        {projectsData.allData.length >= 10 && (
                            <Button
                                fullWidth
                                variant="outlined"
                                onClick={() => navigateToSearch("Own")}
                                sx={{ mt: 2 }}
                            >
                                {t("ViewAllProjects")}
                            </Button>
                        )}
                    </Box>
                )}
            </Box>
        </PageContainer>
    );
}
