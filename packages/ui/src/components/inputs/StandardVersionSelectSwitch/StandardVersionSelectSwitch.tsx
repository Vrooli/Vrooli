import { Box, Stack, Typography, useTheme } from '@mui/material';
import { EditIcon as CustomIcon, LinkIcon } from '@shared/icons';
import { ColorIconButton } from 'components/buttons/ColorIconButton/ColorIconButton';
import { SelectOrCreateDialog } from 'components/dialogs/selectOrCreates';
import { useCallback, useMemo, useState } from 'react';
import { noSelect } from 'styles';
import { StandardVersionSelectSwitchProps } from '../types';

const grey = {
    400: '#BFC7CF',
    800: '#2F3A45',
};

export function StandardVersionSelectSwitch({
    session,
    selected,
    onChange,
    disabled,
    zIndex,
    ...props
}: StandardVersionSelectSwitchProps) {
    const { palette } = useTheme();

    // Create dialog
    const [isCreateDialogOpen, setIsCreateDialogOpen] = useState<boolean>(false);
    const openCreateDialog = useCallback(() => { setIsCreateDialogOpen(true) }, [setIsCreateDialogOpen]);
    const closeCreateDialog = useCallback(() => { setIsCreateDialogOpen(false); }, [setIsCreateDialogOpen]);

    const handleClick = useCallback((ev: React.MouseEvent<any>) => {
        if (disabled) return;
        // If using custom data, remove standardVersion data
        if (Boolean(selected)) {
            onChange(null);
        } 
        // Otherwise, open dialog to select standardVersion
        else {
            openCreateDialog();
            ev.preventDefault();
        }
    }, [disabled, onChange, openCreateDialog, selected]);

    const Icon = useMemo(() => Boolean(selected) ? LinkIcon : CustomIcon, [selected]);

    return (
        <>
            {/* Popup for adding/connecting a new standardVersion */}
            <SelectOrCreateDialog
                isOpen={isCreateDialogOpen}
                handleAdd={onChange as any}
                handleClose={closeCreateDialog}
                objectType="StandardVersion"
                session={session}
                zIndex={zIndex+1}
            />
            {/* Main component */}
            <Stack direction="row" spacing={1}>
                <Typography variant="h6" sx={{ ...noSelect }}>Standard:</Typography>
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
                            position: 'absolute',
                            top: 0,
                            padding: '4px',
                            transition: 'transform 150ms cubic-bezier(0.4, 0, 0.2, 1)',
                            transform: `translateX(${Boolean(selected) ? '24' : '0'}px)`,
                        }}>
                            <Icon width='30px' height='30px' fill="white" />
                        </ColorIconButton>
                    </Box>
                    {/* Input */}
                    <input
                        type="checkbox"
                        checked={Boolean(selected)}
                        readOnly
                        disabled={disabled}
                        aria-label="custom-standard-toggle"
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
                <Typography variant="h6" sx={{ ...noSelect }}>{selected ? selected.root.name : 'Custom'}</Typography>
            </Stack>
        </>
    )
}
