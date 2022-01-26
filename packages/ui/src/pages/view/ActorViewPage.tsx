import { Box } from "@mui/material"
import { ActorView } from "components";
import { ActorViewPageProps } from "./types";

export const ActorViewPage = ({
    session
}: ActorViewPageProps) => {

    return (
        <Box pt="10vh" sx={{minHeight: '88vh'}}>
            <ActorView session={session} />
        </Box>
    )
}