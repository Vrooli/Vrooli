import { type ListObject, uuid } from "@local/shared";
import { loggedOutSession, signedInNoPremiumNoCreditsSession, signedInNoPremiumWithCreditsSession, signedInPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../__test/storybookConsts.js";
import { PageContainer } from "../../components/Page/Page.js";
import { StatsObjectView } from "./StatsObjectView.js";

export default {
    title: "Views/StatsObjectView",
    component: StatsObjectView,
};

const mockObject: ListObject = {
    __typename: "Standard" as const,
    bookmarks: 0,
    createdAt: new Date().toISOString(),
    id: uuid(),
    isDeleted: false,
    score: 0,
    translatedName: "Mock Object",
    you: {
        __typename: "StandardYou" as const,
        canBookmark: true,
        canDelete: false,
        canReact: true,
        canRead: true,
        isViewed: false,
    },
};

function handleObjectUpdate(object: ListObject): void {
    console.log("Object updated:", object);
}

function handleClose(): void {
    console.log("Dialog closed");
}

export function LoggedOut() {
    return (
        <PageContainer>
            <StatsObjectView
                display="Dialog"
                handleObjectUpdate={handleObjectUpdate}
                isOpen={true}
                object={mockObject}
                onClose={handleClose}
            />
        </PageContainer>
    );
}
LoggedOut.parameters = {
    session: loggedOutSession,
};

export function SignedInNoPremiumNoCredits() {
    return (
        <PageContainer>
            <StatsObjectView
                display="Dialog"
                handleObjectUpdate={handleObjectUpdate}
                isOpen={true}
                object={mockObject}
                onClose={handleClose}
            />
        </PageContainer>
    );
}
SignedInNoPremiumNoCredits.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

export function SignedInNoPremiumWithCredits() {
    return (
        <PageContainer>
            <StatsObjectView
                display="Dialog"
                handleObjectUpdate={handleObjectUpdate}
                isOpen={true}
                object={mockObject}
                onClose={handleClose}
            />
        </PageContainer>
    );
}
SignedInNoPremiumWithCredits.parameters = {
    session: signedInNoPremiumWithCreditsSession,
};

export function SignedInPremiumNoCredits() {
    return (
        <PageContainer>
            <StatsObjectView
                display="Dialog"
                handleObjectUpdate={handleObjectUpdate}
                isOpen={true}
                object={mockObject}
                onClose={handleClose}
            />
        </PageContainer>
    );
}
SignedInPremiumNoCredits.parameters = {
    session: signedInPremiumNoCreditsSession,
};

export function SignedInPremiumWithCredits() {
    return (
        <PageContainer>
            <StatsObjectView
                display="Dialog"
                handleObjectUpdate={handleObjectUpdate}
                isOpen={true}
                object={mockObject}
                onClose={handleClose}
            />
        </PageContainer>
    );
}
SignedInPremiumWithCredits.parameters = {
    session: signedInPremiumWithCreditsSession,
}; 
