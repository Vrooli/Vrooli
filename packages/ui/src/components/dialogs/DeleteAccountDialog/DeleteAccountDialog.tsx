import {
    Button,
    Checkbox,
    Dialog,
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
import { useMutation } from 'api/hooks';
import { APP_LINKS, Success, UserDeleteInput } from '@shared/consts';
import { useLocation } from '@shared/route';
import { DialogTitle, PasswordTextField, SnackSeverity } from 'components';
import { DeleteIcon } from '@shared/icons';
import { getCurrentUser } from 'utils/authentication';
import { userDeleteOneSchema as validationSchema } from '@shared/validation';
import { PubSub } from 'utils';
import { useFormik } from 'formik';
import { endpoints } from 'api';

const titleAria = 'delete-object-dialog-title';

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
    const [, setLocation] = useLocation();

    const { id, name } = useMemo(() => getCurrentUser(session), [session]);

    const [deleteAccount] = useMutation<Success, UserDeleteInput, 'userDeleteOne'>(...endpoints.user().deleteOne);
    const formik = useFormik({
        initialValues: {
            password: '',
            deletePublicData: false,
        },
        validationSchema,
        onSubmit: (values) => {
            if (!id) {
                PubSub.get().publishSnack({ messageKey: 'NoUserIdFound', severity: SnackSeverity.Error });
                return;
            }
            mutationWrapper<Success, UserDeleteInput>({
                mutation: deleteAccount,
                input: values,
                successCondition: (data) => data.success,
                successMessage: () => ({ key: 'AccountDeleteSuccess' }),
                onSuccess: () => {
                    setLocation(APP_LINKS.Home);
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
        <Dialog
            open={isOpen}
            onClose={() => { close(); }}
            aria-labelledby={titleAria}
            sx={{
                zIndex
            }}
        >
            <DialogTitle
                ariaLabel={titleAria}
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
                    >Delete</Button>
                </Stack>
            </DialogContent>
        </Dialog>
    )
}