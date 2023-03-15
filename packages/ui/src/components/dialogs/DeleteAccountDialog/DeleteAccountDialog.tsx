import {
    Button,
    Checkbox,
    DialogContent,
    FormControlLabel,
    Stack,
    Tooltip,
    Typography,
    useTheme
} from '@mui/material';
import { DeleteAccountDialogProps } from '../types';
import { useCallback, useMemo } from 'react';
import { mutationWrapper } from 'api/utils';
import { useCustomMutation } from 'api/hooks';
import { LINKS, Success, UserDeleteInput } from '@shared/consts';
import { useLocation } from '@shared/route';
import { DialogTitle, LargeDialog, PasswordTextField } from 'components';
import { DeleteIcon } from '@shared/icons';
import { getCurrentUser } from 'utils/authentication';
import { userDeleteOneSchema as validationSchema } from '@shared/validation';
import { PubSub } from 'utils';
import { useFormik } from 'formik';
import { userDeleteOne } from 'api/generated/endpoints/user_deleteOne';
import { useTranslation } from 'react-i18next';

const titleId = 'delete-object-dialog-title';

/**
 * Dialog for deleting your account
 * @returns 
 */
export const DeleteAccountDialog = ({
    handleClose,
    isOpen,
    session,
    zIndex,
}: DeleteAccountDialogProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { id, name } = useMemo(() => getCurrentUser(session), [session]);

    const [deleteAccount] = useCustomMutation<Success, UserDeleteInput>(userDeleteOne);
    const formik = useFormik({
        initialValues: {
            password: '',
            deletePublicData: false,
        },
        validationSchema,
        onSubmit: (values) => {
            if (!id) {
                PubSub.get().publishSnack({ messageKey: 'NoUserIdFound', severity: 'Error' });
                return;
            }
            mutationWrapper<Success, UserDeleteInput>({
                mutation: deleteAccount,
                input: values,
                successCondition: (data) => data.success,
                successMessage: () => ({ key: 'AccountDeleteSuccess' }),
                onSuccess: () => {
                    setLocation(LINKS.Home);
                    close(true);
                },
                errorMessage: () => ({ key: 'AccountDeleteFail' }),
                onError: () => {
                    close(false);
                }
            })
        },
    });

    const close = useCallback((wasDeleted?: boolean) => {
        formik.resetForm();
        handleClose(wasDeleted ?? false);
    }, [formik, handleClose]);

    return (
        <LargeDialog
            id="delete-account-dialog"
            isOpen={isOpen}
            onClose={() => { close(); }}
            titleId={titleId}
            zIndex={zIndex}
        >
            <DialogTitle
                id={titleId}
                title={`Delete "${name}"`}
                onClose={() => { close() }}
            />
            <DialogContent>
                <Stack direction="column" spacing={2} mt={2}>
                    <Typography variant="h6">Are you absolutely certain you want to delete the account of "{name}"?</Typography>
                    <Typography variant="body1" sx={{ color: palette.background.textSecondary, paddingBottom: 3 }}>This action cannot be undone.</Typography>
                    <Typography variant="h6">Enter your password to confirm.</Typography>
                    <PasswordTextField
                        fullWidth
                        id="password"
                        name="password"
                        autoComplete="current-password"
                        value={formik.values.password}
                        onBlur={formik.handleBlur}
                        onChange={formik.handleChange}
                        error={formik.touched.password && Boolean(formik.errors.password)}
                        helperText={formik.touched.password ? formik.errors.password : null}
                    />
                    {/* Is internal checkbox */}
                    <Tooltip placement={'top'} title="If checked, all public data owned by your account will be deleted. Please consider transferring and/or exporting anything important if you decide to choose this.">
                        <FormControlLabel
                            label='Delete public data'
                            control={
                                <Checkbox
                                    id='delete-public-data'
                                    size="small"
                                    name='deletePublicData'
                                    color='secondary'
                                    checked={formik.values.deletePublicData}
                                    onChange={formik.handleChange}
                                />
                            }
                        />
                    </Tooltip>
                    <Button
                        disabled={formik.isSubmitting || !formik.isValid}
                        startIcon={<DeleteIcon />}
                        color="secondary"
                        onClick={() => { formik.submitForm() }}
                    >{t('Delete')}</Button>
                </Stack>
            </DialogContent>
        </LargeDialog>
    )
}