import { TopBar } from "components/navigation/TopBar/TopBar"
import { Subheader } from "components/text/Subheader/Subheader"
import { AboutViewProps } from "views/types"

export const AboutView = ({
    display = 'page',
}: AboutViewProps) => {
    return (
        <>
            <TopBar
                display={display}
                onClose={() => { }}
                titleData={{
                    titleKey: 'AboutUs',
                }}
            />
            <Subheader title="Leader/Developer - Matt Halloran" />
            {/* TODO finish */}
        </>
    )
}