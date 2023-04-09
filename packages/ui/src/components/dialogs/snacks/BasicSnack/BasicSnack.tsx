import { Box, Button, IconButton, Palette, Typography, useTheme } from '@mui/material';
import { CloseIcon, ErrorIcon, InfoIcon, SuccessIcon, SvgComponent, WarningIcon } from '@shared/icons';
import { useEffect, useMemo, useRef, useState } from 'react';
import { BasicSnackProps } from '../types';

export enum SnackSeverity {
    Error = 'Error',
    Info = 'Info',
    Success = 'Success',
    Warning = 'Warning',
}

const severityStyle = (severity: SnackSeverity | `${SnackSeverity}` | undefined, palette: Palette) => {
    let backgroundColor: string = palette.primary.light;
    let color: string = palette.primary.contrastText;
    switch (severity) {
        case 'Error':
            backgroundColor = palette.error.dark;
            color = palette.error.contrastText;
            break;
        case 'Info':
            backgroundColor = palette.info.main;
            color = palette.info.contrastText;
            break;
        case 'Success':
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

/**
 * Basic snack item in the snack stack. 
 * Look changes based on severity. 
 * Supports a button with a callback.
 */
export const BasicSnack = ({
    buttonClicked,
    buttonText,
    data,
    duration,
    handleClose,
    id,
    message,
    severity,
}: BasicSnackProps) => {
    const { palette } = useTheme();

    const [open, setOpen] = useState<boolean>(true);
    // Timout to close the snack, if not persistent
    const timeoutRef = useRef<NodeJS.Timeout | null>(null);
    // Timeout starts immediately
    useEffect(() => {
        if (duration === 'persist') return;
        timeoutRef.current = setTimeout(() => {
            // First set to close
            setOpen(false);
            // Then start a second timeout to remove from the stack
            timeoutRef.current = setTimeout(() => {
                handleClose();
            }, 400);
        }, duration ?? 5000);
        return () => {
            if (timeoutRef.current) {
                clearTimeout(timeoutRef.current);
            }
        }
    }, [duration, handleClose]);

    useEffect(() => {
        // Log snack errors if in development
        if (import.meta.env.DEV && data) {
            if (severity === 'Error') console.error('Snack data', data);
            else console.info('Snack data', data);
        }
    }, [data, severity])

    const Icon = useMemo<SvgComponent>(() => {
        switch (severity) {
            case 'Error':
                return ErrorIcon;
            case 'Info':
                return InfoIcon;
            case 'Success':
                return SuccessIcon;
            default:
                return WarningIcon;
        }
    }, [severity]);

    return (
        <Box sx={{
            display: 'flex',
            pointerEvents: 'auto',
            justifyContent: 'space-between',
            alignItems: 'center',
            maxWidth: { xs: '100%', sm: '600px' },
            // Scrolls out of view when closed
            transform: open ? 'translateX(0)' : 'translateX(-150%)',
            transition: 'transform 0.4s ease-in-out',
            padding: '8px 16px',
            borderRadius: 2,
            boxShadow: 8,
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