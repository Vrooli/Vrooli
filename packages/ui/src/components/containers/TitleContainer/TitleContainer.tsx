// Used to display popular/search results of a particular object type
import { Box, CircularProgress, IconButton, Link, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { useTranslation } from "react-i18next";
import { clickSize } from "styles";
import { TitleContainerProps } from "../types";

export function TitleContainer({
    id,
    helpKey,
    helpVariables,
    Icon,
    titleKey,
    titleVariables,
    onClick,
    loading = false,
    tooltip = "",
    options = [],
    sx,
    children,
}: TitleContainerProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    return (
        <Tooltip placement="bottom" title={tooltip}>
            <Stack direction="column" id={id} display="flex" justifyContent="center">
                {/* Title container */}
                <Box
                    onClick={(e) => { onClick && onClick(e); }}
                    sx={{
                        color: palette.background.textPrimary,
                        padding: 0.5,
                    }}
                >
                    {/* Title */}
                    <Stack direction="row" justifyContent="center" alignItems="center">
                        {Icon && <Icon fill={palette.background.textPrimary} style={{ width: "30px", height: "30px", marginRight: 8 }} />}
                        <Typography component="h2" variant="h4" textAlign="center">{t(titleKey, { ...titleVariables, defaultValue: titleKey })}</Typography>
                        {helpKey ? <HelpButton markdown={t(helpKey!, { ...helpVariables, defaultValue: helpKey })} /> : null}
                        {/* Display any options with a title */}
                        {options.filter(({ Icon }) => Icon).map(({ Icon, key, onClick, variables }, index) => (
                            <Tooltip key={`title-option-${index}`} title={t(key, { ...variables, defaultValue: key })}>
                                <IconButton onClick={onClick}>
                                    {Icon && <Icon fill={palette.secondary.main} style={{ width: "30px", height: "30px" }} />}
                                </IconButton>
                            </Tooltip>
                        ))}
                    </Stack>
                </Box>
                <Box
                    sx={{
                        boxShadow: 4,
                        borderRadius: { xs: 0, sm: 2 },
                        overflow: "overlay",
                        background: palette.background.paper,
                        width: "min(100%, 700px)",
                        cursor: onClick ? "pointer" : "default",
                        "&:hover": {
                            filter: `brightness(onClick ? ${102} : ${100}%)`,
                        },
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
                                    {options.map(({ key, onClick, variables }, index) => (
                                        <Link key={index} onClick={onClick} sx={{
                                            marginTop: "auto",
                                            marginBottom: "auto",
                                            marginRight: 2,
                                        }}>
                                            <Typography sx={{
                                                color: palette.mode === "light" ? palette.secondary.dark : palette.secondary.light,
                                            }}
                                            >{t(key, { ...variables, defaultValue: key })}
                                            </Typography>
                                        </Link>
                                    ))}
                                </Stack>
                            )
                        }
                    </Stack>
                </Box>
            </Stack>
        </Tooltip>
    );
}
