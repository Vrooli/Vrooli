import { Box } from '@mui/material';
import { LINKS } from '@shared/consts';
import { Route, RouteProps, Switch } from '@shared/route';
import { FullPageSpinner } from 'components/FullPageSpinner/FullPageSpinner';
import { NavbarProps } from 'components/navigation/types';
import { ScrollToTop } from 'components/ScrollToTop';
import { ForgotPasswordForm } from 'forms/ForgotPasswordForm';
import { ResetPasswordForm } from 'forms/ResetPasswordForm';
import { lazily } from 'react-lazily';
import { CommonProps } from 'types';
import { PageProps } from 'views/wrapper/types';
import { Page } from './components/Page/Page';

// Lazy loading in the Routes component is a recommended way to improve performance. See https://reactjs.org/docs/code-splitting.html#route-based-code-splitting
const {
    HomeView,
    HistoryView,
    CreateView,
    NotificationsView,
} = lazily(() => import('./views/main'));
const {
    SettingsView,
    SettingsAuthenticationView,
    SettingsDisplayView,
    SettingsNotificationsView,
    SettingsProfileView,
    SettingsPrivacyView,
    SettingsSchedulesView,
} = lazily(() => import('./views/settings'));
const {
    PrivacyPolicyView,
    TermsView
} = lazily(() => import('./views/legal'));
const { TutorialView } = lazily(() => import('./views/tutorial'));
const { WelcomeView } = lazily(() => import('./views/WelcomeView/WelcomeView'));
const { AboutView } = lazily(() => import('./views/AboutView/AboutView'));
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

/**
 * Fallback displayed while route is being loaded.
 */
const Fallback = <Box>
    {/* A blank Navbar to display before the actual one (which is dynamic depending on the page) is rendered. */}
    <Box sx={{
        background: (t) => t.palette.primary.dark,
        height: '64px!important',
        zIndex: 100,
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
    }} />
    {/* Loading spinner */}
    <FullPageSpinner />
</Box>

/**
 * Wrapper to define Navbar within each route. This allows us to 
 * customize the Navbar for each route, without it flashing on
 * the screen before the route is loaded.
 */
const NavRoute = (props: PageProps & RouteProps & NavbarProps) => {
    return (
        <Route {...props}>
            <Page {...props}>
                {props.children}
            </Page>
        </Route>
    )
}

/**
 * Style for pages that don't use left/right padding
 */
const noSidePadding = {
    paddingLeft: 0,
    paddingRight: 0,
}

export const Routes = (props: CommonProps & { sessionChecked: boolean }) => {
    return (
        <>
            <ScrollToTop />
            <Switch fallback={Fallback}>
                <NavRoute
                    path={LINKS.About}
                    sitemapIndex
                    priority={0.5}
                    changeFreq="monthly"
                    {...props}
                >
                    <AboutView {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.Api}/add`} mustBeLoggedIn={true} {...props}>
                    <ApiCreate {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.Api}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <ApiUpdate {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.Api}/:id`} {...props}>
                    <ApiView {...props} />
                </NavRoute>
                <NavRoute path={LINKS.Awards} {...props}>
                    <AwardsView {...props} />
                </NavRoute>
                <NavRoute path={LINKS.Calendar} {...props}>
                    <CalendarView {...props} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Create}
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    mustBeLoggedIn={true}
                    {...props}
                >
                    <CreateView {...props} />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.ForgotPassword}/:code?`}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...props}
                >
                    <FormView title="Forgot Password" maxWidth="700px" {...props}>
                        <ForgotPasswordForm />
                    </FormView>
                </NavRoute>
                <NavRoute path={LINKS.History} mustBeLoggedIn={true} {...props}>
                    <HistoryView {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.HistorySearch}/:params*`} mustBeLoggedIn={true} {...props}>
                    <HistorySearchView {...props} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Home}
                    sitemapIndex
                    priority={1.0}
                    changeFreq="weekly"
                    {...props}
                >
                    <HomeView {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.Note}/add`} mustBeLoggedIn={true} {...props}>
                    <NoteCreate {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.Note}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <NoteUpdate {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.Note}/:id`} {...props}>
                    <NoteView {...props} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Notifications}
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    mustBeLoggedIn={true}
                    {...props}
                >
                    <NotificationsView {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.Organization}/add`} sx={noSidePadding} mustBeLoggedIn={true} {...props}>
                    <OrganizationCreate {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.Organization}/edit/:id`} sx={noSidePadding} mustBeLoggedIn={true} {...props}>
                    <OrganizationUpdate {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.Organization}/:id`} sx={noSidePadding} {...props}>
                    <OrganizationView {...props} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Premium}
                    sitemapIndex
                    priority={0.5}
                    changeFreq="weekly"
                    {...props}
                >
                    <PremiumView {...props} />
                </NavRoute>
                <NavRoute
                    path={LINKS.PrivacyPolicy}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...props}
                >
                    <PrivacyPolicyView {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.Profile}/:id?`} sx={noSidePadding} {...props}>
                    <UserView {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.Project}/add`} mustBeLoggedIn={true} {...props}>
                    <ProjectCreate {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.Project}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <ProjectUpdate {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.Project}/:id`} {...props}>
                    <ProjectView {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.Question}/add`} mustBeLoggedIn={true} {...props}>
                    <QuestionCreate {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.Question}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <QuestionUpdate {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.Question}/:id`} {...props}>
                    <QuestionView {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.Reminder}/add`} mustBeLoggedIn={true} {...props}>
                    <ReminderCreate {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.Reminder}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <ReminderUpdate {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.Reminder}/:id`} {...props}>
                    <ReminderView {...props} />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.ResetPassword}/:params*`}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...props}
                >
                    <FormView title="Reset Password" maxWidth="700px" {...props}>
                        <ResetPasswordForm />
                    </FormView>
                </NavRoute>
                <NavRoute path={`${LINKS.Routine}/add`} mustBeLoggedIn={true} {...props}>
                    <RoutineCreate {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.Routine}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <RoutineUpdate {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.Routine}/:id`} {...props}>
                    <RoutineView {...props} />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.Search}/:params*`}
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    {...props}
                >
                    <SearchView {...props} />
                </NavRoute>
                <NavRoute path={LINKS.Settings} mustBeLoggedIn={true} {...props}>
                    <SettingsView {...props} />
                </NavRoute>
                <NavRoute path={LINKS.SettingsAuthentication} mustBeLoggedIn={true} {...props}>
                    <SettingsAuthenticationView {...props} />
                </NavRoute>
                <NavRoute path={LINKS.SettingsDisplay} mustBeLoggedIn={true} {...props}>
                    <SettingsDisplayView {...props} />
                </NavRoute>
                <NavRoute path={LINKS.SettingsNotifications} mustBeLoggedIn={true} {...props}>
                    <SettingsNotificationsView {...props} />
                </NavRoute>
                <NavRoute path={LINKS.SettingsProfile} mustBeLoggedIn={true} {...props}>
                    <SettingsProfileView {...props} />
                </NavRoute>
                <NavRoute path={LINKS.SettingsPrivacy} mustBeLoggedIn={true} {...props}>
                    <SettingsPrivacyView {...props} />
                </NavRoute>
                <NavRoute path={LINKS.SettingsSchedules} mustBeLoggedIn={true} {...props}>
                    <SettingsSchedulesView {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.SmartContract}/add`} mustBeLoggedIn={true} {...props}>
                    <SmartContractCreate {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.SmartContract}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <SmartContractUpdate {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.SmartContract}/:id`} {...props}>
                    <SmartContractView {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.Standard}/add`} mustBeLoggedIn={true} {...props}>
                    <StandardCreate {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.Standard}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <StandardUpdate {...props} />
                </NavRoute>
                <NavRoute path={`${LINKS.Standard}/:id`} {...props}>
                    <StandardView {...props} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Start}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...props}
                >
                    <StartView {...props} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Stats}
                    sitemapIndex
                    priority={0.5}
                    changeFreq="weekly"
                    {...props}
                >
                    <StatsView {...props} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Terms}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...props}
                >
                    <TermsView {...props} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Tutorial}
                    sitemapIndex
                    priority={0.5}
                    changeFreq="monthly"
                    {...props}
                >
                    <TutorialView {...props} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Welcome}
                    sitemapIndex
                    priority={0.5}
                    changeFreq="monthly"
                    {...props}
                >
                    <WelcomeView {...props} />
                </NavRoute>
                <NavRoute {...props}>
                    <NotFoundView {...props} />
                </NavRoute>
            </Switch>
        </>
    );
}