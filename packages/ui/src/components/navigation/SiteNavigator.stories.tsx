import { Session, SessionUser, uuid } from "@local/shared";
import { PageContainer } from "../Page/Page.js";
import { SiteNavigator } from "./SiteNavigator.js";

const OPEN_DELAY_MS = 1000;

export default {
    title: "Components/navigation/SiteNavigator",
    component: SiteNavigator,
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
            <SiteNavigator />
        </PageContainer>
    );
}
LoggedOut.parameters = {
    session: loggedOutSession,
};

export function SignedInNoPremiumOrCredits() {
    return (
        <PageContainer>
            <SiteNavigator />
        </PageContainer>
    );
}
SignedInNoPremiumOrCredits.parameters = {
    session: signedInNoPremiumOrCreditsSession,
};

export function SignedInNoPremiumWithCredits() {
    return (
        <PageContainer>
            <SiteNavigator />
        </PageContainer>
    );
}
SignedInNoPremiumWithCredits.parameters = {
    session: signedInNoPremiumWithCreditsSession,
};

export function SignedInPremium() {
    return (
        <PageContainer>
            <SiteNavigator />
        </PageContainer>
    );
}
SignedInPremium.parameters = {
    session: signedInPremiumSession,
};  
