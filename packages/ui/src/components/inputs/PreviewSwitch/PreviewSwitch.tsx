import { Box, Stack, Typography, useTheme } from '@mui/material';
import { useCallback, useMemo } from 'react';
import { PreviewSwitchProps } from '../types';
import { noSelect } from 'styles';
import { BuildIcon, SvgComponent, VisibleIcon } from '@shared/icons';
import { ColorIconButton } from 'components/buttons';

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
    const Icon = useMemo<SvgComponent>(() => isPreviewOn ? VisibleIcon : BuildIcon, [isPreviewOn]);

    const handleClick = useCallback((ev: React.MouseEvent<any>) => {
        if (disabled) return;
        onChange(!isPreviewOn);
    }, [disabled, isPreviewOn, onChange]);

    return (
        <Stack direction="row" spacing={1} justifyContent="center" sx={{ ...(sx ?? {}) }}>
            <Typography variant="h6" sx={{ ...noSelect }}>{isPreviewOn ? 'Preview' : 'Schema'}</Typography>
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
                    <ColorIconButton background={palette.secondary.main} sx={{
                        display: 'inline-flex',
                        width: '30px',
                        height: '30px',
                        padding: 0,
                        position: 'absolute',
                        top: 0,
                        transition: 'transform 150ms cubic-bezier(0.4, 0, 0.2, 1)',
                        transform: `translateX(${isPreviewOn ? '24' : '0'}px)`,
                    }}>
                        <Icon fill="white" width="80%" height="80%" />
                    </ColorIconButton>
                </Box>
                {/* Input */}
                <input
                    type="checkbox"
                    checked={isPreviewOn}
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
