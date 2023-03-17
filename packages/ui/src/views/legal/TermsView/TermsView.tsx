import { TopBar } from "components/navigation/TopBar/TopBar";
import { TermsViewProps } from "../types";

export const TermsView = ({
    display = 'page',
    session,
}: TermsViewProps) => {
    return <>
        <TopBar
            display={display}
            onClose={() => { }}
            session={session}
            titleData={{
                titleKey: 'Terms',
            }}
        />
    </>;
};