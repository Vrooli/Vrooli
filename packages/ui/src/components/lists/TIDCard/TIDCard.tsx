import { Box, Button, Typography, useTheme } from "@mui/material";
import { TIDCardProps } from "../types";

/**
 * A card with a title, description, and icon
 */
export const TIDCard = ({
    buttonText,
    description,
    key,
    Icon,
    onClick,
    title,
}: TIDCardProps) => {
    const { palette } = useTheme();

    return (
        <Box
            key={key}
            onClick={onClick}
            sx={{
                width: "100%",
                boxShadow: 8,
                padding: 1,
                borderRadius: 2,
                cursor: "pointer",
                background: palette.background.paper,
                "&:hover": {
                    filter: "brightness(1.1)",
                    boxShadow: 12,
                },
                display: "flex",
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
                    <Typography variant='h6' component='div'>
                        {title}
                    </Typography>
                    <Typography variant='body2' color={palette.background.textSecondary}>
                        {description}
                    </Typography>
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
        </Box>
    );
};
