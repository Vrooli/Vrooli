import { Box } from "@mui/material"
import { UserView } from "components";
import { UserViewPageProps } from "./types";

export const UserViewPage = ({
    session
}: UserViewPageProps) => {

    return (
        <Box pt="10vh" sx={{minHeight: '88vh'}}>
            <UserView session={session} />
        </Box>
    )
}