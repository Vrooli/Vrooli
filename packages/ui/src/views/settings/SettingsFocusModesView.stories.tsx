import { loggedOutSession, signedInNoPremiumNoCreditsSession, signedInNoPremiumWithCreditsSession, signedInPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../__test/storybookConsts.js";
import { PageContainer } from "../../components/Page/Page.js";
import { SettingsFocusModesView } from "./SettingsFocusModesView.js";

export default {
    title: "Views/Settings/SettingsFocusModesView",
    component: SettingsFocusModesView,
};

export function LoggedOut() {
    return (
        <PageContainer>
            <SettingsFocusModesView display="page" />
        </PageContainer>
    );
}
LoggedOut.parameters = {
    session: loggedOutSession,
};

export function SignedInNoPremiumNoCredits() {
    return (
        <PageContainer>
            <SettingsFocusModesView display="page" />
        </PageContainer>
    );
}
SignedInNoPremiumNoCredits.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

export function SignedInNoPremiumWithCredits() {
    return (
        <PageContainer>
            <SettingsFocusModesView display="page" />
        </PageContainer>
    );
}
SignedInNoPremiumWithCredits.parameters = {
    session: signedInNoPremiumWithCreditsSession,
};

export function SignedInPremiumNoCredits() {
    return (
        <PageContainer>
            <SettingsFocusModesView display="page" />
        </PageContainer>
    );
}
SignedInPremiumNoCredits.parameters = {
    session: signedInPremiumNoCreditsSession,
};

export function SignedInPremiumWithCredits() {
    return (
        <PageContainer>
            <SettingsFocusModesView display="page" />
        </PageContainer>
    );
}
SignedInPremiumWithCredits.parameters = {
    session: signedInPremiumWithCreditsSession,
};
