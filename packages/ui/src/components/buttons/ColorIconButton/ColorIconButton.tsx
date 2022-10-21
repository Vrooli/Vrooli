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
                backgroundColor: background,
                pointerEvents: disabled ? 'none' : 'auto',
                filter: disabled ? 'grayscale(100%) brightness(0.5)' : 'none',
                transition: 'filter 0.2s ease-in-out',
                '&:hover': {
                    backgroundColor: background,
                    filter: disabled ? 'grayscale(100%) brightness(0.5)' : 'brightness(1.2)',
                },
                ...(sx ?? {}),
            }}
        >
            {children}
        </IconButton>
    )
}