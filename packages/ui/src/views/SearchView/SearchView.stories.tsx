import { loggedOutSession, signedInNoPremiumNoCreditsSession, signedInNoPremiumWithCreditsSession, signedInPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../__test/storybookConsts.js";
import { PageContainer } from "../../components/Page/Page.js";
import { SearchView } from "./SearchView.js";

export default {
    title: "Views/SearchView",
    component: SearchView,
};

export function LoggedOut() {
    return (
        <PageContainer>
            <SearchView display="page" />
        </PageContainer>
    );
}
LoggedOut.parameters = {
    session: loggedOutSession,
};

export function SignedInNoPremiumNoCredits() {
    return (
        <PageContainer>
            <SearchView display="page" />
        </PageContainer>
    );
}
SignedInNoPremiumNoCredits.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

export function SignedInNoPremiumWithCredits() {
    return (
        <PageContainer>
            <SearchView display="page" />
        </PageContainer>
    );
}
SignedInNoPremiumWithCredits.parameters = {
    session: signedInNoPremiumWithCreditsSession,
};

export function SignedInPremiumNoCredits() {
    return (
        <PageContainer>
            <SearchView display="page" />
        </PageContainer>
    );
}
SignedInPremiumNoCredits.parameters = {
    session: signedInPremiumNoCreditsSession,
};

export function SignedInPremiumWithCredits() {
    return (
        <PageContainer>
            <SearchView display="page" />
        </PageContainer>
    );
}
SignedInPremiumWithCredits.parameters = {
    session: signedInPremiumWithCreditsSession,
}; 
