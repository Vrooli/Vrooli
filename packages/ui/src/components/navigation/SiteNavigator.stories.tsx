import { loggedOutSession, signedInNoPremiumNoCreditsSession, signedInNoPremiumWithCreditsSession, signedInPremiumWithCreditsSession } from "../../__test/storybookConsts.js";
import { PageContainer } from "../Page/Page.js";
import { SiteNavigator } from "./SiteNavigator.js";

export default {
    title: "Components/navigation/SiteNavigator",
    component: SiteNavigator,
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
    session: signedInNoPremiumNoCreditsSession,
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
    session: signedInPremiumWithCreditsSession,
};  
