import { useEffect, useMemo, useRef, useState } from 'react';
import { IconButton, Button, useTheme, Palette, Box, Typography } from '@mui/material';
import { CloseIcon, ErrorIcon, InfoIcon, SuccessIcon, SvgComponent, WarningIcon } from '@shared/icons';
import { SnackProps } from '../types';

export enum SnackSeverity {
    Error = 'error',
    Info = 'info',
    Success = 'success',
    Warning = 'warning',
}

const severityStyle = (severity: SnackSeverity | undefined, palette: Palette) => {
    let backgroundColor: string = palette.primary.light;
    let color: string = palette.primary.contrastText;
    switch (severity) {
        case SnackSeverity.Error:
            backgroundColor = palette.error.dark;
            color = palette.error.contrastText;
            break;
        case SnackSeverity.Info:
            backgroundColor = palette.info.main;
            color = palette.info.contrastText;
            break;
        case SnackSeverity.Success:
            backgroundColor = palette.success.main;
            color = palette.success.contrastText;
            break;
        default:
            backgroundColor = palette.warning.main;
            color = palette.warning.contrastText;
            break;
    }
    return { backgroundColor, color };
}

const DURATION = 4000;

/**
 * Individual snack item in the snack stack. 
 * Look changes based on severity. 
 * Supports a button with a callback.
 */
export const Snack = ({
    buttonClicked,
    buttonText,
    data,
    handleClose,
    id,
    message,
    severity,
}: SnackProps) => {
    const { palette } = useTheme();

    const [open, setOpen] = useState<boolean>(true);
    // Timout to close the snack
    const timeoutRef = useRef<NodeJS.Timeout | null>(null);
    // Timeout starts immediately
    useEffect(() => {
        timeoutRef.current = setTimeout(() => {
            // First set to close
            setOpen(false);
            // Then start a second timeout to remove from the stack
            timeoutRef.current = setTimeout(() => {
                handleClose();
            }, 400);
        }, DURATION);
        return () => {
            if (timeoutRef.current) {
                clearTimeout(timeoutRef.current);
            }
        }
    }, [handleClose]);

    useEffect(() => {
        // Log snack errors if in development
        if (process.env.NODE_ENV === 'development' && data) {
            if (severity === SnackSeverity.Error) console.error('Snack data', data);
            else console.info('Snack data', data);
        }
    }, [data, severity])

    const Icon = useMemo<SvgComponent>(() => {
        switch (severity) {
            case SnackSeverity.Error:
                return ErrorIcon;
            case SnackSeverity.Info:
                return InfoIcon;
            case SnackSeverity.Success:
                return SuccessIcon;
            default:
                return WarningIcon;
        }
    }, [severity]);

    return (
        <Box sx={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            maxWidth: { xs: '100%', sm: '600px' },
            // Scrolls out of view when closed
            transform: open ? 'translateX(0)' : 'translateX(-150%)',
            transition: 'transform 0.4s ease-in-out',
            padding: '8px 16px',
            borderRadius: '12px',
            ...severityStyle(severity, palette),
        }}>
            {/* Icon */}
            <Icon fill="white" />
            {/* Message */}
            <Typography variant="body1" sx={{ color: 'white', marginLeft: '4px' }}>
                {message}
            </Typography>
            {/* Button */}
            {buttonText && buttonClicked && (
                <Button
                    variant="text"
                    sx={{ color: 'black', marginLeft: '16px', padding: '4px', border: '1px solid black', borderRadius: '8px' }}
                    onClick={buttonClicked}
                >
                    {buttonText}
                </Button>
            )}
            {/* Close icon */}
            <IconButton onClick={handleClose}>
                <CloseIcon />
            </IconButton>
        </Box>
    )
}