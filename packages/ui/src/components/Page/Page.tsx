import { LINKS, stringifySearchParams } from "@local/shared";
import { Box, BoxProps, styled, useTheme } from "@mui/material";
import { PageContainer } from "components/containers/PageContainer/PageContainer";
import { SessionContext } from "contexts/SessionContext";
import { useContext, useEffect, useMemo } from "react";
import { Redirect, useLocation } from "route";
import { PageProps } from "types";
import { PubSub } from "utils/pubsub";

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
            ?? (palette.mode === "light" ? "#c2cadd" : palette.background.default);
        return backgroundColor;
    }, [palette.background.default, palette.mode, sx]);

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
        if (session?.isLoggedIn) return children;
        if (sessionChecked && pathname !== LINKS.Signup) {
            PubSub.get().publish("snack", { messageKey: "PageRestricted", severity: "Error" });
            return <Redirect to={`${LINKS.Signup}${stringifySearchParams({ redirect: pathname })}`} />;
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
