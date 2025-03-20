import { LINKS } from "@local/shared";
import { Box, Button, Stack, Typography } from "@mui/material";
import { useTranslation } from "react-i18next";
import { Link } from "route";
import Bunny404 from "../../assets/img/Bunny404.svg";
import { TopBar } from "../../components/navigation/TopBar.js";
import { ArrowLeftIcon, HomeIcon } from "../../icons/common.js";
import { SlideImage, SlideImageContainer } from "../../styles.js";

function goBack() {
    window.history.back();
}

export function NotFoundView() {
    const { t } = useTranslation();
    const hasPreviousPage = Boolean(sessionStorage.getItem("lastPath"));

    return (
        <>
            <TopBar
                display="page"
                titleBehaviorDesktop="Hide"
                titleBehaviorMobile="Hide"
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
                <SlideImageContainer>
                    <SlideImage
                        alt="A lop-eared bunny in a pastel-themed workspace with a notepad and pen, looking slightly worried."
                        src={Bunny404}
                    />
                </SlideImageContainer>
                <Typography component="h1" variant="h3" textAlign="center" mt={2} mb={2}>
                    {t("PageNotFound", { ns: "error", defaultValue: "Page Not Found" })}
                </Typography>
                <Typography variant="body1" textAlign="center" mb={4}>
                    {t("PageNotFoundDetails", { ns: "error", defaultValue: "PageNotFoundDetails" })}
                </Typography>
                <Stack direction="row" spacing={2} justifyContent="center" alignItems="center">
                    {hasPreviousPage ? (
                        <Button
                            variant="contained"
                            onClick={goBack}
                            startIcon={<ArrowLeftIcon />}
                        >{t("GoBack")}</Button>
                    ) : null}
                    <Link to={LINKS.Home}>
                        <Button
                            variant="contained"
                            startIcon={<HomeIcon />}
                        >{t("GoToHome")}</Button>
                    </Link>
                </Stack>
            </Box>
        </>
    );
}
