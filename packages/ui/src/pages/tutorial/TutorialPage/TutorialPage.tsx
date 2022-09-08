import { Box,  useTheme } from '@mui/material';
import { useLocation } from '@shared/route';

export const TutorialPage = () => {
    const { palette } = useTheme();

    return (
        <Box
            sx={{
                minHeight: '100vh',
                width: '100vw',
                display: 'flex',
                justifyContent: 'center',
                alignItems: 'center',
                textAlign: 'center',
                animation: 'gradient 15s ease infinite',
                overflowX: 'hidden',
                paddingTop: { xs: '64px', md: '80px' },
            }}
        >
            <h1>Tutorial page</h1>
        </Box>
    )
}