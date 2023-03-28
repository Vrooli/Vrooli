import { Stack } from '@mui/material';
import { SlideContentProps } from '../types';

export const SlideContent = ({
    children,
    id,
    sx,
}: SlideContentProps) => {
    return (
        <Stack
            id={id}
            direction="column"
            spacing={2}
            p={2}
            textAlign="center"
            zIndex={5}
            sx={{
                maxWidth: { xs: '100vw', sm: '90vw', md: 'min(80vw, 1000px)' },
                minHeight: '100vh',
                justifyContent: 'center',
                margin: 'auto',
                ...(sx || {}),
            }}
        >
            {children}
        </Stack>
    );
}