import { Box } from "@mui/material"
import { OrganizationView } from "components";
import { OrganizationViewPageProps } from "./types";

export const OrganizationViewPage = ({
    session
}: OrganizationViewPageProps) => {

    return (
        <Box pt="10vh" sx={{minHeight: '88vh'}}>
            <OrganizationView session={session} />
        </Box>
    )
}