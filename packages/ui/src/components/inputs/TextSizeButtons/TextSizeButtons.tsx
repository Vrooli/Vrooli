import { Stack, Tooltip, Typography, useTheme } from '@mui/material';
import { useCallback, useState } from 'react';
import { TextSizeButtonsProps } from '../types';
import { noSelect } from 'styles';
import { BumpModerateIcon } from '@shared/icons';
import { ColorIconButton } from 'components/buttons';
import { useTranslation } from 'react-i18next';
import { PubSub } from 'utils';
import { getCookieFontSize } from 'utils/cookies';

const smallestFontSize = 10;
const largestFontSize = 20;

export function TextSizeButtons({
    session,
}: TextSizeButtonsProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [size, setSize] = useState(getCookieFontSize() ?? 14);

    const handleShrink = useCallback(() => {
        if (size > smallestFontSize) {
            setSize(size - 1);
            PubSub.get().publishFontSize(size - 1);
        }
    }, [size]);

    const handleGrow = useCallback(() => {
        if (size < largestFontSize) {
            setSize(size + 1);
            PubSub.get().publishFontSize(size + 1);
        }
    }, [size]);

    return (
        <Stack direction="row" spacing={1} justifyContent="center" alignItems="center">
            <Typography variant="body1" sx={{
                ...noSelect,
                marginRight: 'auto',
            }}>
                {t(`TextSize`)}: {size}
            </Typography>
            <Stack direction="row" spacing={0} sx={{ paddingTop: 1, paddingBottom: 1 }}>
                <Tooltip placement="top" title="Smaller">
                    <ColorIconButton
                        aria-label='shrink text'
                        background={palette.secondary.main}
                        onClick={handleShrink}
                        sx={{
                            borderRadius: '12px 0 0 12px',
                            borderRight: `1px solid ${palette.secondary.contrastText}`,
                            height: '48px',
                        }}>
                        <BumpModerateIcon style={{ transform: 'rotate(180deg)' }} />
                    </ColorIconButton>
                </Tooltip>
                <Tooltip placement="top" title="Larger">
                    <ColorIconButton
                        aria-label='grow text'
                        background={palette.secondary.main}
                        onClick={handleGrow}
                        sx={{
                            borderRadius: '0 12px 12px 0',
                            height: '48px',
                        }}>
                        <BumpModerateIcon />
                    </ColorIconButton>
                </Tooltip>
            </Stack>
        </Stack>
    )
}
