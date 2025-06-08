// Used to display popular/search results of a particular object type
import Box from "@mui/material/Box";
import CircularProgress from "@mui/material/CircularProgress";
import Stack from "@mui/material/Stack";
import { useTheme } from "@mui/material/styles";
import { Title } from "../text/Title.js";
import { type TitleProps } from "../text/types.js";
import { type TitleContainerProps } from "./types.js";

export function TitleContainer({
    children,
    help,
    iconInfo,
    id,
    loading = false,
    options = [],
    sx,
    title,
}: TitleContainerProps) {
    const { palette } = useTheme();

    return (
        <Stack direction="column" id={id} display="flex" justifyContent="center" alignItems="center">
            <Title
                help={help}
                iconInfo={iconInfo}
                options={options.filter(({ iconInfo }) => iconInfo) as TitleProps["options"]}
                title={title}
                variant="subsection"
            />
            <Box
                sx={{
                    borderRadius: { xs: 0, sm: 2 },
                    overflow: "overlay",
                    background: palette.background.paper,
                    width: "min(100%, 700px)",
                    ...sx,
                }}
            >
                {/* Main content */}
                <Stack direction="column">
                    <Box sx={{
                        ...(loading ? {
                            minHeight: "min(300px, 25vh)",
                            display: "flex",
                            justifyContent: "center",
                            alignItems: "center",
                        } : {
                            display: "block",
                        }),
                    }}>
                        {loading ? <CircularProgress color="secondary" /> : children}
                    </Box>
                </Stack>
            </Box>
        </Stack>
    );
}
