import { Suspense, useContext } from "react";
import { lazily } from "react-lazily";
import { SessionContext } from "../../contexts/session.js";
import { checkIfLoggedIn } from "../../utils/authentication/session.js";
import { HomeViewProps } from "./types.js";

const { DashboardView } = lazily(() => import("./DashboardView.js"));
const { LandingView } = lazily(() => import("./LandingView.js"));

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
