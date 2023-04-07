import { Box } from '@mui/material';
import { SlideContainerProps } from '../types';

export const SlideContainer = ({
    id,
    children,
    sx,
}: SlideContainerProps) => {
    return (
        <Box
            id={id}
            key={id}
            sx={{
                position: 'relative',
                overflow: 'hidden',
                scrollSnapAlign: 'start',
                ...sx
            }}
        >
            {children}
        </Box>
    );
}