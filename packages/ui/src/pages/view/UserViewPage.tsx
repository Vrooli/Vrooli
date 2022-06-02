import { Box } from "@mui/material"
import { UserView } from "components";
import { UserViewPageProps } from "./types";

export const UserViewPage = ({
    session
}: UserViewPageProps) => {
    return (
        <Box sx={{
            minHeight: '100vh',
            paddingTop: { xs: '64px', md: '80px' },
        }}>
            <UserView session={session} zIndex={200} />
        </Box>
    )
}