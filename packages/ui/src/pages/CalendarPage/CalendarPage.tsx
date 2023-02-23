import { CalendarView, PageContainer, PageTitle } from "components";
import { CalendarPageProps } from "pages/types";

export function CalendarPage({
    session,
}: CalendarPageProps) {

    return (
        <PageContainer>
            <PageTitle titleKey='Calendar' session={session} />
            <CalendarView session={session} />
        </PageContainer>
    )
}