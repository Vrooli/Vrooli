import { Subheader, TopBar } from "components"
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