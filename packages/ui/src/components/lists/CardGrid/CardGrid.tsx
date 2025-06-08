import Box from "@mui/material/Box";
import { useTheme } from "@mui/material";
import { useMemo } from "react";
import { type CardGridProps } from "../types.js";

export function CardGrid({
    children,
    disableMargin,
    minWidth,
    sx,
}: CardGridProps) {
    const { breakpoints } = useTheme();

    const boxStyle = useMemo(function boxStyleMemo() {
        return {
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
        } as const;
    }, [breakpoints, disableMargin, minWidth, sx]);

    return (
        <Box sx={boxStyle}>
            {children}
        </Box>
    );
}
