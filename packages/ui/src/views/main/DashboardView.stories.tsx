import { LINKS } from "@local/shared";
import { signedInNoPremiumNoCreditsSession, signedInNoPremiumWithCreditsSession, signedInPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../__test/storybookConsts.js";
import { DashboardView } from "./DashboardView.js";

export default {
    title: "Views/Main/DashboardView",
    component: DashboardView,
};

export function SignedInNoPremiumNoCredits() {
    return (
        <DashboardView display="page" />
    );
}
SignedInNoPremiumNoCredits.parameters = {
    session: signedInNoPremiumNoCreditsSession,
    route: {
        path: LINKS.Home,
    },
};

export function SignedInNoPremiumWithCredits() {
    return (
        <DashboardView display="page" />
    );
}
SignedInNoPremiumWithCredits.parameters = {
    session: signedInNoPremiumWithCreditsSession,
    route: {
        path: LINKS.Home,
    },
};

export function SignedInPremiumNoCredits() {
    return (
        <DashboardView display="page" />
    );
}
SignedInPremiumNoCredits.parameters = {
    session: signedInPremiumNoCreditsSession,
    route: {
        path: LINKS.Home,
    },
};

export function SignedInPremiumWithCredits() {
    return (
        <DashboardView display="page" />
    );
}
SignedInPremiumWithCredits.parameters = {
    session: signedInPremiumWithCreditsSession,
    route: {
        path: LINKS.Home,
    },
};
