import { Suspense } from 'react';
import { lazily } from 'react-lazily';
import { Switch, Route } from 'react-router-dom';
import { ROLES } from '@local/shared';
import { LINKS } from 'utils';
import { Sitemap } from 'Sitemap';
import {
    ForgotPasswordForm,
    LogInForm,
    ProfileForm,
    ResetPasswordForm,
    WaitlistForm
} from 'forms';
import { ScrollToTop } from 'components';
import { CommonProps } from 'types';

// Lazy loading in the Routes component is a recommended way to improve performance. See https://reactjs.org/docs/code-splitting.html#route-based-code-splitting
const {
    AboutPage,
    FormPage,
    LandingPage,
    MissionPage,
    NotFoundPage,
    Page,
    PrivacyPolicyPage,
    TermsPage,
} = lazily(() => import('./pages'));

const Routes = (props: CommonProps) => {

    const title = (page: string) => `${page} | ${props.business?.BUSINESS_NAME?.Short}`;

    return (
        <Suspense fallback={<div>Loading...</div>}>
            <ScrollToTop />
            <Switch>
                {/* START PUBLIC PAGES */}
                <Route
                    path="/sitemap"
                    component={Sitemap}
                />
                <Route
                    exact
                    path={LINKS.Home}
                    sitemapIndex={true}
                    priority={1.0}
                    changefreq="monthly"
                    render={() => (
                        <Page title={title('Home')} {...props}>
                            <LandingPage />
                        </Page>
                    )}
                />
                <Route
                    exact
                    path={LINKS.Mission}
                    sitemapIndex={true}
                    priority={1.0}
                    changefreq="monthly"
                    render={() => (
                        <Page title={title('Mission')} {...props}>
                            <MissionPage />
                        </Page>
                    )}
                />
                <Route
                    exact
                    path={LINKS.About}
                    sitemapIndex={true}
                    priority={0.7}
                    render={() => (
                        <Page title={title('About')} {...props}>
                            <AboutPage {...props} />
                        </Page>
                    )}
                />
                <Route
                    exact
                    path={LINKS.PrivacyPolicy}
                    sitemapIndex={true}
                    priority={0.1}
                    render={() => (
                        <Page title={title('Privacy Policy')} {...props}>
                            <PrivacyPolicyPage business={props.business} />
                        </Page>
                    )}
                />
                <Route
                    exact
                    path={LINKS.Terms}
                    sitemapIndex={true}
                    priority={0.1}
                    render={() => (
                        <Page title={title('Terms & Conditions')} {...props}>
                            <TermsPage business={props.business} />
                        </Page>
                    )}
                />
                {/* <Route
                    exact
                    path={LINKS.Register}
                    sitemapIndex={true}
                    priority={0.9}
                    render={() => (
                        <Page title={title('Sign Up')} {...props}>
                            <FormPage title="Sign Up" maxWidth="700px">
                                <SignUpForm {...props} />
                            </FormPage>
                        </Page>
                    )}
                /> */}
                <Route
                    exact
                    path={`${LINKS.Waitlist}/:code?`}
                    sitemapIndex={true}
                    priority={0.8}
                    render={() => (
                        <Page title={title('Log In')} {...props}>
                            <FormPage title="Join Waitlist" maxWidth="700px">
                                <WaitlistForm {...props} />
                            </FormPage>
                        </Page>
                    )}
                />
                <Route
                    exact
                    path={`${LINKS.LogIn}/:code?`}
                    sitemapIndex={true}
                    priority={0.8}
                    render={() => (
                        <Page title={title('Log In')} {...props}>
                            <FormPage title="Log In" maxWidth="700px">
                                <LogInForm {...props} />
                            </FormPage>
                        </Page>
                    )}
                />
                <Route
                    exact
                    path={`${LINKS.ForgotPassword}/:code?`}
                    sitemapIndex={true}
                    priority={0.1}
                    render={() => (
                        <Page title={title('Forgot Password')} {...props}>
                            <FormPage title="Forgot Password" maxWidth="700px">
                                <ForgotPasswordForm />
                            </FormPage>
                        </Page>
                    )}
                />
                <Route
                    exact
                    path={`${LINKS.ResetPassword}/:id?/:code?`}
                    sitemapIndex={true}
                    priority={0.1}
                    render={() => (
                        <Page title={title('Reset Password')} {...props}>
                            <FormPage title="Reset Password" maxWidth="700px">
                                <ResetPasswordForm {...props} />
                            </FormPage>
                        </Page>
                    )}
                />
                {/* END PUBLIC PAGES */}
                {/* START CUSTOMER PAGES */}
                <Route
                    exact
                    path={LINKS.Profile}
                    sitemapIndex={true}
                    priority={0.4}
                    render={() => (
                        <Page title={title('Profile')} {...props} restrictedToRoles={Object.values(ROLES)}>
                            <FormPage title="Profile">
                                <ProfileForm />
                            </FormPage>
                        </Page>
                    )}
                />
                {/* END CUSTOMER PAGES */}
                {/* 404 page */}
                <Route
                    render={() => (
                        <Page title={title('404')} {...props}>
                            <NotFoundPage />
                        </Page>
                    )}
                />
            </Switch>
        </Suspense>
    );
}

export { Routes };