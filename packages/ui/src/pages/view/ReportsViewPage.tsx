import { Box } from "@mui/material"
import { ReportsView } from "components";
import { ReportsViewPageProps } from "./types";

export const ReportsViewPage = ({
    session
}: ReportsViewPageProps) => {
    return <Box sx={{
        minHeight: '100vh',
        paddingTop: { xs: '64px', md: '80px' },
        paddingBottom: 'calc(56px + env(safe-area-inset-bottom))',
    }}>
        {/* Main view */}
        <ReportsView session={session} />
    </Box>
}