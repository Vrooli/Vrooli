import { APP_LINKS } from "@local/shared";
import { Box } from "@mui/material"
import { SubroutineView } from "components/views/SubroutineView/SubroutineView";
import { useLocation, useRoute } from "wouter";
import { RunRoutinePageProps } from "./types";

export const RunRoutinePage = ({
    session
}: RunRoutinePageProps) => {
    const [, setLocation] = useLocation();
    // Get URL params
    const [hasNotStarted, params] = useRoute(`${APP_LINKS.Run}/:id`);
    const [hasStarted, params2] = useRoute(`${APP_LINKS.Run}/:routineId/:subroutineId`);

    console.log('PARAMSSS', hasStarted, hasNotStarted, params, params2);

    return (
        <Box pt="10vh" sx={{ minHeight: '88vh' }}>
            <Box sx={{
                margin: 'auto',
            }}>
                {/* Display title, description, instructions, etc. */}
                {/* Display  */}
                <SubroutineView
                    hasPrevious={false}
                    hasNext={false}
                    session={session}
                />
            </Box>
        </Box>
    )
}