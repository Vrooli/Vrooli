import { LINKS } from "@local/shared";
import { Box, Button, Stack, Typography } from "@mui/material";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useTranslation } from "react-i18next";
import { Link } from "route";

export const NotFoundView = () => {
    const { t } = useTranslation();
    const hasPreviousPage = Boolean(sessionStorage.getItem("lastPath"));

    return (
        <>
            <TopBar
                display="page"
                hideTitleOnDesktop
                title={t("PageNotFound", { ns: "error", defaultValue: "Page Not Found" })}
            />
            <Box
                sx={{
                    position: "absolute",
                    top: "50%",
                    left: "50%",
                    transform: "translateX(-50%) translateY(-50%)",
                    width: "min(700px, 80%)",
                }}
            >
                <Typography component="h1" variant="h3" textAlign="center" mb={2}>
                    {t("PageNotFound", { ns: "error", defaultValue: "Page Not Found" })}
                </Typography>
                <Typography variant="body1" textAlign="center" mb={4}>
                    {t("PageNotFoundDetails", { ns: "error", defaultValue: "PageNotFoundDetails" })}
                </Typography>
                <Stack direction="row" spacing={2} justifyContent="center" alignItems="center">
                    {hasPreviousPage ? (
                        <Button variant="contained" onClick={() => { window.history.back(); }}>{t("GoBack")}</Button>
                    ) : null}
                    <Link to={LINKS.Home}>
                        <Button variant="contained">{t("GoToHome")}</Button>
                    </Link>
                </Stack>
            </Box>
        </>
    );
};
