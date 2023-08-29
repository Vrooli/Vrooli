import { IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { TitleProps } from "../types";

export const Title = ({
    help,
    Icon,
    options,
    sxs,
    title,
    titleComponent,
    variant = "header",
}: TitleProps) => {
    const { palette } = useTheme();

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
                    fontSize: {
                        xs: variant === "header" ? "1.75rem" : "1.5rem",
                        sm: variant === "header" ? "2rem" : "1.75rem",
                        md: variant === "header" ? "2.5rem" : "2rem",
                    },
                    ...sxs?.text,
                }}
            >{title}</Typography>}
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
