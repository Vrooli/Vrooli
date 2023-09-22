import { LINKS } from "@local/shared";
import { Box, useTheme } from "@mui/material";
import { PageContainer } from "components/containers/PageContainer/PageContainer";
import { SessionContext } from "contexts/SessionContext";
import { useContext } from "react";
import { Redirect, stringifySearchParams, useLocation } from "route";
import { PubSub } from "utils/pubsub";
import { PageProps } from "../../views/wrapper/types";

export const Page = ({
    children,
    excludePageContainer = false,
    mustBeLoggedIn = false,
    sessionChecked,
    sx,
}: PageProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [location] = useLocation();

    // If this page has restricted access
    if (mustBeLoggedIn) {
        if (session?.isLoggedIn) return children;
        if (sessionChecked && location !== LINKS.Start) {
            PubSub.get().publishSnack({ messageKey: "PageRestricted", severity: "Error" });
            return <Redirect to={`${LINKS.Start}${stringifySearchParams({ redirect: location })}`} />;
        }
        return null;
    }

    return (
        <>
            {/* Hidden div under the page for top overscroll color.
            Color should mimic `content-wrap` component, but with sx override */}
            <Box sx={{
                backgroundColor: (sx as any)?.background ?? (sx as any)?.backgroundColor ?? (palette.mode === "light" ? "#c2cadd" : palette.background.default),
                height: "100vh",
                position: "fixed",
                top: "0",
                width: "100%",
                zIndex: -3, // Below the footer's hidden div
            }} />
            {!excludePageContainer && <PageContainer sx={sx}>
                {children}
            </PageContainer>}
            {excludePageContainer && children}
        </>
    );
};
