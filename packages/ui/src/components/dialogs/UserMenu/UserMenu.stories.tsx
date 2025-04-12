import { Button } from "@mui/material";
import { useRef } from "react";
import { loggedOutSession, multipleUsersSession, signedInNoPremiumNoCreditsSession, signedInNoPremiumWithCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { ELEMENT_IDS } from "../../../utils/consts.js";
import { PubSub } from "../../../utils/pubsub.js";
import { PageContainer } from "../../Page/Page.js";
import { UserMenu } from "./UserMenu.js";

export default {
    title: "Components/dialogs/UserMenu",
    component: UserMenu,
};

function MenuTrigger() {
    const buttonRef = useRef<HTMLButtonElement>(null);

    function handleClick() {
        if (buttonRef.current) {
            PubSub.get().publish("menu", {
                id: ELEMENT_IDS.UserMenu,
                isOpen: true,
                data: { anchorEl: buttonRef.current },
            });
        }
    }

    return (
        <Button ref={buttonRef} onClick={handleClick} variant="contained">
            Open Menu
        </Button>
    );
}

export function LoggedOut() {
    return (
        <PageContainer>
            <MenuTrigger />
            <UserMenu />
        </PageContainer>
    );
}
LoggedOut.parameters = {
    session: loggedOutSession,
};

export function SignedInNoPremiumNoCredits() {
    return (
        <PageContainer>
            <MenuTrigger />
            <UserMenu />
        </PageContainer>
    );
}
SignedInNoPremiumNoCredits.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

export function SignedInNoPremiumWithCredits() {
    return (
        <PageContainer>
            <MenuTrigger />
            <UserMenu />
        </PageContainer>
    );
}
SignedInNoPremiumWithCredits.parameters = {
    session: signedInNoPremiumWithCreditsSession,
};

export function SignedInPremiumWithCredits() {
    return (
        <PageContainer>
            <MenuTrigger />
            <UserMenu />
        </PageContainer>
    );
}
SignedInPremiumWithCredits.parameters = {
    session: signedInPremiumWithCreditsSession,
};

export function MultipleUsers() {
    return (
        <PageContainer>
            <MenuTrigger />
            <UserMenu />
        </PageContainer>
    );
}
MultipleUsers.parameters = {
    session: multipleUsersSession,
};
