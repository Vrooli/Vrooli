import { Session, SessionUser, uuid } from "@local/shared";
import { PageContainer } from "../../components/Page/Page.js";
import { ProView } from "./ProView.js";

export default {
    title: "Views/ProView",
    component: ProView,
};

const loggedOutSession: Partial<Session> = {
    isLoggedIn: false,
    users: [],
};

const signedInNoPremiumOrCreditsSession: Partial<Session> = {
    isLoggedIn: true,
    users: [{
        credits: "0",
        hasPremium: false,
        id: uuid(),
    }] as SessionUser[],
};

const signedInNoPremiumWithCreditsSession: Partial<Session> = {
    isLoggedIn: true,
    users: [{
        credits: "1234567",
        hasPremium: false,
        id: uuid(),
    }] as SessionUser[],
};

const signedInPremiumSession: Partial<Session> = {
    isLoggedIn: true,
    users: [{
        credits: "12345678912",
        hasPremium: true,
        id: uuid(),
    }] as SessionUser[],
};

export function LoggedOut() {
    return (
        <PageContainer>
            <ProView display="page" />
        </PageContainer>
    );
}
LoggedOut.parameters = {
    docs: {
        description: {
            story: "Displays the pro view when the user is signed out.",
        },
    },
    session: loggedOutSession,
};

export function SignedInNoPremiumOrCredits() {
    return (
        <PageContainer>
            <ProView display="page" />
        </PageContainer>
    );
}
SignedInNoPremiumOrCredits.parameters = {
    docs: {
        description: {
            story: "Displays the pro view when the user is signed in and has no premium or credits.",
        },
    },
    session: signedInNoPremiumOrCreditsSession,
};

export function SignedInNoPremiumWithCredits() {
    return (
        <PageContainer>
            <ProView display="page" />
        </PageContainer>
    );
}
SignedInNoPremiumWithCredits.parameters = {
    docs: {
        description: {
            story: "Displays the pro view when the user is signed in and has no premium but has credits.",
        },
    },
    session: signedInNoPremiumWithCreditsSession,
};

export function SignedInPremium() {
    return (
        <PageContainer>
            <ProView display="page" />
        </PageContainer>
    );
}
SignedInPremium.parameters = {
    docs: {
        description: {
            story: "Displays the pro view when the user is signed in and has premium.",
        },
    },
    session: signedInPremiumSession,
};
