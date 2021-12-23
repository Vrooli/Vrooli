import { emailRequestPasswordChangeMutation } from 'graphql/mutation';
import { useMutation } from '@apollo/client';
import { emailRequestPasswordChangeSchema } from '@local/shared';
import { useFormik } from 'formik';
import {
    Button,
    Grid,
    Link,
    TextField,
    Typography
} from '@mui/material';
import { makeStyles } from '@mui/styles';
import { APP_LINKS } from '@local/shared';
import { FORMS } from 'utils';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { useNavigate } from 'react-router-dom';
import { formStyles } from './styles';
import { emailRequestPasswordChange } from 'graphql/generated/emailRequestPasswordChange';
import { FormProps } from './types';

const useStyles = makeStyles(formStyles);

export const ForgotPasswordForm = ({
    onFormChange = () => {}
}: FormProps) => {
    const classes = useStyles();
    const navigate = useNavigate();
    const [emailRequestPasswordChange, {loading}] = useMutation<emailRequestPasswordChange>(emailRequestPasswordChangeMutation);

    const formik = useFormik({
        initialValues: {
            email: ''
        },
        validationSchema: emailRequestPasswordChangeSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation: emailRequestPasswordChange,
                input: { values },
                successCondition: (response) => response.data.emailRequestPasswordChange,
                onSuccess: () => navigate(APP_LINKS.Home),
                successMessage: () => 'Request sent. Please check email.',
            })
        },
    });

    const toSignUp = () => onFormChange(FORMS.SignUp);
    const toLogIn = () => onFormChange(FORMS.LogIn);

    return (
        <form className={classes.form} onSubmit={formik.handleSubmit}>
            <Grid container spacing={2}>
                <Grid item xs={12}>
                    <TextField
                        fullWidth
                        autoFocus
                        id="email"
                        name="email"
                        autoComplete="email"
                        label="Email Address"
                        value={formik.values.email}
                        onChange={formik.handleChange}
                        error={formik.touched.email && Boolean(formik.errors.email)}
                        helperText={formik.touched.email && formik.errors.email}
                    />
                </Grid>
            </Grid>
            <Button
                fullWidth
                disabled={loading}
                type="submit"
                color="secondary"
                className={classes.submit}
            >
                Submit
            </Button>
            <Grid container spacing={2}>
                <Grid item xs={6}>
                    <Link onClick={toLogIn}>
                        <Typography className={classes.clickSize}>
                            Remember? Back to Log In
                        </Typography>
                    </Link>
                </Grid>
                <Grid item xs={6}>
                    <Link onClick={toSignUp}>
                        <Typography className={`${classes.clickSize} ${classes.linkRight}`}>
                            Don't have an account? Sign up
                        </Typography>
                    </Link>
                </Grid>
            </Grid>
        </form>
    );
}