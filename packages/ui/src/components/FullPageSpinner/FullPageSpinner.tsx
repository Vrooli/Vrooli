import { Box, CircularProgress } from "@mui/material"

export const FullPageSpinner = () => {
    return (
        <Box sx={{
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: 'translate(-50%, -50%)',
            zIndex: 100000,
        }}>
            <CircularProgress size={100} />
        </Box>
    )
}