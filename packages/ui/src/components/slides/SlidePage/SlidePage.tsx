import { Box } from "@mui/material";
import { SlidePageProps } from "../types";

export const SlidePage = ({
    children,
    id,
    sx,
}: SlidePageProps) => {
    return (
        <Box
            id={id}
            sx={{
                scrollBehavior: "smooth",
                ...(sx || {}),
            }}
        >
            {children}
        </Box>
    );
};
