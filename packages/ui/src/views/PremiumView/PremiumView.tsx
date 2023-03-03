import { useTopBar } from "utils";
import { PremiumViewProps } from "../types";

export const PremiumView = ({
    display = 'page',
    session,
}: PremiumViewProps) => {

    const TopBar = useTopBar({
        display,
        session,
    })

    // TODO convert MaxObjects to list of limit increases 
    return (
        <>
        {TopBar}
        </>
    )
}