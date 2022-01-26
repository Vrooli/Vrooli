import { Box } from "@mui/material"
import { StandardView } from "components";
import { StandardViewPageProps } from "./types";

export const StandardViewPage = ({
    session
}: StandardViewPageProps) => {

    return (
        <Box pt="10vh" sx={{minHeight: '88vh'}}>
            <StandardView session={session} />
        </Box>
    )
}