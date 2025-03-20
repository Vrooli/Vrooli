import { Session, SessionUser, uuid } from "@local/shared";
import { useEffect } from "react";
import { ELEMENT_IDS } from "../../../utils/consts.js";
import { PubSub } from "../../../utils/pubsub.js";
import { PageContainer } from "../../Page/Page.js";
import { UserMenu } from "./UserMenu.js";

const OPEN_DELAY_MS = 1000;

export default {
    title: "Components/dialogs/UserMenu",
    component: UserMenu,
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
    useEffect(function publishUserMenuOpen() {
        setTimeout(() => {
            PubSub.get().publish("menu", { id: ELEMENT_IDS.UserMenu, isOpen: true });
        }, OPEN_DELAY_MS);
    }, []);

    return (
        <PageContainer>
            <UserMenu />
        </PageContainer>
    );
}
LoggedOut.parameters = {
    session: loggedOutSession,
};

export function SignedInNoPremiumOrCredits() {
    useEffect(function publishUserMenuOpen() {
        setTimeout(() => {
            PubSub.get().publish("menu", { id: ELEMENT_IDS.UserMenu, isOpen: true });
        }, OPEN_DELAY_MS);
    }, []);

    return (
        <PageContainer>
            <UserMenu />
        </PageContainer>
    );
}
SignedInNoPremiumOrCredits.parameters = {
    session: signedInNoPremiumOrCreditsSession,
};

export function SignedInNoPremiumWithCredits() {
    useEffect(function publishUserMenuOpen() {
        setTimeout(() => {
            PubSub.get().publish("menu", { id: ELEMENT_IDS.UserMenu, isOpen: true });
        }, OPEN_DELAY_MS);
    }, []);

    return (
        <PageContainer>
            <UserMenu />
        </PageContainer>
    );
}
SignedInNoPremiumWithCredits.parameters = {
    session: signedInNoPremiumWithCreditsSession,
};

export function SignedInPremium() {
    useEffect(function publishUserMenuOpen() {
        setTimeout(() => {
            PubSub.get().publish("menu", { id: ELEMENT_IDS.UserMenu, isOpen: true });
        }, OPEN_DELAY_MS);
    }, []);

    return (
        <PageContainer>
            <UserMenu />
        </PageContainer>
    );
}
SignedInPremium.parameters = {
    session: signedInPremiumSession,
};  
