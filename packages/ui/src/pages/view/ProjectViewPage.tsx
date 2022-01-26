import { Box } from "@mui/material"
import { ProjectView } from "components";
import { ProjectViewPageProps } from "./types";

export const ProjectViewPage = ({
    session
}: ProjectViewPageProps) => {

    return (
        <Box pt="10vh" sx={{minHeight: '88vh'}}>
            <ProjectView session={session} />
        </Box>
    )
}