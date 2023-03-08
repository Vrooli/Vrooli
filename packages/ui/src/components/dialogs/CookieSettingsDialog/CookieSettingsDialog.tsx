/**
 * Displays all search options for an organization
 */
import {
    Button,
    Dialog,
    Grid,
    Stack,
    Switch,
    Typography,
    useTheme
} from '@mui/material';
import { CookieSettingsDialogProps } from '../types';
import { useFormik } from 'formik';
import { DialogTitle, HelpButton, LargeDialog } from 'components';
import { CookiePreferences, setCookiePreferences } from 'utils/cookies';
import { useTranslation } from 'react-i18next';

const titleId = 'cookie-settings-dialog-title';
const strictlyNecessaryUses = ['Authentication'] as const;
const functionalUses = ['Theme', 'Font Size'] as const;

//TODO not fully translated
export const CookieSettingsDialog = ({
    handleClose,
    isOpen,
}: CookieSettingsDialogProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const setPreferences = (preferences: CookiePreferences) => {
        console.log('in setcookiepreferences', preferences);
        // Set preference in local storage
        setCookiePreferences(preferences);
        // Close dialog
        handleClose(preferences);
    }
    const onCancel = () => { handleClose(); }

    const formik = useFormik({
        initialValues: {
            strictlyNecessary: true,
            performance: false,
            functional: true,
            targeting: false,
        },
        onSubmit: (values) => {
            console.log('setting cookie preferences', values);
            setPreferences(values);
        }
    });

    const handleAcceptAllCookies = () => {
        const preferences: CookiePreferences = {
            strictlyNecessary: true,
            performance: true,
            functional: true,
            targeting: true,
        };
        setPreferences(preferences);
    }

    return (
        <LargeDialog
            id="cookie-settings-dialog"
            isOpen={isOpen}
            onClose={onCancel}
            titleId={titleId}
            zIndex={1234}
        >
            {/* Title */}
            <DialogTitle
                id={titleId}
                title={t(`CookieSettings`)}
                onClose={onCancel}
            />
            <form onSubmit={formik.handleSubmit} style={{ padding: '16px' }}>
                {/* Description of cookies and why we use them */}
                <Typography variant="body1" mb={4}>
                    {t(`CookieSettings`)}
                </Typography>
                {/* Strictly necessary */}
                <Stack direction="column" spacing={1} sx={{ marginBottom: 2 }}>
                    <Stack direction="row" marginRight="auto" alignItems="center">
                        <Typography component="h2" variant="h5" textAlign="center">{t(`CookieStrictlyNecessary`)}</Typography>
                        <HelpButton markdown={t(`CookieStrictlyNecessaryDescription`)} />
                        <Switch
                            checked={formik.values.strictlyNecessary}
                            onChange={formik.handleChange}
                            name="strictlyNecessary"
                            // Can't turn off
                            disabled
                            sx={{
                                position: 'absolute',
                                right: '16px',
                            }}
                        />
                    </Stack>
                    <Typography variant="body1">
                        Current uses: {strictlyNecessaryUses.join(', ')}
                    </Typography>
                </Stack>
                {/* Performance */}
                <Stack direction="column" spacing={1} sx={{ marginBottom: 2 }}>
                    <Stack direction="row" marginRight="auto" alignItems="center">
                        <Typography component="h2" variant="h5" textAlign="center">{t('Performance')}</Typography>
                        <HelpButton markdown={t(`CookiePerformanceDescription`)} />
                        <Switch
                            checked={formik.values.performance}
                            onChange={formik.handleChange}
                            name="performance"
                            sx={{
                                position: 'absolute',
                                right: '16px',
                            }}
                        />
                    </Stack>
                    <Typography variant="body1">
                        Current uses: <b>None</b>
                    </Typography>
                </Stack>
                {/* Functional */}
                <Stack direction="column" spacing={1} sx={{ marginBottom: 2 }}>
                    <Stack direction="row" marginRight="auto" alignItems="center">
                        <Typography component="h2" variant="h5" textAlign="center">{t('Functional')}</Typography>
                        <HelpButton markdown={t(`CookieFunctionalDescription`)} />
                        <Switch
                            checked={formik.values.functional}
                            onChange={formik.handleChange}
                            name="functional"
                            sx={{
                                position: 'absolute',
                                right: '16px',
                            }}
                        />
                    </Stack>
                    <Typography variant="body1">
                        Current uses: {functionalUses.join(', ')}
                    </Typography>
                </Stack>
                {/* Targeting */}
                <Stack direction="column" spacing={1} sx={{ marginBottom: 4 }}>
                    <Stack direction="row" marginRight="auto" alignItems="center">
                        <Typography component="h2" variant="h5" textAlign="center">{t('Targeting')}</Typography>
                        <HelpButton markdown={t(`CookieTargetingDescription`)} />
                        <Switch
                            checked={formik.values.targeting}
                            onChange={formik.handleChange}
                            name="targeting"
                            sx={{
                                position: 'absolute',
                                right: '16px',
                            }}
                        />
                    </Stack>
                    <Typography variant="body1">
                        Current uses: <b>None</b>
                    </Typography>
                </Stack>
                {/* Search/Cancel buttons */}
                <Grid container spacing={1} sx={{
                    paddingBottom: 'env(safe-area-inset-bottom)',
                }}>
                    <Grid item xs={4}>
                        <Button
                            fullWidth
                            type="submit"
                        >{t(`Confirm`)}</Button>
                    </Grid>
                    <Grid item xs={4}>
                        <Button
                            fullWidth
                            onClick={handleAcceptAllCookies}
                        >{t(`AcceptAll`)}</Button>
                    </Grid>
                    <Grid item xs={4}>
                        <Button
                            fullWidth
                            variant="text"
                            onClick={onCancel}
                        >{t(`Cancel`)}</Button>
                    </Grid>
                </Grid>
            </form>
        </LargeDialog>
    )
}