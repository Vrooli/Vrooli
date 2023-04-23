import { jsx as _jsx, Fragment as _Fragment } from "react/jsx-runtime";
import { LINKS } from "@local/consts";
import { useContext } from "react";
import { PubSub } from "../../utils/pubsub";
import { Redirect, useLocation } from "../../utils/route";
import { SessionContext } from "../../utils/SessionContext";
import { PageContainer } from "../containers/PageContainer/PageContainer";
export const Page = ({ children, excludePageContainer = false, mustBeLoggedIn = false, redirect = LINKS.Start, sessionChecked, sx, }) => {
    const session = useContext(SessionContext);
    const [location] = useLocation();
    if (mustBeLoggedIn) {
        if (session?.isLoggedIn)
            return children;
        if (sessionChecked && location !== redirect) {
            PubSub.get().publishSnack({ messageKey: "PageRestricted", severity: "Error" });
            return _jsx(Redirect, { to: redirect });
        }
        return null;
    }
    if (excludePageContainer)
        return _jsx(_Fragment, { children: children });
    return (_jsx(PageContainer, { sx: sx, children: children }));
};
//# sourceMappingURL=Page.js.map