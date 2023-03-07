import { Box, Stack, Typography, useTheme } from '@mui/material';
import { useCallback, useMemo } from 'react';
import { ThemeSwitchProps } from '../types';
import { noSelect } from 'styles';
import { PubSub } from 'utils';
import { DarkModeIcon, LightModeIcon } from '@shared/icons';
import { ColorIconButton } from 'components/buttons';
import { useTranslation } from 'react-i18next';

export function ThemeSwitch({
    showText = true,
    theme,
    onChange,
}: ThemeSwitchProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const handleChange = useCallback(() => {
        const newTheme = theme === 'light' ? 'dark' : 'light';
        PubSub.get().publishTheme(newTheme)
        onChange(newTheme);
    }, [onChange, theme]);

    const isDark = useMemo(() => theme === 'dark', [theme]);
    const Icon = useMemo(() => isDark ? DarkModeIcon : LightModeIcon, [isDark]);
    const trackColor = useMemo(() => isDark ? '#2F3A45' : '#BFC7CF', [isDark]);

    return (
        <Stack direction="row" spacing={1} justifyContent="center" alignItems="center">
            <Typography variant="body1" sx={{
                ...noSelect,
                marginRight: 'auto',
            }}>
                {t(`Theme`)}: {theme === 'light' ? t(`Light`) : t(`Dark`)}
            </Typography>
            <Box component="span" sx={{
                display: 'inline-block',
                position: 'relative',
                width: '80px',
                height: '40px',
                padding: '8px',
            }}>
                {/* Track */}
                <Box component="span" sx={{
                    display: 'flex',
                    backgroundColor: trackColor,
                    borderRadius: '16px',
                    width: '100%',
                    height: '100%',
                }}>
                    {/* Thumb */}
                    <ColorIconButton background={palette.secondary.main} sx={{
                        display: 'inline-flex',
                        width: '40px',
                        height: '40px',
                        position: 'absolute',
                        top: 0,
                        transition: 'transform 150ms cubic-bezier(0.4, 0, 0.2, 1)',
                        transform: `translateX(${isDark ? '24' : '0'}px)`,
                    }}>
                        <Icon fill={palette.secondary.contrastText} />
                    </ColorIconButton>
                </Box>
                {/* Input */}
                <input
                    type="checkbox"
                    checked={isDark}
                    aria-label="user-organization-toggle"
                    onChange={handleChange}
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
