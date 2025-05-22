import { LINKS } from "@local/shared";
import { Box, Button, Container, Stack, type SxProps, type Theme, Typography } from "@mui/material";
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
    minHeight: "calc(100vh - 64px)", // Adjust based on TopBar height
    py: 4, // Add padding for smaller screens
    overflowY: "auto",
};

const contentBoxStyles: SxProps<Theme> = {
    width: "min(700px, 100%)",
    display: "flex",
    flexDirection: "column",
    alignItems: "center",
};

const imageContainerStyles: SxProps<Theme> = {
    width: "100%",
    height: "auto",
    maxWidth: "500px",
    display: "flex",
    justifyContent: "center",
    mb: 2,
};

const imageStyles: SxProps<Theme> = {
    width: "100%",
    height: "auto",
    maxHeight: "300px",
};

export function NotFoundView(_props: ViewProps) {
    const { t } = useTranslation();
    const hasPreviousPage = Boolean(sessionStorage.getItem("lastPath"));

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
                <Box
                    sx={contentBoxStyles}
                >
                    <Box sx={imageContainerStyles}>
                        <img
                            alt="A lop-eared bunny in a pastel-themed workspace with a notepad and pen, looking slightly worried."
                            src={Bunny404}
                            style={{ width: "100%", height: "auto", maxWidth: "100%" }}
                        />
                    </Box>
                    <Typography component="h1" variant="h3" textAlign="center" mt={2} mb={2}>
                        {t("PageNotFound", { ns: "error", defaultValue: "Page Not Found" })}
                    </Typography>
                    <Typography variant="body1" textAlign="center" mb={4}>
                        {t("PageNotFoundDetails", { ns: "error", defaultValue: "PageNotFoundDetails" })}
                    </Typography>
                    <Stack direction="row" spacing={2} justifyContent="center" alignItems="center">
                        {hasPreviousPage ? (
                            <Button
                                aria-label={t("GoBack")}
                                variant="contained"
                                onClick={goBack}
                                startIcon={<IconCommon
                                    decorative
                                    name="ArrowLeft"
                                />}
                            >{t("GoBack")}</Button>
                        ) : null}
                        <Link to={LINKS.Home}>
                            <Button
                                aria-label={t("GoToHome")}
                                variant="contained"
                                startIcon={<IconCommon
                                    decorative
                                    name="Home"
                                />}
                            >{t("GoToHome")}</Button>
                        </Link>
                    </Stack>
                </Box>
            </Container>
        </>
    );
}
