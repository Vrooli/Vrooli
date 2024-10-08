import { SessionContext } from "contexts";
import { Suspense, useContext } from "react";
import { lazily } from "react-lazily";
import { checkIfLoggedIn } from "utils/authentication/session";
import { LandingView } from "../LandingView/LandingView";
import { HomeViewProps } from "../types";

const { DashboardView } = lazily(() => import("../DashboardView/DashboardView"));

export const HomeView = (props: HomeViewProps) => {
    const session = useContext(SessionContext);
    const isLoggedIn = checkIfLoggedIn(session);

    if (isLoggedIn) {
        return (
            <Suspense>
                <DashboardView {...props} />
            </Suspense>
        );
    }

    return <LandingView {...props} />;
};
