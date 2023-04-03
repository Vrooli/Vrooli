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
                width: '100%',
                boxShadow: 8,
                padding: 1,
                borderRadius: 2,
                cursor: 'pointer',
                background: palette.background.paper,
                '&:hover': {
                    filter: 'brightness(1.1)',
                    boxShadow: 12,
                },
            }}>
            {/* Left of card is icon */}
            {Icon && <Box sx={{
                float: 'left',
                width: '75px',
                height: '100%',
                padding: 2,
            }}>
                <Box sx={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    width: '100%',
                    height: '100%',
                }}>
                    <Icon width={'50px'} height={'50px'} fill={palette.background.textPrimary} />
                </Box>
            </Box>}
            {/* Right of card is title and description */}
            <Box sx={{
                float: 'right',
                width: Icon ? 'calc(100% - 75px)' : '100%',
                height: '100%',
                padding: '1rem',
                display: 'contents',
            }}>
                <Typography variant='h6' component='div'>
                    {title}
                </Typography>
                <Typography variant='body2' color={palette.background.textSecondary}>
                    {description}
                </Typography>
                {/* Bottom of card is button */}
                <Button
                    size='small'
                    sx={{
                        marginTop: 2,
                        marginLeft: 'auto',
                        display: 'flex',
                    }}
                >{buttonText}</Button>
            </Box>
        </Box>
    )
}