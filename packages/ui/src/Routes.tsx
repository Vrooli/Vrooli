import { Suspense, useCallback } from 'react';
import { lazily } from 'react-lazily';
import { Route, Switch } from '@shared/route';
import { APP_LINKS as LINKS, BUSINESS_NAME } from '@shared/consts';
// import { Sitemap } from 'Sitemap';
import {
    ForgotPasswordForm,
    ResetPasswordForm
} from 'forms';
import { ScrollToTop } from 'components';
import { CommonProps } from 'types';
import { Page } from 'pages/wrapper/Page';
import { Box, CircularProgress } from '@mui/material';

// Lazy loading in the Routes component is a recommended way to improve performance. See https://reactjs.org/docs/code-splitting.html#route-based-code-splitting
const {
    HomePage,
    HistoryPage,
    LearnPage,
    ResearchPage,
    DevelopPage,
} = lazily(() => import('./pages/dashboard'));
const { WelcomePage } = lazily(() => import('./pages/WelcomePage/WelcomePage'));
const { SettingsPage } = lazily(() => import('./pages/SettingsPage/SettingsPage'));
const { StartPage } = lazily(() => import('./pages/StartPage/StartPage'));
const { StatsPage } = lazily(() => import('./pages/dashboard/StatsPage/StatsPage'));
const { SearchPage } = lazily(() => import('./pages/SearchPage/SearchPage'));
const { HistorySearchPage } = lazily(() => import('./pages/HistorySearchPage/HistorySearchPage'));
const { DevelopSearchPage } = lazily(() => import('./pages/DevelopSearchPage/DevelopSearchPage'));
const { ObjectPage } = lazily(() => import('./pages/ObjectPage/ObjectPage'));
const { UserViewPage } = lazily(() => import('./pages/view/UserViewPage'));
const { FormPage } = lazily(() => import('./pages/wrapper/FormPage'));
const { NotFoundPage } = lazily(() => import('./pages/NotFoundPage'));

const Fallback = <Box sx={{
    position: 'absolute',
    top: '50%',
    left: '50%',
    transform: 'translate(-50%, -50%)',
    zIndex: 100000,
}}>
    <CircularProgress size={100} />
</Box>

export const AllRoutes = (props: CommonProps) => {

    const title = useCallback((page: string) => `${page} | ${BUSINESS_NAME}`, []);

    return (
        <>
            <ScrollToTop />
            {/* <Route
                    path="/sitemap"
                    element={Sitemap}
                /> */}
            <Switch>
                {/* ========= #region Dashboard Routes ========= */}
                {/* Customizable pages available to logged in users */}
                <Route path={LINKS.Home}>
                    <Suspense fallback={Fallback}>
                        <Page title={title('Home')} {...props}>
                            <HomePage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                <Route path={LINKS.History}>
                    <Suspense fallback={Fallback}>
                        <Page title={title('History')} mustBeLoggedIn={true} {...props}>
                            <HistoryPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                <Route path={LINKS.Learn} >
                    <Suspense fallback={Fallback}>
                        <Page title={title('Learn')} {...props}>
                            <LearnPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                <Route path={LINKS.Research}>
                    <Suspense fallback={Fallback}>
                        <Page title={title('Research')} {...props}>
                            <ResearchPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                <Route path={LINKS.Develop}>
                    <Suspense fallback={Fallback}>
                        <Page title={title('Develop')} {...props}>
                            <DevelopPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                {/* ========= #endregion Dashboard Routes ========= */}

                {/* ========= #region Search Routes ========= */}
                <Route path={`${LINKS.Search}/:params*`}>
                    <Suspense fallback={Fallback}>
                        <Page {...props}>
                            <SearchPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                <Route path={`${LINKS.HistorySearch}/:params*`}>
                    <Suspense fallback={Fallback}>
                        <Page {...props}>
                            <HistorySearchPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                <Route path={`${LINKS.DevelopSearch}/:params*`}>
                    <Suspense fallback={Fallback}>
                        <Page {...props}>
                            <DevelopSearchPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                {/* ========= #endregion Search Routes ========= */}

                {/* ========= #region Views Routes ========= */}
                {/* Views for main Vrooli components (i.e. organizations, projects, routines, standards, users) */}
                <Route path={`${LINKS.Organization}/:params*`}>
                    <Suspense fallback={Fallback}>
                        <Page {...props}>
                            <ObjectPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                <Route path={`${LINKS.Project}/:params*`}>
                    <Suspense fallback={Fallback}>
                        <Page {...props}>
                            <ObjectPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                <Route path={`${LINKS.Routine}/:params*`}>
                    <Suspense fallback={Fallback}>
                        <Page {...props}>
                            <ObjectPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                <Route path={`${LINKS.Standard}/:params*`}>
                    <Suspense fallback={Fallback}>
                        <Page {...props}>
                            <ObjectPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                {/* Profile editing is done on settings page, so no need for extra route */}
                <Route path={`${LINKS.Profile}/:id?`}>
                    <Suspense fallback={Fallback}>
                        <Page {...props}>
                            <UserViewPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                {/* =========  #endregion ========= */}

                {/* ========= #region Authentication Routes ========= */}
                <Route path={LINKS.Start}>
                    <Suspense fallback={Fallback}>
                        <Page title={title('Start')} {...props}>
                            <StartPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                <Route path={`${LINKS.ForgotPassword}/:code?`} >
                    <Suspense fallback={Fallback}>
                        <Page title={title('Forgot Password')} {...props}>
                            <FormPage title="Forgot Password" maxWidth="700px">
                                <ForgotPasswordForm />
                            </FormPage>
                        </Page>
                    </Suspense>
                </Route>
                <Route path={`${LINKS.ResetPassword}/:userId?/:code?`}>
                    {(params: any) => (
                        <Suspense fallback={Fallback}>
                            <Page title={title('Reset Password')} {...props}>
                                <FormPage title="Reset Password" maxWidth="700px">
                                    <ResetPasswordForm userId={params.userId} code={params.code} />
                                </FormPage>
                            </Page>
                        </Suspense>
                    )}
                </Route>
                {/* ========= #endregion ========= */}
                <Route path={LINKS.Settings}>
                    <Suspense fallback={Fallback}>
                        <Page title={title('Settings')} {...props} mustBeLoggedIn={true} >
                            <SettingsPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                <Route path={LINKS.Welcome}>
                    <Suspense fallback={Fallback}>
                        <Page title={title('Welcome')} {...props}>
                            <WelcomePage />
                        </Page>
                    </Suspense>
                </Route>
                <Route path={LINKS.Stats}>
                    <Suspense fallback={Fallback}>
                        <Page title={title('Stats📊')} {...props}>
                            <StatsPage />
                        </Page>
                    </Suspense>
                </Route>
                <Route>
                    <Suspense fallback={Fallback}>
                        <Page title={title('404')} {...props}>
                            <NotFoundPage />
                        </Page>
                    </Suspense>
                </Route>
            </Switch>
        </>
    );
}