import { TopBar } from "components";
import { PremiumViewProps } from "../types";

export const PremiumView = ({
    display = 'page',
    session,
}: PremiumViewProps) => {

    // TODO convert MaxObjects to list of limit increases 
    return (
        <>
            <TopBar
                display={display}
                onClose={() => { }}
            />
        </>
    )
}