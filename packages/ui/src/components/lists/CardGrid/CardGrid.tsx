import { Box } from "@mui/material";
import { CardGridProps } from "../types";

export const CardGrid = ({
    children,
    disableMargin,
    minWidth,
    sx,
}: CardGridProps) => {
    return (
        <Box sx={{
            display: "grid",
            gridTemplateColumns: `repeat(auto-fit, minmax(${minWidth}px, 1fr))`,
            alignItems: "stretch",
            gap: 2,
            margin: disableMargin ? 0 : 2,
            borderRadius: 2,
            ...(sx ?? {}),
        }}>
            {children}
        </Box>
    );
};
