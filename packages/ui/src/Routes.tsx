import { Suspense, useCallback } from 'react';
import { lazily } from 'react-lazily';
import { Route, Routes } from 'react-router-dom';
import { BUSINESS_NAME, ROLES } from '@local/shared';
import { APP_LINKS as LINKS } from '@local/shared';
// import { Sitemap } from 'Sitemap';
import {
    ForgotPasswordForm,
    ResetPasswordForm
} from 'forms';
import { ScrollToTop } from 'components';
import { CommonProps } from 'types';

// Lazy loading in the Routes component is a recommended way to improve performance. See https://reactjs.org/docs/code-splitting.html#route-based-code-splitting
const {
    ActorViewPage,
    DevelopPage,
    FormPage,
    HomePage,
    LearnPage,
    NotFoundPage,
    Page,
    OrganizationViewPage,
    ProjectsPage,
    ProjectViewPage,
    ResearchPage,
    RoutineOrchestratorPage,
    RoutineViewPage,
    RunRoutinePage,
    StartPage,
    StatsPage,
} = lazily(() => import('./pages'));

export const AllRoutes = (props: CommonProps) => {

    const title = useCallback((page: string) => `${page} | ${BUSINESS_NAME}`, []);

    return (
        <Suspense fallback={<div>Loading...</div>}>
            <ScrollToTop />
            <Routes>
                {/* <Route
                    path="/sitemap"
                    element={Sitemap}
                /> */}
                {/* ========= #region Dashboard Routes ========= */}
                {/* Customizable pages available to logged in users */}
                <Route
                    path={LINKS.Home}
                    // sitemapIndex={false}
                    element={
                        <Page title={title('Home')} {...props}>
                            <HomePage />
                        </Page>
                    }
                />
                <Route
                    path={`${LINKS.Projects}/*`}
                    // sitemapIndex={false}
                    element={
                        <Page title={title('Projects')} {...props}>
                            <ProjectsPage session={props.session} />
                        </Page>
                    }
                />
                <Route
                    path={`${LINKS.Learn}/*`}
                    // sitemapIndex={false}
                    element={
                        <Page title={title('Learn')} {...props}>
                            <LearnPage />
                        </Page>
                    }
                />
                <Route
                    path={`${LINKS.Research}/*`}
                    // sitemapIndex={false}
                    element={
                        <Page title={title('Research')} {...props}>
                            <ResearchPage />
                        </Page>
                    }
                />
                <Route
                    path={`${LINKS.Develop}/*`}
                    // sitemapIndex={false}
                    element={
                        <Page title={title('Develop')} {...props}>
                            <DevelopPage />
                        </Page>
                    }
                />
                {/* ========= #endregion Dashboard Routes ========= */}

                {/* ========= #region Orchestration Routes ========= */}
                {/* Pages for creating and running routine orchestrations */}
                <Route
                    path={`${LINKS.Orchestrate}/:id`}
                    // sitemapIndex={false}
                    element={
                        <Page title={title('Plan Routine')} {...props} restrictedToRoles={Object.values(ROLES)}>
                            <RoutineOrchestratorPage />
                        </Page>
                    }
                />
                <Route
                    path={`${LINKS.Run}/:id?`}
                    // sitemapIndex={false}
                    element={
                        <Page title={title('Run Routine')} {...props}>
                            <RunRoutinePage />
                        </Page>
                    }
                />
                {/* ========= #endregion Orchestration Routes ========= */}

                {/* ========= #region Views Routes ========= */}
                {/* Views for main Vrooli components (organizations, actors, projects, routines, resources, data) */}
                <Route
                    path={`${LINKS.Profile}/:id?`}
                    // sitemapIndex={true}
                    // priority={0.1}
                    element={
                        <Page title={title('Profile')} {...props}>
                            <ActorViewPage />
                        </Page>
                    }
                />
                <Route
                    path={`${LINKS.Organization}/:id?`}
                    // sitemapIndex={true}
                    // priority={0.1}
                    element={
                        <Page title={title('Organization')} {...props}>
                            <OrganizationViewPage />
                        </Page>
                    }
                />
                <Route
                    path={`${LINKS.Project}/:id?`}
                    // sitemapIndex={true}
                    // priority={0.1}
                    element={
                        <Page title={title('Project')} {...props}>
                            <ProjectViewPage />
                        </Page>
                    }
                />
                <Route
                    path={`${LINKS.Routine}/:id?`}
                    // sitemapIndex={true}
                    // priority={0.1}
                    element={
                        <Page title={title('Routine')} {...props}>
                            <RoutineViewPage />
                        </Page>
                    }
                />

                {/* =========  #endregion ========= */}

                {/* ========= #region Authentication Routes ========= */}
                <Route
                    path={LINKS.Start}
                    // sitemapIndex={true}
                    // priority={0.8}
                    element={
                        <Page title={title('Start')} {...props}>
                            <StartPage {...props} />
                        </Page>
                    }
                />
                <Route
                    path={`${LINKS.ForgotPassword}/:code?`}
                    // sitemapIndex={true}
                    // priority={0.1}
                    element={
                        <Page title={title('Forgot Password')} {...props}>
                            <FormPage title="Forgot Password" maxWidth="700px">
                                <ForgotPasswordForm />
                            </FormPage>
                        </Page>
                    }
                />
                <Route
                    path={`${LINKS.ResetPassword}/:id?/:code?`}
                    // sitemapIndex={true}
                    // priority={0.1}
                    element={
                        <Page title={title('Reset Password')} {...props}>
                            <FormPage title="Reset Password" maxWidth="700px">
                                <ResetPasswordForm {...props} />
                            </FormPage>
                        </Page>
                    }
                />

                {/* =========  #endregion ========= */}

                <Route
                    path={LINKS.Stats}
                    // sitemapIndex={false}
                    element={
                        <Page title={title('Stats📊')} {...props}>
                            <StatsPage />
                        </Page>
                    }
                />

                <Route
                    element={
                        <Page title={title('404')} {...props}>
                            <NotFoundPage />
                        </Page>
                    }
                />
            </Routes>
        </Suspense>
    );
}