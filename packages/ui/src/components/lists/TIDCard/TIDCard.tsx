import { Box, Button, Typography, useTheme } from "@mui/material";
import { WarningIcon } from "icons";
import { TIDCardProps } from "../types";

/**
 * A card with a title, description, and icon
 */
export const TIDCard = ({
    buttonText,
    description,
    Icon,
    id,
    onClick,
    title,
    warning,
    ...props
}: TIDCardProps) => {
    const { breakpoints, palette } = useTheme();

    return (
        <Box
            {...props}
            id={id}
            onClick={onClick}
            sx={{
                width: "100%",
                boxShadow: { xs: 0, sm: 4 },
                padding: 1,
                borderRadius: { xs: 0, sm: 2 },
                cursor: "pointer",
                background: palette.background.paper,
                "&:hover": {
                    filter: "brightness(1.05)",
                },
                display: "flex",
                [breakpoints.down("sm")]: {
                    borderBottom: `1px solid ${palette.divider}`,
                },
            }}>
            {/* Left of card is icon */}
            {Icon && <Box sx={{
                width: "75px",
                height: "100%",
                padding: 2,
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
            }}>
                <Icon width={"50px"} height={"50px"} fill={palette.background.textPrimary} />
            </Box>}
            {/* Right of card is title and description */}
            <Box sx={{
                flexGrow: 1,
                height: "100%",
                padding: "1rem",
                display: "flex",
                flexDirection: "column",
                justifyContent: "space-between",
            }}>
                <Box>
                    <Typography variant='h6' component='div' sx={{ overflowWrap: "anywhere" }}>
                        {title}
                    </Typography>
                    <Typography variant='body2' color={palette.background.textSecondary} sx={{ overflowWrap: "anywhere" }}>
                        {description}
                    </Typography>
                    {warning && (
                        <Box sx={{ display: "flex", alignItems: "center", color: palette.warning.main, marginTop: 1 }}>
                            <WarningIcon style={{ fontSize: 20, marginRight: "8px" }} />
                            <Typography variant="body2">{warning}</Typography>
                        </Box>
                    )}
                </Box>
                {/* Bottom of card is button */}
                <Button
                    size='small'
                    sx={{
                        marginTop: 2,
                        marginLeft: "auto",
                        alignSelf: "flex-end",
                    }}
                    variant="text"
                >{buttonText}</Button>
            </Box>
        </Box >
    );
};
