import { SessionContext } from "contexts";
import { Suspense, useContext } from "react";
import { lazily } from "react-lazily";
import { checkIfLoggedIn } from "utils/authentication/session.js";
import { HomeViewProps } from "../types.js";

const { DashboardView } = lazily(() => import("../DashboardView/DashboardView"));
const { LandingView } = lazily(() => import("../LandingView/LandingView"));

export function HomeView(props: HomeViewProps) {
    const session = useContext(SessionContext);
    const isLoggedIn = checkIfLoggedIn(session);

    if (isLoggedIn) {
        return (
            <Suspense>
                <DashboardView {...props} />
            </Suspense>
        );
    }

    return (
        <Suspense>
            <LandingView {...props} />
        </Suspense>
    );
}
