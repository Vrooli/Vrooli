import { Box } from "@mui/material";
import { pagePaddingBottom } from "styles";
import { PageContainerProps } from "../types";

/**
 * Container which can be wrapped around most pages to provide a consistent layout.
 */
export const PageContainer = ({
    children,
    sx,
}: PageContainerProps) => {
    return (
        <Box id="page" sx={{
            minWidth: "100%",
            minHeight: "100%",
            width: "min(100%, 700px)",
            margin: "auto",
            paddingBottom: pagePaddingBottom,
            paddingLeft: { xs: 0, sm: "max(1em, calc(15% - 75px))" },
            paddingRight: { xs: 0, sm: "max(1em, calc(15% - 75px))" },
            ...(sx ?? {}),
        }}>
            {children}
        </Box>
    );
};
