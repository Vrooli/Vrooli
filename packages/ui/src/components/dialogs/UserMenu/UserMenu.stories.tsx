import { useEffect } from "react";
import { loggedOutSession, multipleUsersSession, signedInNoPremiumNoCreditsSession, signedInNoPremiumWithCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { ELEMENT_IDS } from "../../../utils/consts.js";
import { PubSub } from "../../../utils/pubsub.js";
import { PageContainer } from "../../Page/Page.js";
import { UserMenu } from "./UserMenu.js";

const OPEN_DELAY_MS = 1000;

export default {
    title: "Components/dialogs/UserMenu",
    component: UserMenu,
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

export function SignedInNoPremiumNoCredits() {
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
SignedInNoPremiumNoCredits.parameters = {
    session: signedInNoPremiumNoCreditsSession,
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

export function SignedInPremiumWithCredits() {
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
SignedInPremiumWithCredits.parameters = {
    session: signedInPremiumWithCreditsSession,
};

export function MultipleUsers() {
    useEffect(function publishUserMenuOpen() {
        setTimeout(() => {
            PubSub.get().publish("menu", { id: ELEMENT_IDS.UserMenu, isOpen: true });
        }, OPEN_DELAY_MS);
    }, []);
}
MultipleUsers.parameters = {
    session: multipleUsersSession,
};
