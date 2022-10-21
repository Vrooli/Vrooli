import { IconButton } from '@mui/material';
import { ColorIconButtonProps } from '../types';

/**
 * IconButton with a custom color
 */
export const ColorIconButton = ({
    background,
    children,
    disabled,
    sx,
    ...props
}: ColorIconButtonProps) => {
    return (
        <IconButton
            {...props}
            sx={{
                ...(sx ?? {}),
                backgroundColor: background,
                pointerEvents: disabled ? 'none' : 'auto',
                filter: disabled ? 'grayscale(1) opacity(0.5)' : 'none',
                '&:hover': {
                    backgroundColor: background,
                    filter: disabled ? 'grayscale(1) opacity(0.5)' : 'brightness(1.2)',
                },
            }}
        >
            {children}
        </IconButton>
    )
}