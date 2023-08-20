import { SessionContext } from "contexts/SessionContext";
import { useContext, useMemo } from "react";
import { checkIfLoggedIn } from "utils/authentication/session";
import { DashboardView, LandingView } from "..";
import { HomeViewProps } from "../types";

export const HomeView = (props: HomeViewProps) => {
    const session = useContext(SessionContext);
    const isLoggedIn = useMemo(() => checkIfLoggedIn(session), [session]);

    if (isLoggedIn) return <DashboardView  {...props} />;
    return <LandingView {...props} />;
};
