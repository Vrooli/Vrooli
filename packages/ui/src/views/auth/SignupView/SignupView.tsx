import { BUSINESS_NAME, emailSignUpFormValidation, EmailSignUpInput, endpointPostAuthEmailSignup, LINKS, Session } from "@local/shared";
import { Box, Button, Checkbox, FormControl, FormControlLabel, FormHelperText, Grid, InputAdornment, keyframes, Link, Typography, useTheme } from "@mui/material";
import { fetchLazyWrapper, hasErrorCode } from "api";
import AiDrivenConvo from "assets/img/AiDrivenConvo.png";
import Blob1 from "assets/img/blob1.svg";
import Blob2 from "assets/img/blob2.svg";
import CollaborativeRoutines from "assets/img/CollaborativeRoutines.png";
import OrganizationalManagement from "assets/img/OrganizationalManagement.png";
import { PasswordTextInput } from "components/inputs/PasswordTextInput/PasswordTextInput";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Testimonials } from "components/Testimonials/Testimonials";
import { Field, Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { formNavLink, formPaper, formSubmit } from "forms/styles";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useWindowSize } from "hooks/useWindowSize";
import { EmailIcon, UserIcon } from "icons";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { clickSize } from "styles";
import { Cookies } from "utils/cookies";
import { PubSub } from "utils/pubsub";
import { setupPush } from "utils/push";
import { SignupViewProps } from "views/types";

const SignupForm = () => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const [emailSignUp, { loading }] = useLazyFetch<EmailSignUpInput, Session>(endpointPostAuthEmailSignup);

    return (
        <>
            <Formik
                initialValues={{
                    agreeToTerms: false,
                    marketingEmails: true,
                    name: "",
                    email: "",
                    password: "",
                    confirmPassword: "",
                }}
                onSubmit={(values, helpers) => {
                    if (values.password !== values.confirmPassword) {
                        PubSub.get().publishSnack({ messageKey: "PasswordsDontMatch", severity: "Error" });
                        helpers.setSubmitting(false);
                        return;
                    }
                    fetchLazyWrapper<EmailSignUpInput, Session>({
                        fetch: emailSignUp,
                        inputs: {
                            name: values.name,
                            email: values.email,
                            password: values.password,
                            confirmPassword: values.confirmPassword,
                            marketingEmails: Boolean(values.marketingEmails),
                            theme: palette.mode ?? "light",
                        },
                        onSuccess: (data) => {
                            localStorage.removeItem(Cookies.FormData); // Clear old form data cache
                            setupPush();
                            PubSub.get().publishSession(data);
                            PubSub.get().publishAlertDialog({
                                messageKey: "WelcomeVerifyEmail",
                                messageVariables: { appName: BUSINESS_NAME },
                                buttons: [{
                                    labelKey: "Ok", onClick: () => {
                                        setLocation(LINKS.Home);
                                        PubSub.get().publishTutorial();
                                    },
                                }],
                            });
                        },
                        onError: (response) => {
                            if (hasErrorCode(response, "EmailInUse")) {
                                PubSub.get().publishAlertDialog({
                                    messageKey: "EmailInUseWrongPassword",
                                    buttons: [
                                        { labelKey: "Yes", onClick: () => { setLocation(LINKS.ForgotPassword); } },
                                        { labelKey: "No" },
                                    ],
                                });
                            }
                            helpers.setSubmitting(false);
                        },
                    });
                }}
                validationSchema={emailSignUpFormValidation}
            >
                {(formik) => <BaseForm
                    display={"dialog"}
                    isLoading={loading}
                    style={{
                        ...formPaper,
                        paddingBottom: "unset",
                    }}
                >
                    <Grid container spacing={2}>
                        <Grid item xs={12}>
                            <Field
                                fullWidth
                                autoComplete="name"
                                name="name"
                                label={t("Name")}
                                placeholder={t("NamePlaceholder")}
                                as={TextInput}
                                InputProps={{
                                    startAdornment: (
                                        <InputAdornment position="start">
                                            <UserIcon />
                                        </InputAdornment>
                                    ),
                                }}
                            />
                        </Grid>
                        <Grid item xs={12}>
                            <Field
                                fullWidth
                                autoComplete="email"
                                name="email"
                                label={t("Email", { count: 1 })}
                                placeholder={t("EmailPlaceholder")}
                                as={TextInput}
                                InputProps={{
                                    startAdornment: (
                                        <InputAdornment position="start">
                                            <EmailIcon />
                                        </InputAdornment>
                                    ),
                                }}
                                helperText={formik.touched.email && formik.errors.email}
                                error={formik.touched.email && Boolean(formik.errors.email)}
                            />
                        </Grid>
                        <Grid item xs={12}>
                            <PasswordTextInput
                                fullWidth
                                name="password"
                                autoComplete="new-password"
                                label={t("Password")}
                            />
                        </Grid>
                        <Grid item xs={12}>
                            <PasswordTextInput
                                fullWidth
                                name="confirmPassword"
                                autoComplete="new-password"
                                label={t("PasswordConfirm")}
                            />
                        </Grid>
                        <Grid item xs={12} sx={{ display: "flex", justifyContent: "left" }}>
                            <FormControlLabel
                                control={
                                    <Checkbox
                                        id="marketingEmails"
                                        name="marketingEmails"
                                        color="secondary"
                                        checked={Boolean(formik.values.marketingEmails)}
                                        onBlur={formik.handleBlur}
                                        onChange={formik.handleChange}
                                    />
                                }
                                label="I agree to receive marketing promotions and updates via email."
                            />
                        </Grid>
                        <Grid item xs={12} sx={{ display: "flex", justifyContent: "left" }}>
                            <FormControl required error={!formik.values.agreeToTerms && formik.touched.agreeToTerms}>
                                <FormControlLabel
                                    control={
                                        <Checkbox
                                            id="agreeToTerms"
                                            name="agreeToTerms"
                                            color="secondary"
                                            checked={Boolean(formik.values.agreeToTerms)}
                                            onBlur={formik.handleBlur}
                                            onChange={formik.handleChange}
                                        />
                                    }
                                    label={
                                        <>
                                            I agree to the{" "}
                                            <Link
                                                href="/terms-and-conditions"
                                                target="_blank"
                                                rel="noopener noreferrer"
                                                onClick={(event) => event.stopPropagation()}
                                            >
                                                terms and conditions
                                            </Link>
                                            .
                                        </>
                                    }
                                />
                                <FormHelperText>
                                    {formik.touched.agreeToTerms && !formik.values.agreeToTerms && "You must agree to the terms and conditions"}
                                </FormHelperText>
                            </FormControl>
                        </Grid>
                    </Grid>
                    <Button
                        fullWidth
                        disabled={loading}
                        type="submit"
                        color="secondary"
                        variant="contained"
                        sx={{ ...formSubmit }}
                    >
                        {t("SignUp")}
                    </Button>
                    <Box sx={{
                        display: "flex",
                        flexDirection: "row",
                        justifyContent: "space-between",
                        alignItems: "center",
                    }}>
                        <Link href={LINKS.Login}>
                            <Typography
                                sx={{
                                    ...clickSize,
                                    ...formNavLink,
                                }}
                            >
                                {t("LogIn")}
                            </Typography>
                        </Link>
                        <Link href={LINKS.ForgotPassword}>
                            <Typography
                                sx={{
                                    ...clickSize,
                                    ...formNavLink,
                                    flexDirection: "row-reverse",
                                }}
                            >
                                {t("ForgotPassword")}
                            </Typography>
                        </Link>
                    </Box>
                </BaseForm>}
            </Formik>
        </>
    );
};

const blueRadial = "radial-gradient(circle, rgb(6 46 46) 12%, rgb(1 36 36) 52%, rgb(3 20 20) 80%)";

// Animation for blob1
// Moves up and grows, then moves down to the right and shrinks.
// Then it moves to the left - while continuing to shrink- until it reaches the starting position.
const blob1Animation = keyframes`
    0% {
        transform: translateY(0) scale(0.7);
        filter: hue-rotate(0deg) blur(100px);
    }
    33% {
        transform: translateY(-110px) scale(1) rotate(-150deg);
        filter: hue-rotate(40deg) blur(100px);
    }
    66% {
        transform: translate(30px, -50px) scale(0.8) rotate(-250deg);
        filter: hue-rotate(80deg) blur(100px);
    }
    100% {
        transform: translate(0px, 0px) scale(0.7) rotate(0deg);
        filter: hue-rotate(0deg) blur(100px);
    }
`;

// Animation for blob2
// Moves to the right and changes hue, then moves back to the left and turns its original color.
const blob2Animation = keyframes`
    0% {
        transform: translateX(0) scale(0.8);
        filter: hue-rotate(0deg) blur(50px);
    }
    50% {
        transform: translateX(120px) scale(0.9);
        filter: hue-rotate(-70deg) blur(50px);
    }
    100% {
        transform: translateX(0) scale(0.8);
        filter: hue-rotate(0deg) blur(50px);
    }
`;

const ImageWithCaption = ({ src, alt, caption }) => (
    <Box sx={{
        flex: "0 0 auto",
        margin: "0 10px",
        minWidth: { xs: "250px", lg: "300px" },
        maxWidth: { xs: "250px", lg: "300px" },
    }}>
        <img src={src} alt={alt} style={{ width: "100%", borderRadius: "8px" }} />
        <Typography variant="caption" align="center" sx={{ whiteSpace: "nowrap" }}>{caption}</Typography>
    </Box>
);

const Promo = () => {
    return (
        <>
            {/* Blob 1 */}
            <Box sx={{
                position: "fixed",
                pointerEvents: "none",
                top: -200,
                left: -50,
                width: "100%",
                height: "100%",
                opacity: 0.5,
                filter: "hue-rotate(150deg)",
                transition: "opacity 1s ease-in-out",
                zIndex: 0,
            }}>
                <Box
                    component="img"
                    src={Blob1}
                    alt="Blob 1"
                    sx={{
                        width: "100%",
                        height: "100%",
                        animation: `${blob1Animation} 20s linear infinite`,
                    }}
                />
            </Box>
            {/* Blob 2 */}
            <Box sx={{
                position: "fixed",
                pointerEvents: "none",
                bottom: -175,
                right: -250,
                width: "100%",
                height: "100%",
                opacity: 0.5,
                filter: "hue-rotate(300deg)",
                transition: "opacity 1s ease-in-out",
                zIndex: 0,
            }}>
                <Box
                    component="img"
                    src={Blob2}
                    alt="Blob 2"
                    sx={{
                        width: "150%",
                        height: "150%",
                        animation: `${blob2Animation} 20s linear infinite`,
                    }}
                />
            </Box>
            <Box sx={{ position: "relative", zIndex: 1 }}>
                <Typography variant="h4" sx={{ marginBottom: 2 }}>
                    Where Imagination Drives Automation
                </Typography>
                <Typography variant="body1" sx={{ marginBottom: 2 }}>
                    Welcome to Vrooli: Where autonomous agents turn your dreams into action. Discover the Vrooli difference:
                </Typography>
                <ul style={{ marginBottom: "32px" }}>
                    <Typography component="li" variant="body1" sx={{ marginBottom: 1 }}>
                        Meet our bots: Like having a helpful friend, they chat with you and help you get things done.
                    </Typography>
                    <Typography component="li" variant="body1" sx={{ marginBottom: 1 }}>
                        Introducing Routines: Think of them as step-by-step guides that can be used, shared, or tweaked to suit your needs. Whether it's writing a blog or sending an email, there's a routine for that.
                    </Typography>
                    <Typography component="li" variant="body1" sx={{ marginBottom: 1 }}>
                        Share and Grow: Got a cool routine? Share it! Need one? Use one shared by others and even combine them for new solutions.
                    </Typography>
                    <Typography component="li" variant="body1" sx={{ marginBottom: 1 }}>
                        Personality Matters: Our bots have their own styles! For instance, get a story written by one bot and it might sound poetic, while another might give it a thrilling twist.
                    </Typography>
                    <Typography component="li" variant="body1" sx={{ marginBottom: 1 }}>
                        Teamwork made easy: Chat with humans and bots all in one space. Everyone's in the loop and tasks get done faster!
                    </Typography>
                    <Typography component="li" variant="body1" sx={{ marginBottom: 1 }}>
                        Set and Forget: Have recurring tasks? Just schedule them and let the bots handle the rest.
                    </Typography>
                </ul>
                <Box sx={{
                    margin: "20px 0",
                    display: "flex",
                    gap: { xs: 0, md: 4 },
                    overflowX: "auto",
                    alignItems: "center",
                }}>
                    <ImageWithCaption
                        src={AiDrivenConvo}
                        alt="A conversation between a user and a bot. The user asks the bot about starting a business, and the bot gives suggestions on how to get started."
                        caption="Chat seamlessly with bots"
                    />
                    <ImageWithCaption
                        src={CollaborativeRoutines}
                        alt="A graphical representation of the nodes and edges of a routine."
                        caption="Design routines tailored for you"
                    />
                    <ImageWithCaption
                        src={OrganizationalManagement}
                        alt="The page for an organization, showing the organization's name, bio, picture, and members."
                        caption="Build and showcase your automated organization"
                    />
                </Box>
                <Testimonials />
            </Box>
        </>
    );
};

export const SignupView = ({
    display,
    onClose,
}: SignupViewProps) => {
    const { breakpoints, palette } = useTheme();
    const { t } = useTranslation();
    const isXs = useWindowSize(({ width }) => width <= breakpoints.values.sm);

    return (
        <Box sx={{ maxHeight: "100vh", overflow: "hidden" }}>
            <TopBar
                display={display}
                onClose={onClose}
                hideTitleOnDesktop
                title={t("SignUp")}
            />
            <Box sx={{
                display: "flex",
                flexDirection: "row",
                height: "100vh",
            }}>
                <Box sx={{
                    width: { xs: "100%", sm: "min(400px, 100%)" },
                    height: "100vh",
                    background: palette.background.paper,
                    borderRadius: 0,
                    overflow: "auto",
                    paddingBottom: "100px",
                    zIndex: 3,
                }}>
                    <SignupForm />
                </Box>
                {!isXs && <Box sx={{
                    flex: 1,
                    background: blueRadial,
                    backgroundAttachment: "fixed",
                    color: "white",
                    padding: 2,
                    paddingBottom: "64px",
                    overflowY: "auto",
                }}>
                    <Promo />
                </Box>}
            </Box>
        </Box>
    );
};


// // Wallet provider popups
    // const [connectOpen, setConnectOpen] = useState(false);
    // const [installOpen, setInstallOpen] = useState(false);
    // const openWalletConnectDialog = useCallback(() => { setConnectOpen(true); }, []);
    // const openWalletInstallDialog = useCallback(() => { setInstallOpen(true); }, []);

    // // Performs handshake to establish trust between site backend and user's wallet.
    // // 1. Whitelist website on wallet
    // // 2. Send public address to backend
    // // 3. Store public address and nonce in database
    // // 4. Sign human-readable message (which includes nonce) using wallet
    // // 5. Send signed message to backend for verification
    // // 6. Receive JWT and user session
    // const walletLogin = useCallback(async (providerKey: string) => {
    //     // Check if wallet extension installed
    //     if (!hasWalletExtension(providerKey)) {
    //         PubSub.get().publishAlertDialog({
    //             messageKey: "WalletProviderNotFoundDetails",
    //             buttons: [
    //                 { labelKey: "TryAgain", onClick: openWalletConnectDialog },
    //                 { labelKey: "InstallWallet", onClick: openWalletInstallDialog },
    //                 { labelKey: "EmailLogin", onClick: toEmailLogIn },
    //             ],
    //         });
    //         return;
    //     }
    //     // Validate wallet
    //     const walletCompleteResult = await validateWallet(providerKey);
    //     if (walletCompleteResult?.session) {
    //         PubSub.get().publishSnack({ messageKey: "WalletVerified", severity: "Success" });
    //         PubSub.get().publishSession(walletCompleteResult.session);
    //         // Redirect to main dashboard
    //         setLocation(redirect ?? LINKS.Home);
    //         // Set up push notifications
    //         setupPush();
    //     }
    // }, [openWalletConnectDialog, openWalletInstallDialog, toEmailLogIn, setLocation, redirect]);

    // const closeWalletConnectDialog = useCallback((providerKey: string | null) => {
    //     setConnectOpen(false);
    //     if (providerKey) {
    //         walletLogin(providerKey);
    //     }
    // }, [walletLogin]);

    // const closeWalletInstallDialog = useCallback(() => { setInstallOpen(false); }, []);

    //  <WalletSelectDialog
    //             handleOpenInstall={openWalletInstallDialog}
    //             open={connectOpen}
    //             onClose={closeWalletConnectDialog}
    //         />
    //         <WalletInstallDialog
    //             open={installOpen}
    //             onClose={closeWalletInstallDialog}
    //         />

    //          <Box sx={{
    //                     display: "flex",
    //                     justifyContent: "center",
    //                     alignItems: "center",
    //                     marginBottom: 2,
    //                 }}>
    //                     <Typography
    //                         variant="h6"
    //                         sx={{
    //                             display: "inline-block",
    //                         }}
    //                     >
    //                         {t("SelectLogInMethod")}
    //                     </Typography>
    //                     <HelpButton markdown={helpText} />
    //                 </Box>
    //                 <Stack
    //                     direction="column"
    //                     spacing={2}
    //                 >
    //                     <Button
    //                         fullWidth
    //                         onClick={openWalletConnectDialog}
    //                         startIcon={<WalletIcon />}
    //                         sx={{ ...buttonProps }}
    //                     >{t("Wallet")}</Button>
    //                     <Button
    //                         fullWidth
    //                         onClick={toEmailLogIn}
    //                         startIcon={<EmailIcon />}
    //                         sx={{ ...buttonProps }}
    //                     >{t("Email")}</Button>
    //                 </Stack>
