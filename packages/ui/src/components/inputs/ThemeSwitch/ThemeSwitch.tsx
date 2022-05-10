import { Box, IconButton, Stack, Typography } from '@mui/material';
import { useCallback, useMemo } from 'react';
import { ThemeSwitchProps } from '../types';
import { noSelect } from 'styles';
import {
    DarkMode as DarkIcon,
    LightMode as LightIcon,
} from '@mui/icons-material';
import { Pubs } from 'utils';

export function ThemeSwitch({
    theme,
    onChange,
}: ThemeSwitchProps) {

    const handleChange = useCallback(() => {
        const newTheme = theme === 'light' ? 'dark' : 'light';
        PubSub.publish(Pubs.Theme, newTheme)
        onChange(newTheme);
    }, [onChange, theme]);

    const isDark = useMemo(() => theme === 'dark', [theme]);
    const Icon = useMemo(() => isDark ? DarkIcon : LightIcon, [isDark]);
    const trackColor = useMemo(() => isDark ? '#2F3A45' : '#BFC7CF', [isDark]);

    return (
        <Stack direction="row" spacing={1} justifyContent="center">
            <Typography variant="h6" sx={{ ...noSelect }}>Theme:</Typography>
            <Box component="span" sx={{
                display: 'inline-block',
                position: 'relative',
                width: '64px',
                height: '36px',
                padding: '8px',
            }}>
                {/* Track */}
                <Box component="span" sx={{
                    backgroundColor: trackColor,
                    borderRadius: '16px',
                    width: '100%',
                    height: '65%',
                    display: 'block',
                }}>
                    {/* Thumb */}
                    <IconButton sx={{
                        backgroundColor: (t) => t.palette.secondary.main,
                        display: 'inline-flex',
                        width: '30px',
                        height: '30px',
                        position: 'absolute',
                        top: 0,
                        transition: 'transform 150ms cubic-bezier(0.4, 0, 0.2, 1)',
                        transform: `translateX(${isDark ? '24' : '0'}px)`,
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
                    checked={isDark}
                    aria-label="user-organization-toggle"
                    onClick={handleChange}
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
            <Typography variant="h6" sx={{ ...noSelect }}>{isDark ? 'Dark' : 'Light'}</Typography>
        </Stack>
    )
}
