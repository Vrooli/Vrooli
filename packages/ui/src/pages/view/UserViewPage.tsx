import { PageContainer, UserView } from "components";
import { UserViewPageProps } from "./types";

export const UserViewPage = ({
    session
}: UserViewPageProps) => {
    return (
        <PageContainer sx={{ paddingLeft: 0, paddingRight: 0 }}>
            <UserView session={session} zIndex={200} />
        </PageContainer>
    )
}