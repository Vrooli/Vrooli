import { TopBar } from "components/navigation/TopBar/TopBar";
import { TermsViewProps } from "../types";

export const TermsView = ({
    display = 'page',
}: TermsViewProps) => {
    return <>
        <TopBar
            display={display}
            onClose={() => { }}
            titleData={{
                titleKey: 'Terms',
            }}
        />
    </>;
};