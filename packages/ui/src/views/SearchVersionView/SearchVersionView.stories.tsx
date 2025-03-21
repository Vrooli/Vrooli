import { loggedOutSession, signedInNoPremiumNoCreditsSession, signedInNoPremiumWithCreditsSession, signedInPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../__test/storybookConsts.js";
import { PageContainer } from "../../components/Page/Page.js";
import { SearchVersionView } from "./SearchVersionView.js";

export default {
    title: "Views/SearchVersionView",
    component: SearchVersionView,
};

export function LoggedOut() {
    return (
        <PageContainer>
            <SearchVersionView display="page" />
        </PageContainer>
    );
}
LoggedOut.parameters = {
    session: loggedOutSession,
};

export function SignedInNoPremiumNoCredits() {
    return (
        <PageContainer>
            <SearchVersionView display="page" />
        </PageContainer>
    );
}
SignedInNoPremiumNoCredits.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

export function SignedInNoPremiumWithCredits() {
    return (
        <PageContainer>
            <SearchVersionView display="page" />
        </PageContainer>
    );
}
SignedInNoPremiumWithCredits.parameters = {
    session: signedInNoPremiumWithCreditsSession,
};

export function SignedInPremiumNoCredits() {
    return (
        <PageContainer>
            <SearchVersionView display="page" />
        </PageContainer>
    );
}
SignedInPremiumNoCredits.parameters = {
    session: signedInPremiumNoCreditsSession,
};

export function SignedInPremiumWithCredits() {
    return (
        <PageContainer>
            <SearchVersionView display="page" />
        </PageContainer>
    );
}
SignedInPremiumWithCredits.parameters = {
    session: signedInPremiumWithCreditsSession,
}; 
