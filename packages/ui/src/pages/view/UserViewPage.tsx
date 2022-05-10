import { Box, useTheme } from "@mui/material"
import { UserView } from "components";
import { UserViewPageProps } from "./types";

export const UserViewPage = ({
    session
}: UserViewPageProps) => {
    const { breakpoints } = useTheme();

    return (
        <Box sx={{
            minHeight: '100vh',
            [breakpoints.up('md')]: {
                paddingTop: '10vh',
                minHeight: '88vh',
            },
        }}>
            <UserView session={session} />
        </Box>
    )
}