import { jsx as _jsx } from "react/jsx-runtime";
import { useContext, useMemo } from "react";
import { DashboardView, LandingView } from "..";
import { checkIfLoggedIn } from "../../../utils/authentication/session";
import { SessionContext } from "../../../utils/SessionContext";
export const HomeView = (props) => {
    const session = useContext(SessionContext);
    const isLoggedIn = useMemo(() => checkIfLoggedIn(session), [session]);
    if (isLoggedIn)
        return _jsx(DashboardView, { ...props });
    return _jsx(LandingView, { ...props });
};
//# sourceMappingURL=HomeView.js.map