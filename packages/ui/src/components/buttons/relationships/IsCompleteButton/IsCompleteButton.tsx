import { Stack, Tooltip, useTheme } from '@mui/material';
import { CompleteIcon } from '@shared/icons';
import { ColorIconButton } from 'components/buttons/ColorIconButton/ColorIconButton';
import { TextShrink } from 'components/text/TextShrink/TextShrink';
import { useField } from 'formik';
import { useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { commonButtonProps, commonIconProps, commonLabelProps } from '../styles';
import { IsCompleteButtonProps } from '../types';

export function IsCompleteButton({
    isEditing,
    objectType,
}: IsCompleteButtonProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [field, , helpers] = useField('isComplete');

    const isAvailable = useMemo(() => ['Project', 'Routine'].includes(objectType), [objectType]);

    const { Icon, tooltip } = useMemo(() => {
        const isComplete = field?.value;
        return {
            Icon: isComplete ? CompleteIcon : null,
            tooltip: isComplete ? `This is complete${isEditing ? '' : '. Press to mark incomplete'}` : `This is incomplete${isEditing ? '' : '. Press to mark complete'}`
        }
    }, [field?.value, isEditing]);

    const handleClick = useCallback((ev: React.MouseEvent<any>) => {
        if (!isEditing || !isAvailable) return;
        helpers.setValue(!field?.value);
    }, [isEditing, isAvailable, helpers, field?.value]);

    // If not available, return null
    if (!isAvailable || !isEditing) return null;
    // Return button with label on top
    return (
        <Stack
            direction="column"
            alignItems="center"
            justifyContent="center"
        >
            <TextShrink id="complete" sx={{ ...commonLabelProps() }}>Complete?</TextShrink>
            <Tooltip title={tooltip}>
                <ColorIconButton background={palette.primary.light} sx={{ ...commonButtonProps(isEditing, false) }} onClick={handleClick}>
                    {Icon && <Icon {...commonIconProps()} />}
                </ColorIconButton>
            </Tooltip>
        </Stack>
    )
}