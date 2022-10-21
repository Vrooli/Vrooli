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
                filter: disabled ? 'grayscale(100%) brightness(0.5)' : 'none',
                '&:hover': {
                    backgroundColor: background,
                    filter: disabled ? 'grayscale(100%) brightness(0.5)' : 'brightness(1.1)',
                },
            }}
        >
            {children}
        </IconButton>
    )
}