import { LINKS, UrlTools } from "@local/shared";
import { Box, type BoxProps, styled, useTheme } from "@mui/material";
import { useContext, useEffect, useMemo } from "react";
import { SessionContext } from "../../contexts/session.js";
import { Redirect, useLocation } from "../../route/router.js";
import { pagePaddingBottom } from "../../styles.js";
import { type PageProps, type SxType } from "../../types.js";
import { ELEMENT_IDS } from "../../utils/consts.js";
import { PubSub } from "../../utils/pubsub.js";

interface PageContainerProps extends BoxProps {
    contentType?: "normal" | "text";
    /**
     * Full size causes scrollbar to be all the way to the right. 
     * Normal size causes scrollbar to be to the right of the page content.
     */
    size?: "normal" | "fullSize";
    sx?: SxType;
}
const StyledPageContainer = styled(Box, {
    shouldForwardProp: (prop) => prop !== "contentType" && prop !== "size" && prop !== "sx",
})<PageContainerProps>(({ contentType, size, sx, theme }) => ({
    background: contentType === "text" ?
        theme.palette.background.paper :
        theme.palette.background.default,
    minWidth: "100%",
    width: size === "fullSize" ? "100%" : "min(100%, 800px)",
    height: "100vh",
    overflow: "hidden",
    margin: "auto",
    paddingBottom: pagePaddingBottom,
    paddingLeft: size === "fullSize" ? 0 : "max(1em, calc(15% - 75px))",
    paddingRight: size === "fullSize" ? 0 : "max(1em, calc(15% - 75px))",
    position: "relative",
    ...sx,
} as any));
export function PageContainer({
    children,
    ...props
}: PageContainerProps) {
    return <StyledPageContainer id={ELEMENT_IDS.PageContainer} {...props}>
        {children}
    </StyledPageContainer>;
}

/**
 * Hidden div under the page for top overscroll color
 */
interface PageColorProps extends BoxProps {
    background: string;
}

const PageColor = styled(Box, {
    shouldForwardProp: (prop) => prop !== "background",
})<PageColorProps>(({ background }) => ({
    background,
    position: "fixed",
    height: "100vh",
    width: "100vw",
    zIndex: -3, // Below the footer's hidden div
}));

export function Page({
    children,
    excludePageContainer = false,
    mustBeLoggedIn = false,
    sessionChecked,
    sx,
}: PageProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [{ pathname }] = useLocation();

    const background = useMemo(function backgroundMemo() {
        const backgroundColor = (sx as { background?: string })?.background
            ?? (sx as { backgroundColor?: string })?.backgroundColor
            ?? (palette.background.default);
        return backgroundColor;
    }, [palette.background.default, sx]);

    useEffect(function backgroundEffect() {
        const rootDiv = document.getElementById("root");
        const appDiv = document.getElementById("App");

        const originalHtmlBackground = document.documentElement.style.background;
        const originalBodyBackground = document.body.style.background;
        const originalRootBackground = rootDiv?.style.background;
        const originalAppBackground = appDiv?.style.background;

        document.documentElement.style.background = background;
        document.body.style.background = background;
        if (rootDiv) rootDiv.style.background = background;
        if (appDiv) appDiv.style.background = background;

        return () => {
            document.documentElement.style.background = originalHtmlBackground;
            document.body.style.background = originalBodyBackground;
            if (rootDiv && originalRootBackground) rootDiv.style.background = originalRootBackground;
            if (appDiv && originalAppBackground) appDiv.style.background = originalAppBackground;
        };
    }, [background]);

    // If this page has restricted access
    if (mustBeLoggedIn) {
        if (session?.isLoggedIn) {
            if (excludePageContainer) {
                return children;
            }
            return (<PageContainer sx={sx}>
                {children}
            </PageContainer>);
        }
        if (sessionChecked && pathname !== LINKS.Signup) {
            console.log("Redirecting to signup page...", sessionChecked, pathname, mustBeLoggedIn, session);
            PubSub.get().publish("snack", { messageKey: "PageRestricted", severity: "Error" });
            return <Redirect to={UrlTools.linkWithSearchParams(LINKS.Signup, { redirect: pathname })} />;
        }
        return null;
    }

    return (
        <>
            <PageColor background={background} />
            {!excludePageContainer && <PageContainer sx={sx}>
                {children}
            </PageContainer>}
            {excludePageContainer && children}
        </>
    );
}
