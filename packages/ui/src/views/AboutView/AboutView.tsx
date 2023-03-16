import { TopBar } from "components/navigation/TopBar/TopBar"
import { Subheader } from "components/text/Subheader/Subheader"
import { AboutViewProps } from "views/types"

export const AboutView = ({
    display = 'page',
    session,
}: AboutViewProps) => {
    return (
        <>
            <TopBar
                display={display}
                onClose={() => { }}
                session={session}
                titleData={{
                    titleKey: 'AboutUs',
                }}
            />
            <Subheader title="Leader/Developer - Matt Halloran" />
            {/* TODO finish */}
        </>
    )
}