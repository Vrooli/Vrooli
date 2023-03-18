import { TopBar } from "components/navigation/TopBar/TopBar";
import { PrivacyPolicyViewProps } from "../types";

export const PrivacyPolicyView = ({
    display = 'page',
}: PrivacyPolicyViewProps) => {
    return <>
        <TopBar
            display={display}
            onClose={() => { }}
            titleData={{
                titleKey: 'Privacy',
            }}
        />
    </>;
};