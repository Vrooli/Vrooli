import { Box, IconButton, Stack, Typography, useTheme } from '@mui/material';
import { useCallback, useMemo } from 'react';
import { PreviewSwitchProps } from '../types';
import { noSelect } from 'styles';
import {
    Build as BuildIcon,
    Visibility as PreviewIcon,
} from '@mui/icons-material';

const grey = {
    400: '#BFC7CF',
    800: '#2F3A45',
};

export function PreviewSwitch({
    disabled = false,
    isPreviewOn,
    onChange,
    sx,
}: PreviewSwitchProps) {
    const { palette } = useTheme();
    const Icon = useMemo(() => isPreviewOn ? PreviewIcon : BuildIcon, [isPreviewOn]);

    const handleClick = useCallback((ev: React.MouseEvent<any>) => {
        if (disabled) return;
        onChange(!isPreviewOn);
    }, [disabled, isPreviewOn, onChange]);

    return (
        <Stack direction="row" spacing={1} justifyContent="center" sx={{ ...(sx ?? {}) }}>
            <Typography variant="h6" sx={{ ...noSelect }}>{isPreviewOn ? 'Preview mode' : 'Build mode'}</Typography>
            <Box component="span" sx={{
                display: 'inline-block',
                position: 'relative',
                width: '64px',
                height: '36px',
                padding: '8px',
            }}>
                {/* Track */}
                <Box component="span" sx={{
                    backgroundColor: palette.mode === 'dark' ? grey[800] : grey[400],
                    borderRadius: '16px',
                    width: '100%',
                    height: '65%',
                    display: 'block',
                }}>
                    {/* Thumb */}
                    <IconButton sx={{
                        backgroundColor: palette.secondary.main,
                        display: 'inline-flex',
                        width: '30px',
                        height: '30px',
                        position: 'absolute',
                        top: 0,
                        transition: 'transform 150ms cubic-bezier(0.4, 0, 0.2, 1)',
                        transform: `translateX(${isPreviewOn ? '24' : '0'}px)`,
                    }}>
                        <Icon sx={{
                            position: 'absolute',
                            display: 'block',
                            fill: 'white',
                            borderRadius: '8px',
                        }} />
                    </IconButton>
                </Box>
                {/* Input */}
                <input
                    type="checkbox"
                    checked={isPreviewOn}
                    readOnly
                    aria-label="build-preview-toggle"
                    onClick={handleClick}
                    style={{
                        position: 'absolute',
                        width: '100%',
                        height: '100%',
                        top: '0',
                        left: '0',
                        opacity: '0',
                        zIndex: '1',
                        margin: '0',
                        cursor: 'pointer',
                    }} />
            </Box >
        </Stack>
    )
}
