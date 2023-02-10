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
    CreatePage,
    NotificationsPage,
    SettingsPage,
} = lazily(() => import('./pages/main'));
const { TutorialPage, WelcomePage } = lazily(() => import('./pages/tutorial'));
const { StartPage } = lazily(() => import('./pages/StartPage/StartPage'));
const { StatsPage } = lazily(() => import('./pages/StatsPage/StatsPage'));
const { SearchPage } = lazily(() => import('./pages/SearchPage/SearchPage'));
const { HistorySearchPage } = lazily(() => import('./pages/HistorySearchPage/HistorySearchPage'));
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

export const Routes = (props: CommonProps) => {

    const title = useCallback((page: string) => `${page} | ${BUSINESS_NAME}`, []);

    return (
        <>
            <ScrollToTop />
            <Switch>
                {/* ========= #region Main Routes ========= */}
                {/* Pages for each of the bottom navigation items */}
                <Route
                    path={LINKS.Home}
                    sitemapIndex
                    priority={1.0}
                    changeFreq="weekly"
                >
                    <Suspense fallback={Fallback}>
                        <Page title={title('Home')} {...props}>
                            <HomePage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                <Route
                    path={`${LINKS.Search}/:params*`}
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                >
                    <Suspense fallback={Fallback}>
                        <Page {...props}>
                            <SearchPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                <Route
                    path={LINKS.Create}
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                >
                    <Suspense fallback={Fallback}>
                        <Page {...props}>
                            <CreatePage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                <Route
                    path={LINKS.Notifications}
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                >
                    <Suspense fallback={Fallback}>
                        <Page {...props}>
                            <NotificationsPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                {/* ========= #endregion Dashboard Routes ========= */}
                <Route
                    path={LINKS.History}
                    sitemapIndex={false}
                >
                    <Suspense fallback={Fallback}>
                        <Page title={title('History')} mustBeLoggedIn={true} {...props}>
                            <HistoryPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                <Route
                    path={`${LINKS.HistorySearch}/:params*`}
                    sitemapIndex={false}
                >
                    <Suspense fallback={Fallback}>
                        <Page {...props}>
                            <HistorySearchPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                {/* ========= #region Views Routes ========= */}
                {/* Views for main Vrooli components (e.g. organizations, projects, routines, standards, users) */}
                <Route
                    path={`${LINKS.Api}/:params*`}
                    sitemapIndex={false} // TODO: Add to sitemap once we can create URLS for each organization
                >
                    <Suspense fallback={Fallback}>
                        <Page {...props}>
                            <ObjectPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                <Route
                    path={`${LINKS.Note}/:params*`}
                    sitemapIndex={false} // TODO: Add to sitemap once we can create URLS for each organization
                >
                    <Suspense fallback={Fallback}>
                        <Page {...props}>
                            <ObjectPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                <Route
                    path={`${LINKS.Organization}/:params*`}
                    sitemapIndex={false} // TODO: Add to sitemap once we can create URLS for each organization
                >
                    <Suspense fallback={Fallback}>
                        <Page {...props}>
                            <ObjectPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                <Route
                    path={`${LINKS.Project}/:params*`}
                    sitemapIndex={false} // TODO: Add to sitemap once we can create URLS for each project
                >
                    <Suspense fallback={Fallback}>
                        <Page {...props}>
                            <ObjectPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                <Route
                    path={`${LINKS.Question}/:params*`}
                    sitemapIndex={false} // TODO: Add to sitemap once we can create URLS for each project
                >
                    <Suspense fallback={Fallback}>
                        <Page {...props}>
                            <ObjectPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                <Route
                    path={`${LINKS.Reminder}/:params*`}
                    sitemapIndex={false} // TODO: Add to sitemap once we can create URLS for each organization
                >
                    <Suspense fallback={Fallback}>
                        <Page {...props}>
                            <ObjectPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                <Route
                    path={`${LINKS.Routine}/:params*`}
                    sitemapIndex={false} // TODO: Add to sitemap once we can create URLS for each routine
                >
                    <Suspense fallback={Fallback}>
                        <Page {...props}>
                            <ObjectPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                <Route
                    path={`${LINKS.SmartContract}/:params*`}
                    sitemapIndex={false} // TODO: Add to sitemap once we can create URLS for each organization
                >
                    <Suspense fallback={Fallback}>
                        <Page {...props}>
                            <ObjectPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                <Route
                    path={`${LINKS.Standard}/:params*`}
                    sitemapIndex={false} // TODO: Add to sitemap once we can create URLS for each standard
                >
                    <Suspense fallback={Fallback}>
                        <Page {...props}>
                            <ObjectPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                {/* Profile editing is done on settings page, so no need for extra route */}
                <Route
                    path={`${LINKS.Profile}/:id?`}
                    sitemapIndex={false} // TODO: Add to sitemap once we can create URLS for each user
                >
                    <Suspense fallback={Fallback}>
                        <Page {...props}>
                            <UserViewPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                {/* =========  #endregion ========= */}

                {/* ========= #region Authentication Routes ========= */}
                <Route
                    path={LINKS.Start}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                >
                    <Suspense fallback={Fallback}>
                        <Page title={title('Start')} {...props}>
                            <StartPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                <Route
                    path={`${LINKS.ForgotPassword}/:code?`}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                >
                    <Suspense fallback={Fallback}>
                        <Page title={title('Forgot Password')} {...props}>
                            <FormPage title="Forgot Password" maxWidth="700px">
                                <ForgotPasswordForm />
                            </FormPage>
                        </Page>
                    </Suspense>
                </Route>
                <Route
                    path={`${LINKS.ResetPassword}/:userId?/:code?`}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                >
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
                <Route
                    path={LINKS.Settings}
                    sitemapIndex={false}
                >
                    <Suspense fallback={Fallback}>
                        <Page title={title('Settings')} {...props} mustBeLoggedIn={true} >
                            <SettingsPage session={props.session} />
                        </Page>
                    </Suspense>
                </Route>
                <Route
                    path={LINKS.Tutorial}
                    sitemapIndex
                    priority={0.5}
                    changeFreq="monthly"
                >
                    <Suspense fallback={Fallback}>
                        <Page title={title('Tutorial')} {...props}>
                            <TutorialPage />
                        </Page>
                    </Suspense>
                </Route>
                <Route
                    path={LINKS.Welcome}
                    sitemapIndex
                    priority={0.5}
                    changeFreq="monthly"
                >
                    <Suspense fallback={Fallback}>
                        <Page title={title('Welcome')} {...props}>
                            <WelcomePage {...props} />
                        </Page>
                    </Suspense>
                </Route>
                <Route
                    path={LINKS.Stats}
                    sitemapIndex
                    priority={0.5}
                    changeFreq="weekly"
                >
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