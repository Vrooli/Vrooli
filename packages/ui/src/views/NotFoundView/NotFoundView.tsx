import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Container from "@mui/material/Container";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import Chip from "@mui/material/Chip";
import Fade from "@mui/material/Fade";
import type { SxProps, Theme } from "@mui/material";
import { useTheme } from "@mui/material/styles";
import { LINKS } from "@vrooli/shared";
import { useTranslation } from "react-i18next";
import Bunny404 from "../../assets/img/Bunny404.svg";
import { TopBar } from "../../components/navigation/TopBar.js";
import { IconCommon } from "../../icons/Icons.js";
import { Link } from "../../route/router.js";
import { type ViewProps } from "../../types.js";

function goBack() {
    window.history.back();
}

const containerStyles: SxProps<Theme> = {
    display: "flex",
    flexDirection: "column",
    justifyContent: "center",
    alignItems: "center",
    minHeight: "calc(100vh - 64px)",
    py: 4,
    overflowY: "auto",
    position: "relative",
};

const contentBoxStyles: SxProps<Theme> = {
    width: "min(700px, 100%)",
    display: "flex",
    flexDirection: "column",
    alignItems: "center",
    position: "relative",
    zIndex: 1,
};

const imageContainerStyles: SxProps<Theme> = {
    width: "100%",
    height: "auto",
    maxWidth: "400px",
    display: "flex",
    justifyContent: "center",
    mb: 3,
};

const errorCodeStyles: SxProps<Theme> = (theme: Theme) => ({
    fontSize: "120px",
    fontWeight: 900,
    color: theme.palette.primary.main,
    opacity: 0.1,
    position: "absolute",
    top: "50%",
    left: "50%",
    transform: "translate(-50%, -50%)",
    userSelect: "none",
    [theme.breakpoints.down("sm")]: {
        fontSize: "80px",
    },
});

export function NotFoundView(_props: ViewProps) {
    const { t } = useTranslation();
    const theme = useTheme();
    const hasPreviousPage = Boolean(sessionStorage.getItem("lastPath"));

    const popularPages = [
        { label: t("Dashboard"), link: LINKS.Dashboard, icon: "Grid" },
        { label: t("Search"), link: LINKS.Search, icon: "Search" },
        { label: t("Projects"), link: LINKS.Projects, icon: "Project" },
        { label: t("Teams"), link: LINKS.Teams, icon: "Team" },
    ];

    return (
        <>
            <TopBar
                display="Page"
                titleBehavior="Hide"
            />
            <Container
                maxWidth="md"
                sx={containerStyles}
            >
                <Typography sx={errorCodeStyles}>404</Typography>
                <Fade in timeout={1000}>
                    <Box sx={contentBoxStyles}>
                        <Box sx={imageContainerStyles}>
                            <img
                                alt="A lop-eared bunny in a pastel-themed workspace with a notepad and pen, looking slightly worried."
                                src={Bunny404}
                                style={{ width: "100%", height: "auto", maxWidth: "100%" }}
                            />
                        </Box>
                        <Typography 
                            component="h1" 
                            variant="h3" 
                            textAlign="center" 
                            mb={1}
                            sx={{
                                fontWeight: 700,
                                color: theme.palette.text.primary,
                                [theme.breakpoints.down("sm")]: {
                                    fontSize: "2rem",
                                },
                            }}
                        >
                            {t("PageNotFound", { ns: "error", defaultValue: "Page Not Found" })}
                        </Typography>
                        <Typography 
                            variant="body1" 
                            textAlign="center" 
                            mb={4}
                            sx={{
                                color: theme.palette.text.secondary,
                                maxWidth: "500px",
                                mx: "auto",
                            }}
                        >
                            {t("PageNotFoundDetails", { ns: "error", defaultValue: "Oops! The page you're looking for seems to have hopped away. Let's get you back on track!" })}
                        </Typography>
                        
                        <Stack direction="row" spacing={2} justifyContent="center" alignItems="center" mb={4} flexWrap="wrap">
                            {hasPreviousPage ? (
                                <Button
                                    aria-label={t("GoBack")}
                                    variant="outlined"
                                    onClick={goBack}
                                    size="large"
                                    startIcon={<IconCommon decorative name="ArrowLeft" />}
                                    sx={{
                                        borderWidth: 2,
                                        "&:hover": {
                                            borderWidth: 2,
                                        },
                                    }}
                                >{t("GoBack")}</Button>
                            ) : null}
                            <Link to={LINKS.Home}>
                                <Button
                                    aria-label={t("GoToHome")}
                                    variant="contained"
                                    size="large"
                                    startIcon={<IconCommon decorative name="Home" />}
                                    sx={{
                                        backgroundColor: theme.palette.primary.main,
                                        "&:hover": {
                                            backgroundColor: theme.palette.primary.dark,
                                        },
                                    }}
                                >{t("GoToHome")}</Button>
                            </Link>
                        </Stack>

                        <Box sx={{ mt: 2, mb: 2 }}>
                            <Typography 
                                variant="body2" 
                                textAlign="center" 
                                sx={{ 
                                    color: theme.palette.text.secondary,
                                    mb: 2,
                                }}
                            >
                                {t("PopularPages", { defaultValue: "Popular pages:" })}
                            </Typography>
                            <Stack 
                                direction="row" 
                                spacing={1} 
                                justifyContent="center" 
                                flexWrap="wrap"
                                sx={{ gap: 1 }}
                            >
                                {popularPages.map((page) => (
                                    <Link key={page.link} to={page.link}>
                                        <Chip
                                            icon={<IconCommon decorative name={page.icon as any} />}
                                            label={page.label}
                                            clickable
                                            variant="outlined"
                                            sx={{
                                                borderColor: theme.palette.divider,
                                                "&:hover": {
                                                    borderColor: theme.palette.primary.main,
                                                    backgroundColor: theme.palette.action.hover,
                                                },
                                            }}
                                        />
                                    </Link>
                                ))}
                            </Stack>
                        </Box>
                    </Box>
                </Fade>
            </Container>
        </>
    );
}
