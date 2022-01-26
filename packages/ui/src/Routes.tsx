import { Suspense, useCallback } from 'react';
import { lazily } from 'react-lazily';
import { Route, Switch } from 'wouter';
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
    ProjectViewPage,
    ResearchPage,
    RoutineOrchestratorPage,
    RoutineViewPage,
    RunRoutinePage,
    SearchActorsPage,
    SearchOrganizationsPage,
    SearchProjectsPage,
    SearchRoutinesPage,
    SearchStandardsPage,
    StandardViewPage,
    StartPage,
    StatsPage,
} = lazily(() => import('./pages'));

export const AllRoutes = (props: CommonProps) => {

    const title = useCallback((page: string) => `${page} | ${BUSINESS_NAME}`, []);

    return (
        <Suspense fallback={<div>Loading...</div>}>
            <ScrollToTop />
            {/* <Route
                    path="/sitemap"
                    element={Sitemap}
                /> */}
            <Switch>
                {/* ========= #region Dashboard Routes ========= */}
                {/* Customizable pages available to logged in users */}
                <Route path={LINKS.Home}>
                    <Page title={title('Home')} {...props}>
                        <HomePage session={props.session ?? {}} />
                    </Page>
                </Route>
                <Route path={LINKS.Learn} >
                    <Page title={title('Learn')} {...props}>
                        <LearnPage />
                    </Page>
                </Route>
                <Route path={LINKS.Research}>
                    <Page title={title('Research')} {...props}>
                        <ResearchPage />
                    </Page>
                </Route>
                <Route path={LINKS.Develop}>
                    <Page title={title('Develop')} {...props}>
                        <DevelopPage />
                    </Page>
                </Route>
                {/* ========= #endregion Dashboard Routes ========= */}

                {/* ========= #region Search Routes ========= */}
                <Route path={`${LINKS.SearchUsers}/:id?`}>
                    <Page title={title('Users Search')} {...props}>
                        <SearchActorsPage session={props.session ?? {}} />
                    </Page>
                </Route>
                <Route path={`${LINKS.SearchOrganizations}/:id?`}>
                    <Page title={title('Organizations Search')} {...props}>
                        <SearchOrganizationsPage session={props.session ?? {}} />
                    </Page>
                </Route>
                <Route path={`${LINKS.SearchProjects}/:id?`}>
                    <Page title={title('Projects Search')} {...props}>
                        <SearchProjectsPage session={props.session ?? {}} />
                    </Page>
                </Route>
                <Route path={`${LINKS.SearchRoutines}/:id?`}>
                    <Page title={title('Routines Search')} {...props}>
                        <SearchRoutinesPage session={props.session ?? {}} />
                    </Page>
                </Route>
                <Route path={`${LINKS.SearchStandards}/:id?`}>
                    <Page title={title('Standards Search')} {...props}>
                        <SearchStandardsPage session={props.session ?? {}} />
                    </Page>
                </Route>
                {/* ========= #endregion Search Routes ========= */}

                {/* ========= #region Orchestration Routes ========= */}
                {/* Pages for creating and running routine orchestrations */}
                <Route path={`${LINKS.Orchestrate}/:id`}>
                    <Page title={title('Plan Routine')} {...props} restrictedToRoles={Object.values(ROLES)}>
                        <RoutineOrchestratorPage />
                    </Page>
                </Route>
                <Route path={`${LINKS.Run}/:id?`}>
                    <Page title={title('Run Routine')} {...props}>
                        <RunRoutinePage />
                    </Page>
                </Route>
                {/* ========= #endregion Orchestration Routes ========= */}

                {/* ========= #region Views Routes ========= */}
                {/* Views for main Vrooli components (organizations, actors, projects, routines, resources, data) */}
                {/* Opens objects as their own page, as opposed to the search routes which open them as popup dialogs */}
                <Route path={`${LINKS.Profile}/:id?`}>
                    <Page title={title('Profile')} {...props}>
                        <ActorViewPage session={props.session ?? {}}/>
                    </Page>
                </Route>
                <Route path={`${LINKS.Organization}/:id?`}>
                    <Page title={title('Organization')} {...props}>
                        <OrganizationViewPage session={props.session ?? {}} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Project}/:id?`}>
                    <Page title={title('Project')} {...props}>
                        <ProjectViewPage session={props.session ?? {}} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Routine}/:id?`}>
                    <Page title={title('Routine')} {...props}>
                        <RoutineViewPage session={props.session ?? {}} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Standard}/:id?`}>
                    <Page title={title('Standard')} {...props}>
                        <StandardViewPage session={props.session ?? {}} />
                    </Page>
                </Route>
                {/* =========  #endregion ========= */}

                {/* ========= #region Authentication Routes ========= */}
                <Route path={LINKS.Start}>
                    <Page title={title('Start')} {...props}>
                        <StartPage {...props} />
                    </Page>
                </Route>
                <Route path={`${LINKS.ForgotPassword}/:code?`} >
                    <Page title={title('Forgot Password')} {...props}>
                        <FormPage title="Forgot Password" maxWidth="700px">
                            <ForgotPasswordForm />
                        </FormPage>
                    </Page>
                </Route>
                <Route path={`${LINKS.ResetPassword}/:userId?/:code?`}>
                    {(params: any) => <Page title={title('Reset Password')} {...props}>
                        <FormPage title="Reset Password" maxWidth="700px">
                            <ResetPasswordForm userId={params.userId} code={params.code} onSessionUpdate={props.onSessionUpdate} />
                        </FormPage>
                    </Page>}
                </Route>
                {/* =========  #endregion ========= */}

                <Route path={LINKS.Stats}>
                    <Page title={title('Stats📊')} {...props}>
                        <StatsPage />
                    </Page>
                </Route>
                <Route>
                    <Page title={title('404')} {...props}>
                        <NotFoundPage />
                    </Page>
                </Route>
            </Switch>
        </Suspense>
    );
}