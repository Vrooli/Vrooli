import { useCallback } from 'react';
import { lazily } from 'react-lazily';
import { Route, Switch } from '@shared/route';
import { APP_LINKS as LINKS, BUSINESS_NAME } from '@shared/consts';
import {
    ForgotPasswordForm,
    ResetPasswordForm
} from 'forms';
import { ScrollToTop } from 'components';
import { CommonProps } from 'types';
import { Page } from './components/Page/Page';
import { Box, CircularProgress } from '@mui/material';

// Lazy loading in the Routes component is a recommended way to improve performance. See https://reactjs.org/docs/code-splitting.html#route-based-code-splitting
const {
    HomeView,
    HistoryView,
    CreateView,
    NotificationsView,
    SettingsView,
} = lazily(() => import('./views/main'));
const { TutorialView, WelcomeView } = lazily(() => import('./views/tutorial'));
const { AwardsView } = lazily(() => import('./views/AwardsView/AwardsView'));
const { CalendarView } = lazily(() => import('./views/CalendarView/CalendarView'));
const { FormView } = lazily(() => import('./views/wrapper/FormView'));
const { HistorySearchView } = lazily(() => import('./views/HistorySearchView/HistorySearchView'));
const { NotFoundView } = lazily(() => import('./views/NotFoundView/NotFoundView'));
const { PremiumView } = lazily(() => import('./views/PremiumView/PremiumView'));
const { SearchView } = lazily(() => import('./views/SearchView/SearchView'));
const { StartView } = lazily(() => import('./views/StartView/StartView'));
const { StatsView } = lazily(() => import('./views/StatsView/StatsView'));
const { ApiCreate, ApiUpdate, ApiView } = lazily(() => import('./views/objects/api'));
const { NoteCreate, NoteUpdate, NoteView } = lazily(() => import('./views/objects/note'));
const { OrganizationCreate, OrganizationUpdate, OrganizationView } = lazily(() => import('./views/objects/organization'));
const { ProjectCreate, ProjectUpdate, ProjectView } = lazily(() => import('./views/objects/project'));
const { QuestionCreate, QuestionUpdate, QuestionView } = lazily(() => import('./views/objects/question'));
const { ReminderCreate, ReminderUpdate, ReminderView } = lazily(() => import('./views/objects/reminder'));
const { RoutineCreate, RoutineUpdate, RoutineView } = lazily(() => import('./views/objects/routine'));
const { SmartContractCreate, SmartContractUpdate, SmartContractView } = lazily(() => import('./views/objects/smartContract'));
const { StandardCreate, StandardUpdate, StandardView } = lazily(() => import('./views/objects/standard'));
const { UserView } = lazily(() => import('./views/objects/user'));

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
    // Tab title for static (non-dynamic) pages (e.g. Home, Search, Create, Notifications).
    const title = useCallback((page: string) => `${page} | ${BUSINESS_NAME}`, []);

    return (
        <>
            <ScrollToTop />
            <Switch fallback={Fallback}>
                <Route path={`${LINKS.Api}/add`} sitemapIndex={false}>
                    <Page {...props}>
                        <ApiCreate session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Api}/edit/:id`} sitemapIndex={false}>
                    <Page {...props}>
                        <ApiUpdate session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Api}/:id`} sitemapIndex={false}>
                    <Page {...props}>
                        <ApiView session={props.session} />
                    </Page>
                </Route>
                <Route
                    path={LINKS.Awards}
                    sitemapIndex
                    priority={0.5}
                    changeFreq="weekly"
                >
                    <Page title={title('Awards🏆')} {...props}>
                        <AwardsView {...props} />
                    </Page>
                </Route>
                <Route path={LINKS.Calendar} sitemapIndex={false}>
                    <Page title={title('Calendar')} {...props}>
                        <CalendarView session={props.session} />
                    </Page>
                </Route>
                <Route
                    path={LINKS.Create}
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                >
                    <Page {...props}>
                        <CreateView session={props.session} />
                    </Page>
                </Route>
                <Route
                    path={`${LINKS.ForgotPassword}/:code?`}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                >
                    <Page title={title('Forgot Password')} {...props}>
                        <FormView title="Forgot Password" maxWidth="700px">
                            <ForgotPasswordForm />
                        </FormView>
                    </Page>
                </Route>
                <Route path={LINKS.History} sitemapIndex={false}>
                    <Page title={title('History')} mustBeLoggedIn={true} {...props}>
                        <HistoryView session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.HistorySearch}/:params*`} sitemapIndex={false}>
                    <Page {...props}>
                        <HistorySearchView session={props.session} />
                    </Page>
                </Route>
                <Route
                    path={LINKS.Home}
                    sitemapIndex
                    priority={1.0}
                    changeFreq="weekly"
                >
                    <Page title={title('Home')} {...props}>
                        <HomeView session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Note}/add`} sitemapIndex={false}>
                    <Page {...props}>
                        <NoteCreate session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Note}/edit/:id`} sitemapIndex={false}>
                    <Page {...props}>
                        <NoteUpdate session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Note}/:id`} sitemapIndex={false}>
                    <Page {...props}>
                        <NoteView session={props.session} />
                    </Page>
                </Route>
                <Route
                    path={LINKS.Notifications}
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                >
                    <Page {...props}>
                        <NotificationsView session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Organization}/add`} sitemapIndex={false}>
                    <Page {...props}>
                        <OrganizationCreate session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Organization}/edit/:id`} sitemapIndex={false}>
                    <Page {...props}>
                        <OrganizationUpdate session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Organization}/:id`} sitemapIndex={false}>
                    <Page {...props}>
                        <OrganizationView session={props.session} />
                    </Page>
                </Route>
                <Route
                    path={LINKS.Premium}
                    sitemapIndex
                    priority={0.5}
                    changeFreq="weekly"
                >
                    <Page title={title('Premium😎')} {...props}>
                        <PremiumView {...props} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Profile}/:id?`} sitemapIndex={false}>
                    <Page {...props}>
                        <UserView session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Project}/add`} sitemapIndex={false}>
                    <Page {...props}>
                        <ProjectCreate session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Project}/edit/:id`} sitemapIndex={false}>
                    <Page {...props}>
                        <ProjectUpdate session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Project}/:id`} sitemapIndex={false}>
                    <Page {...props}>
                        <ProjectView session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Question}/add`} sitemapIndex={false}>
                    <Page {...props}>
                        <QuestionCreate session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Question}/edit/:id`} sitemapIndex={false}>
                    <Page {...props}>
                        <QuestionUpdate session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Question}/:id`} sitemapIndex={false}>
                    <Page {...props}>
                        <QuestionView session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Reminder}/add`} sitemapIndex={false}>
                    <Page {...props}>
                        <ReminderCreate session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Reminder}/edit/:id`} sitemapIndex={false}>
                    <Page {...props}>
                        <ReminderUpdate session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Reminder}/:id`} sitemapIndex={false}>
                    <Page {...props}>
                        <ReminderView session={props.session} />
                    </Page>
                </Route>
                <Route
                    path={`${LINKS.ResetPassword}/:userId?/:code?`}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                >
                    {(params: any) => (
                        <Page title={title('Reset Password')} {...props}>
                            <FormView title="Reset Password" maxWidth="700px">
                                <ResetPasswordForm userId={params.userId} code={params.code} />
                            </FormView>
                        </Page>
                    )}
                </Route>
                <Route path={`${LINKS.Routine}/add`} sitemapIndex={false}>
                    <Page {...props}>
                        <RoutineCreate session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Routine}/edit/:id`} sitemapIndex={false}>
                    <Page {...props}>
                        <RoutineUpdate session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Routine}/:id`} sitemapIndex={false}>
                    <Page {...props}>
                        <RoutineView session={props.session} />
                    </Page>
                </Route>
                <Route
                    path={`${LINKS.Search}/:params*`}
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                >
                    <Page {...props}>
                        <SearchView session={props.session} />
                    </Page>
                </Route>
                <Route path={LINKS.Settings} sitemapIndex={false}>
                    <Page title={title('Settings⚙️')} {...props} mustBeLoggedIn={true} >
                        <SettingsView session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.SmartContract}/add`} sitemapIndex={false}>
                    <Page {...props}>
                        <SmartContractCreate session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.SmartContract}/edit/:id`} sitemapIndex={false}>
                    <Page {...props}>
                        <SmartContractUpdate session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.SmartContract}/:id`} sitemapIndex={false}>
                    <Page {...props}>
                        <SmartContractView session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Standard}/add`} sitemapIndex={false}>
                    <Page {...props}>
                        <StandardCreate session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Standard}/edit/:id`} sitemapIndex={false}>
                    <Page {...props}>
                        <StandardUpdate session={props.session} />
                    </Page>
                </Route>
                <Route path={`${LINKS.Standard}/:id`} sitemapIndex={false}>
                    <Page {...props}>
                        <StandardView session={props.session} />
                    </Page>
                </Route>
                <Route
                    path={LINKS.Start}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                >
                    <Page title={title('Start')} {...props}>
                        <StartView session={props.session} />
                    </Page>
                </Route>
                <Route
                    path={LINKS.Stats}
                    sitemapIndex
                    priority={0.5}
                    changeFreq="weekly"
                >
                    <Page title={title('Stats📊')} {...props}>
                        <StatsView {...props} />
                    </Page>
                </Route>
                <Route
                    path={LINKS.Tutorial}
                    sitemapIndex
                    priority={0.5}
                    changeFreq="monthly"
                >
                    <Page title={title('Tutorial🤔')} {...props}>
                        <TutorialView />
                    </Page>
                </Route>
                <Route
                    path={LINKS.Welcome}
                    sitemapIndex
                    priority={0.5}
                    changeFreq="monthly"
                >
                    <Page title={title('Welcome!💙')} {...props} sx={{
                        display: 'flex',
                        justifyContent: 'center',
                        alignItems: 'center',
                        textAlign: 'center',
                    }}>
                        <WelcomeView {...props} />
                    </Page>
                </Route>
                <Route>
                    <Page title={title('404🥺')} {...props}>
                        <NotFoundView />
                    </Page>
                </Route>
            </Switch>
        </>
    );
}