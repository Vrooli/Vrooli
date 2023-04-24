import { LINKS } from ":/consts";
import { useContext } from "react";
import { PubSub } from "../../utils/pubsub";
import { Redirect, useLocation } from "../../utils/route";
import { SessionContext } from "../../utils/SessionContext";
import { PageProps } from "../../views/wrapper/types";
import { PageContainer } from "../containers/PageContainer/PageContainer";

export const Page = ({
    children,
    excludePageContainer = false,
    mustBeLoggedIn = false,
    redirect = LINKS.Start,
    sessionChecked,
    sx,
}: PageProps) => {
    const session = useContext(SessionContext);
    const [location] = useLocation();

    // If this page has restricted access
    if (mustBeLoggedIn) {
        if (session?.isLoggedIn) return children;
        if (sessionChecked && location !== redirect) {
            PubSub.get().publishSnack({ messageKey: "PageRestricted", severity: "Error" });
            return <Redirect to={redirect} />;
        }
        return null;
    }

    if (excludePageContainer) return <>{children}</>;
    return (
        <PageContainer sx={sx}>
            {children}
        </PageContainer>
    );
};
