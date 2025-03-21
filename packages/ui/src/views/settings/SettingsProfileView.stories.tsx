import { loggedOutSession, signedInNoPremiumNoCreditsSession, signedInNoPremiumWithCreditsSession, signedInPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../__test/storybookConsts.js";
import { PageContainer } from "../../components/Page/Page.js";
import { SettingsProfileView } from "./SettingsProfileView.js";

export default {
    title: "Views/Settings/SettingsProfileView",
    component: SettingsProfileView,
};

export function LoggedOut() {
    return (
        <PageContainer>
            <SettingsProfileView display="page" />
        </PageContainer>
    );
}
LoggedOut.parameters = {
    session: loggedOutSession,
};

export function SignedInNoPremiumNoCredits() {
    return (
        <PageContainer>
            <SettingsProfileView display="page" />
        </PageContainer>
    );
}
SignedInNoPremiumNoCredits.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

export function SignedInNoPremiumWithCredits() {
    return (
        <PageContainer>
            <SettingsProfileView display="page" />
        </PageContainer>
    );
}
SignedInNoPremiumWithCredits.parameters = {
    session: signedInNoPremiumWithCreditsSession,
};

export function SignedInPremiumNoCredits() {
    return (
        <PageContainer>
            <SettingsProfileView display="page" />
        </PageContainer>
    );
}
SignedInPremiumNoCredits.parameters = {
    session: signedInPremiumNoCreditsSession,
};

export function SignedInPremiumWithCredits() {
    return (
        <PageContainer>
            <SettingsProfileView display="page" />
        </PageContainer>
    );
}
SignedInPremiumWithCredits.parameters = {
    session: signedInPremiumWithCreditsSession,
};
