import { Box, useTheme } from "@mui/material";
import { CardGridProps } from "../types";

export const CardGrid = ({
    children,
    disableMargin,
    minWidth,
    sx,
}: CardGridProps) => {
    const { breakpoints } = useTheme();

    return (
        <Box sx={{
            display: "grid",
            gridTemplateColumns: `repeat(auto-fit, minmax(${minWidth}px, 1fr))`,
            alignItems: "stretch",
            gap: 2,
            margin: disableMargin ? 0 : 2,
            borderRadius: 2,
            [breakpoints.down("sm")]: {
                gap: 0,
                margin: 0,
            },
            ...(sx ?? {}),
        }}>
            {children}
        </Box>
    );
};
