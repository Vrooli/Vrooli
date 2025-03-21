import { loggedOutSession, signedInNoPremiumNoCreditsSession, signedInNoPremiumWithCreditsSession, signedInPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../__test/storybookConsts.js";
import { PageContainer } from "../../components/Page/Page.js";
import { CalendarView } from "./CalendarView.js";

export default {
    title: "Views/CalendarView",
    component: CalendarView,
};

export function LoggedOut() {
    return (
        <PageContainer>
            <CalendarView display="page" />
        </PageContainer>
    );
}
LoggedOut.parameters = {
    session: loggedOutSession,
};

export function SignedInNoPremiumNoCredits() {
    return (
        <PageContainer>
            <CalendarView display="page" />
        </PageContainer>
    );
}
SignedInNoPremiumNoCredits.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

export function SignedInNoPremiumWithCredits() {
    return (
        <PageContainer>
            <CalendarView display="page" />
        </PageContainer>
    );
}
SignedInNoPremiumWithCredits.parameters = {
    session: signedInNoPremiumWithCreditsSession,
};

export function SignedInPremiumNoCredits() {
    return (
        <PageContainer>
            <CalendarView display="page" />
        </PageContainer>
    );
}
SignedInPremiumNoCredits.parameters = {
    session: signedInPremiumNoCreditsSession,
};

export function SignedInPremiumWithCredits() {
    return (
        <PageContainer>
            <CalendarView display="page" />
        </PageContainer>
    );
}
SignedInPremiumWithCredits.parameters = {
    session: signedInPremiumWithCreditsSession,
};
