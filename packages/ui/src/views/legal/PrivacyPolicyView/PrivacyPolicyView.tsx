import { TopBar } from "components";
import { PrivacyPolicyViewProps } from "../types";

export const PrivacyPolicyView = ({
    display = 'page',
    session,
}: PrivacyPolicyViewProps) => {
    return <>
        <TopBar
            display={display}
            onClose={() => { }}
            session={session}
            titleData={{
                titleKey: 'Privacy',
            }}
        />
    </>;
};