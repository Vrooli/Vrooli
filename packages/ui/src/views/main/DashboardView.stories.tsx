import { LINKS } from "@vrooli/shared";
import { signedInNoPremiumNoCreditsSession, signedInNoPremiumWithCreditsSession, signedInPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../__test/storybookConsts.js";
import { DashboardView } from "./DashboardView.js";

export default {
    title: "Views/Main/DashboardView",
    component: DashboardView,
};

export function SignedInNoPremiumNoCredits() {
    return (
        <DashboardView display="Page" />
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
        <DashboardView display="Page" />
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
        <DashboardView display="Page" />
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
        <DashboardView display="Page" />
    );
}
SignedInPremiumWithCredits.parameters = {
    session: signedInPremiumWithCreditsSession,
    route: {
        path: LINKS.Home,
    },
};
