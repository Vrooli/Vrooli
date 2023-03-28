import { Box } from "@mui/material";
import { GlossyContainerProps } from "../types";

/**
 * A semi-transparent container with a glossy (blur) effect
 */
export const GlossyContainer = ({
    children,
    sx,
    ...props
}: GlossyContainerProps) => {
    return (
        <Box
            sx={{
                boxShadow: '0px 0px 6px #040505',
                backgroundColor: 'rgba(255,255,255,0.3)',
                backdropFilter: 'blur(24px)',
                borderRadius: '0.5rem',
                padding: 2,
                ...sx
            }}
            {...props}
        >
            {children}
        </Box>
    );
};