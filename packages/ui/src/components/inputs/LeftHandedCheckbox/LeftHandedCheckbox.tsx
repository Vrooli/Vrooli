import { Checkbox, Stack, Typography } from '@mui/material';
import { useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { noSelect } from 'styles';
import { getCookieIsLeftHanded } from 'utils/cookies';
import { PubSub } from 'utils/pubsub';
import { LeftHandedCheckboxProps } from '../types';

/**
 * Updates the font size of the entire app
 */
export function LeftHandedCheckbox({
    session,
}: LeftHandedCheckboxProps) {
    const { t } = useTranslation();

    const [isLeftHanded, setIsLeftHanded] = useState(getCookieIsLeftHanded() ?? false);

    const handleToggle = useCallback(() => {
        setIsLeftHanded(!isLeftHanded);
        PubSub.get().publishIsLeftHanded(!isLeftHanded);
    }, [isLeftHanded]);

    return (
        <Stack direction="row" spacing={1} justifyContent="center" alignItems="center">
            <Typography variant="body1" sx={{
                ...noSelect,
                marginRight: 'auto',
            }}>
                {t(`LeftHandedQuestion`)}
            </Typography>
            <Checkbox
                id="leftHandedCheckbox"
                size="medium"
                color='secondary'
                checked={isLeftHanded}
                onChange={handleToggle}
            />
        </Stack>
    )
}
