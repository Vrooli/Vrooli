import { Box, IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { useWindowSize } from "hooks/useWindowSize";
import { fontSizeToPixels } from "utils/display/textTools";
import { TitleProps } from "../types";

export const Title = ({
    adornments,
    help,
    Icon,
    options,
    sxs,
    title,
    titleComponent,
    variant = "header",
}: TitleProps) => {
    const { breakpoints, palette, typography } = useTheme();

    // Determine title size based on variant and screen size
    const matchesXS = useWindowSize(({ width }) => width <= breakpoints.values.xs);
    const matchesSM = useWindowSize(({ width }) => width <= breakpoints.values.sm);
    let fontSize: string;
    if (matchesXS) {
        fontSize = variant === "header" ? "1.75rem" : "1.5rem";
    } else if (matchesSM) {
        fontSize = variant === "header" ? "2rem" : "1.75rem";
    } else {
        fontSize = variant === "header" ? "2.5rem" : "2rem";
    }

    return (
        <Stack
            direction="row"
            justifyContent="center"
            alignItems="center"
            sx={{
                padding: 2,
                wordBreak: "break-word",
                ...sxs?.stack,
            }}
        >
            {/* Icon */}
            {Icon && <Icon fill={palette.background.textPrimary} style={{ width: "30px", height: "30px", marginRight: 8 }} />}
            {/* Title */}
            {titleComponent ?? <Typography
                component={variant === "header" ? "h1" : "h2"}
                variant={variant === "header" ? "h3" : "h4"}
                textAlign="center"
                sx={{
                    fontSize,
                    ...sxs?.text,
                }}
            >{title}</Typography>}
            {/* Adornments */}
            {adornments && adornments.map((Adornment) => (
                <Box sx={{
                    width: fontSizeToPixels(fontSize ?? "1rem") * Number(typography[variant === "header" ? "h3" : "h4"].lineHeight ?? "1.5"),
                    height: fontSizeToPixels(fontSize ?? "1rem") * Number(typography[variant === "header" ? "h3" : "h4"].lineHeight ?? "1.5"),
                }}>
                    {Adornment}
                </Box>
            ))}
            {/* Help button */}
            {help && help.length > 0 ? <HelpButton
                markdown={help}
                sx={{
                    width: variant === "header" ? "40px" : "30px",
                    height: variant === "header" ? "40px" : "30px",
                }}
            /> : null}
            {/* Additional options */}
            {options?.filter(({ Icon }) => Icon).map(({ Icon, label, onClick }, index) => (
                <Tooltip key={`title-option-${index}`} title={label}>
                    <IconButton onClick={onClick}>
                        {Icon && <Icon fill={palette.secondary.main} style={{ width: "30px", height: "30px" }} />}
                    </IconButton>
                </Tooltip>
            ))}
        </Stack>
    );
};
