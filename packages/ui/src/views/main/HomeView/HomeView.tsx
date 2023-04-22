import { useContext, useMemo } from "react";
import { DashboardView, LandingView } from "..";
import { checkIfLoggedIn } from "../../../utils/authentication/session";
import { SessionContext } from "../../../utils/SessionContext";
import { HomeViewProps } from "../types";

export const HomeView = (props: HomeViewProps) => {
    const session = useContext(SessionContext);
    const isLoggedIn = useMemo(() => checkIfLoggedIn(session), [session]);

    if (isLoggedIn) return <DashboardView  {...props} />
    return <LandingView {...props} />
}