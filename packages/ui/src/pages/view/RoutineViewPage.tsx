import { Box } from "@mui/material";
import { RoutineView } from 'components';
import { RoutineViewPageProps } from "./types";

export const RoutineViewPage = ({
    session,
}: RoutineViewPageProps) => {
    return (
        <Box pt="10vh" sx={{ minHeight: '88vh' }}>
            <RoutineView session={session} />
        </Box>
    )
}