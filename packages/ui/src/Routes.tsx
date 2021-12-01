import React, { Suspense, useCallback, useMemo } from 'react';
import { lazily } from 'react-lazily';
import { Switch, Route } from 'react-router-dom';
import { ROLES } from '@local/shared';
import { LINKS } from 'utils';
import { Sitemap } from 'Sitemap';
import {
    ForgotPasswordForm,
    LogInForm,
    ResetPasswordForm,
    SignUpForm,
    WaitlistForm
} from 'forms';
import { ScrollToTop } from 'components';
import { CommonProps } from 'types';

// Lazy loading in the Routes component is a recommended way to improve performance. See https://reactjs.org/docs/code-splitting.html#route-based-code-splitting
const {
    AboutPage,
    ActorViewPage,
    DevelopPage,
    FormPage,
    HomePage,
    LandingPage,
    LearnPage,
    MissionPage,
    NotFoundPage,
    Page,
    PrivacyPolicyPage,
    OrganizationViewPage,
    ProjectsPage,
    ProjectViewPage,
    ResearchPage,
    RoutineOrchestratorPage,
    RoutineViewPage,
    RunRoutinePage,
    TermsPage,
} = lazily(() => import('./pages'));

const Routes = (props: CommonProps) => {

    const title = useCallback((page: string) => `${page} | ${props.business?.BUSINESS_NAME?.Short}`, [props.business?.BUSINESS_NAME?.Short]);

    // Informational pages to describe Vrooli to potential customers
    const informational = useMemo(() => (
        <React.Fragment>
            <Route
                exact
                path={LINKS.Landing}
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
        </React.Fragment>
    ), [props, title])

    // Customizable pages available to logged in users
    const dashboards = useMemo(() => (
        <React.Fragment>
            <Route
                exact
                path={LINKS.Home}
                sitemapIndex={false}
                render={() => (
                    <Page title={title('Home')} {...props} restrictedToRoles={Object.values(ROLES)}>
                        <HomePage />
                    </Page>
                )}
            />
            <Route
                exact
                path={LINKS.Projects}
                sitemapIndex={false}
                render={() => (
                    <Page title={title('Projects')} {...props} restrictedToRoles={Object.values(ROLES)}>
                        <ProjectsPage />
                    </Page>
                )}
            />
            <Route
                exact
                path={LINKS.Learn}
                sitemapIndex={false}
                render={() => (
                    <Page title={title('Learn')} {...props} restrictedToRoles={Object.values(ROLES)}>
                        <LearnPage />
                    </Page>
                )}
            />
            <Route
                exact
                path={LINKS.Research}
                sitemapIndex={false}
                render={() => (
                    <Page title={title('Research')} {...props} restrictedToRoles={Object.values(ROLES)}>
                        <ResearchPage />
                    </Page>
                )}
            />
            <Route
                exact
                path={LINKS.Develop}
                sitemapIndex={false}
                render={() => (
                    <Page title={title('Develop')} {...props} restrictedToRoles={Object.values(ROLES)}>
                        <DevelopPage />
                    </Page>
                )}
            />
        </React.Fragment>
    ), [props, title])

    // Pages for creating and running routine orchestrations
    const orchestrations = useMemo(() => (
        <React.Fragment>
            <Route
                exact
                path={`${LINKS.Orchestration}/:id?`}
                sitemapIndex={false}
                render={() => (
                    <Page title={title('Plan Routine')} {...props} restrictedToRoles={Object.values(ROLES)}>
                        <RoutineOrchestratorPage />
                    </Page>
                )}
            />
            <Route
                exact
                path={`${LINKS.Run}/:id?`}
                sitemapIndex={false}
                render={() => (
                    <Page title={title('Run Routine')} {...props} restrictedToRoles={Object.values(ROLES)}>
                        <RunRoutinePage />
                    </Page>
                )}
            />
        </React.Fragment>
    ), [props, title])

    // Views for main Vrooli components (organizations, actors, projects, routines, resources, data)
    const views = useMemo(() => (
        <React.Fragment>
            <Route
                exact
                path={`${LINKS.Profile}/:id?`}
                sitemapIndex={true}
                priority={0.1}
                render={() => (
                    <Page title={title('Profile')} {...props}>
                        <ActorViewPage />
                    </Page>
                )}
            />
            <Route
                exact
                path={`${LINKS.Organization}/:id?`}
                sitemapIndex={true}
                priority={0.1}
                render={() => (
                    <Page title={title('Organization')} {...props}>
                        <OrganizationViewPage />
                    </Page>
                )}
            />
            <Route
                exact
                path={`${LINKS.Project}/:id?`}
                sitemapIndex={true}
                priority={0.1}
                render={() => (
                    <Page title={title('Project')} {...props}>
                        <ProjectViewPage />
                    </Page>
                )}
            />
            <Route
                exact
                path={`${LINKS.Routine}/:id?`}
                sitemapIndex={true}
                priority={0.1}
                render={() => (
                    <Page title={title('Routine')} {...props}>
                        <RoutineViewPage />
                    </Page>
                )}
            />
        </React.Fragment>
    ), [props, title])

    const authentication = useMemo(() => (
        <React.Fragment>
            <Route
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
            />
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
        </React.Fragment>
    ), [props, title])

    return (
        <Suspense fallback={<div>Loading...</div>}>
            <ScrollToTop />
            <Switch>
                <Route
                    path="/sitemap"
                    component={Sitemap}
                />
                {informational}
                {dashboards}
                {orchestrations}
                {views}
                {authentication}
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