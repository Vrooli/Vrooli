// Used to display popular/search results of a particular object type
import { Box, CircularProgress, Link, Stack, Typography, useTheme } from "@mui/material";
import { Title } from "components/text/Title/Title";
import { TitleProps } from "components/text/types";
import { clickSize } from "styles";
import { TitleContainerProps } from "../types";

export function TitleContainer({
    children,
    help,
    Icon,
    id,
    loading = false,
    onClick,
    options = [],
    sx,
    title,
    zIndex,
}: TitleContainerProps) {
    const { palette } = useTheme();

    return (
        <Stack direction="column" id={id} display="flex" justifyContent="center" alignItems="center">
            <Title
                help={help}
                Icon={Icon}
                options={options.filter(({ Icon }) => Icon) as TitleProps["options"]}
                title={title}
                variant="subheader"
                zIndex={zIndex}
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
                    {/* Links */}
                    {
                        options.length > 0 && (
                            <Stack direction="row" sx={{
                                ...clickSize,
                                justifyContent: "end",
                            }}
                            >
                                {options.map(({ label, onClick }, index) => (
                                    <Link key={index} onClick={onClick} sx={{
                                        cursor: "pointer",
                                        marginTop: "auto",
                                        marginBottom: "auto",
                                        marginRight: 2,
                                    }}>
                                        <Typography sx={{
                                            color: palette.mode === "light" ? palette.secondary.dark : palette.secondary.light,
                                        }}
                                        >{label}
                                        </Typography>
                                    </Link>
                                ))}
                            </Stack>
                        )
                    }
                </Stack>
            </Box>
        </Stack>
    );
}
