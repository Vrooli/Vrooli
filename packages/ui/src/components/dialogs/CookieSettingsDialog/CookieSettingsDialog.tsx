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
import { DialogTitle, HelpButton } from 'components';
import { CookiePreferences, setCookiePreferences } from 'utils/cookies';

const titleAria = 'cookie-settings-dialog-title';

const strictlyNecessaryText = `These cookies are necessary for our application to function properly and cannot be turned off. These cookies do not store any personally identifiable information.`;
const strictlyNecessaryUses = ['Authentication'];

const performanceText = `These cookies allow us to count visits and traffic sources so we can measure and improve the performance of our site. They help us to know which pages are the most and least popular and see how visitors move around the site, which helps us optimize your experience. All information these cookies collect is aggregated and therefore anonymous. If you do not allow these cookies we will not be able to use your data in this way.`;

const functionalText = `These cookies are used to enhance the functionality of our site. They allow us to remember your preferences (e.g. your choice of language or region) and provide enhanced, more personal features. If you do not allow these cookies then some or all of these features may not function properly.`;
const functionalUses = ['Theme', 'Font Size'];

const targetingText = `These cookies may be set through our site by our advertising partners. They may be used by those companies to build a profile of your interests and show you relevant ads on other sites. They do not store directly personal information, but are based on uniquely identifying your browser and internet device. If you do not allow these cookies, you will experience less targeted advertising.`;


export const CookieSettingsDialog = ({
    handleClose,
    isOpen,
}: CookieSettingsDialogProps) => {
    const { palette } = useTheme();

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
        <Dialog
            id="advanced-search-dialog"
            open={isOpen}
            onClose={onCancel}
            scroll="body"
            aria-labelledby={titleAria}
            sx={{
                zIndex: 1234,
                '& .MuiDialogContent-root': {
                    minWidth: 'min(400px, 100%)',
                },
                '& .MuiPaper-root': {
                    margin: { xs: 0, sm: 2, md: 4 },
                    maxWidth: { xs: '100%', sm: '500px' },
                    display: { xs: 'block', sm: 'inline-block' },
                    background: palette.background.default,
                    color: palette.background.textPrimary,
                },
                // Remove ::after element that is added to the dialog
                '& .MuiDialog-container::after': {
                    content: 'none',
                },
            }}
        >
            {/* Title */}
            <DialogTitle
                ariaLabel={titleAria}
                title={'Cookie Settings'}
                onClose={onCancel}
            />
            <form onSubmit={formik.handleSubmit} style={{ padding: '16px' }}>
                {/* Description of cookies and why we use them */}
                <Typography variant="body1" mb={4}>
                    We use cookies to improve your experience on our website. By using our website, you agree to our use of cookies.
                </Typography>
                {/* Strictly necessary */}
                <Stack direction="column" spacing={1} sx={{ marginBottom: 2 }}>
                    <Stack direction="row" marginRight="auto" alignItems="center">
                        <Typography component="h2" variant="h5" textAlign="center">Strictly Necessary</Typography>
                        <HelpButton markdown={strictlyNecessaryText} />
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
                        <Typography component="h2" variant="h5" textAlign="center">Performance</Typography>
                        <HelpButton markdown={performanceText} />
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
                        <Typography component="h2" variant="h5" textAlign="center">Functional</Typography>
                        <HelpButton markdown={functionalText} />
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
                        <Typography component="h2" variant="h5" textAlign="center">Targeting</Typography>
                        <HelpButton markdown={targetingText} />
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
                        >Confirm</Button>
                    </Grid>
                    <Grid item xs={4}>
                        <Button
                            fullWidth
                            onClick={handleAcceptAllCookies}
                        >Accept all</Button>
                    </Grid>
                    <Grid item xs={4}>
                        <Button
                            fullWidth
                            variant="text"
                            onClick={onCancel}
                        >Cancel</Button>
                    </Grid>
                </Grid>
            </form>
        </Dialog>
    )
}