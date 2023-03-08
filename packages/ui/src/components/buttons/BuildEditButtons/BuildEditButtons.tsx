import { Box, Grid } from '@mui/material';
import { BuildEditButtonsProps } from '../types';
import { GridSubmitButtons } from 'components/buttons';

export const BuildEditButtons = ({
    canSubmitMutate,
    canCancelMutate,
    errors,
    handleCancel,
    handleSubmit,
    isAdding,
    isEditing,
    loading,
}: BuildEditButtonsProps) => {

    if (!isEditing) return null;
    return (
        <Box sx={{
            alignItems: 'center',
            background: 'transparent',
            display: 'flex',
            justifyContent: 'center',
            position: 'absolute',
            zIndex: 2,
            bottom: 0,
            right: 0,
            paddingBottom: 'calc(16px + env(safe-area-inset-bottom))',
            paddingLeft: 'calc(16px + env(safe-area-inset-left))',
            paddingRight: 'calc(16px + env(safe-area-inset-right))',
            height: 'calc(64px + env(safe-area-inset-bottom))',
        }}>
            <Grid container spacing={1} sx={{ width: 'min(100%, 600px)' }}>
                <GridSubmitButtons
                    disabledCancel={loading || !canCancelMutate}
                    disabledSubmit={loading || !canSubmitMutate}
                    display="page"
                    errors={errors}
                    isCreate={isAdding}
                    onCancel={handleCancel}
                    onSubmit={handleSubmit}
                />
            </Grid>
        </Box>
    )
};