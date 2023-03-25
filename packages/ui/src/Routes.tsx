import { Box } from '@mui/material';
import { LINKS } from '@shared/consts';
import { Route, RouteProps, Switch } from '@shared/route';
import { FullPageSpinner } from 'components/FullPageSpinner/FullPageSpinner';
import { NavbarProps } from 'components/navigation/types';
import { ScrollToTop } from 'components/ScrollToTop';
import { ForgotPasswordForm, ResetPasswordForm } from 'forms/auth';
import { lazily } from 'react-lazily';
import { PageProps } from 'views/wrapper/types';
import { Page } from './components/Page/Page';

// Lazy loading in the Routes component is a recommended way to improve performance. See https://reactjs.org/docs/code-splitting.html#route-based-code-splitting
const {
    HomeView,
    HistoryView,
    CreateView,
    MyStuffView,
    NotificationsView,
} = lazily(() => import('./views/main'));
const {
    SettingsView,
    SettingsAuthenticationView,
    SettingsDisplayView,
    SettingsNotificationsView,
    SettingsProfileView,
    SettingsPrivacyView,
    SettingsFocusModesView: SettingsSchedulesView,
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

export const Routes = (props: { sessionChecked: boolean }) => {
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
                    <AboutView />
                </NavRoute>
                <NavRoute path={`${LINKS.Api}/add`} mustBeLoggedIn={true} {...props}>
                    <ApiCreate />
                </NavRoute>
                <NavRoute path={`${LINKS.Api}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <ApiUpdate />
                </NavRoute>
                <NavRoute path={`${LINKS.Api}/:id`} {...props}>
                    <ApiView />
                </NavRoute>
                <NavRoute path={LINKS.Awards} {...props}>
                    <AwardsView />
                </NavRoute>
                <NavRoute path={LINKS.Calendar} {...props}>
                    <CalendarView />
                </NavRoute>
                <NavRoute
                    path={LINKS.Create}
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    mustBeLoggedIn={true}
                    {...props}
                >
                    <CreateView />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.ForgotPassword}/:code?`}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...props}
                >
                    <FormView title="Forgot Password" maxWidth="700px" {...props}>
                        <ForgotPasswordForm onClose={() => { }} />
                    </FormView>
                </NavRoute>
                <NavRoute path={LINKS.History} mustBeLoggedIn={true} {...props}>
                    <HistoryView />
                </NavRoute>
                <NavRoute
                    path={LINKS.Home}
                    sitemapIndex
                    priority={1.0}
                    changeFreq="weekly"
                    {...props}
                >
                    <HomeView />
                </NavRoute>
                <NavRoute path={LINKS.MyStuff} mustBeLoggedIn={true} {...props}>
                    <MyStuffView />
                </NavRoute>
                <NavRoute path={`${LINKS.Note}/add`} mustBeLoggedIn={true} {...props}>
                    <NoteCreate />
                </NavRoute>
                <NavRoute path={`${LINKS.Note}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <NoteUpdate />
                </NavRoute>
                <NavRoute path={`${LINKS.Note}/:id`} {...props}>
                    <NoteView />
                </NavRoute>
                <NavRoute
                    path={LINKS.Notifications}
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    mustBeLoggedIn={true}
                    {...props}
                >
                    <NotificationsView />
                </NavRoute>
                <NavRoute path={`${LINKS.Organization}/add`} sx={noSidePadding} mustBeLoggedIn={true} {...props}>
                    <OrganizationCreate />
                </NavRoute>
                <NavRoute path={`${LINKS.Organization}/edit/:id`} sx={noSidePadding} mustBeLoggedIn={true} {...props}>
                    <OrganizationUpdate />
                </NavRoute>
                <NavRoute path={`${LINKS.Organization}/:id`} sx={noSidePadding} {...props}>
                    <OrganizationView />
                </NavRoute>
                <NavRoute
                    path={LINKS.Premium}
                    sitemapIndex
                    priority={0.5}
                    changeFreq="weekly"
                    {...props}
                >
                    <PremiumView />
                </NavRoute>
                <NavRoute
                    path={LINKS.Privacy}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...props}
                >
                    <PrivacyPolicyView />
                </NavRoute>
                <NavRoute path={`${LINKS.Profile}/:id?`} sx={noSidePadding} {...props}>
                    <UserView />
                </NavRoute>
                <NavRoute path={`${LINKS.Project}/add`} mustBeLoggedIn={true} {...props}>
                    <ProjectCreate />
                </NavRoute>
                <NavRoute path={`${LINKS.Project}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <ProjectUpdate />
                </NavRoute>
                <NavRoute path={`${LINKS.Project}/:id`} {...props}>
                    <ProjectView />
                </NavRoute>
                <NavRoute path={`${LINKS.Question}/add`} mustBeLoggedIn={true} {...props}>
                    <QuestionCreate />
                </NavRoute>
                <NavRoute path={`${LINKS.Question}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <QuestionUpdate />
                </NavRoute>
                <NavRoute path={`${LINKS.Question}/:id`} {...props}>
                    <QuestionView />
                </NavRoute>
                <NavRoute path={`${LINKS.Reminder}/add`} mustBeLoggedIn={true} {...props}>
                    <ReminderCreate />
                </NavRoute>
                <NavRoute path={`${LINKS.Reminder}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <ReminderUpdate />
                </NavRoute>
                <NavRoute path={`${LINKS.Reminder}/:id`} {...props}>
                    <ReminderView />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.ResetPassword}/:params*`}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...props}
                >
                    <FormView title="Reset Password" maxWidth="700px" {...props}>
                        <ResetPasswordForm onClose={() => { }} />
                    </FormView>
                </NavRoute>
                <NavRoute path={`${LINKS.Routine}/add`} mustBeLoggedIn={true} {...props}>
                    <RoutineCreate />
                </NavRoute>
                <NavRoute path={`${LINKS.Routine}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <RoutineUpdate />
                </NavRoute>
                <NavRoute path={`${LINKS.Routine}/:id`} {...props}>
                    <RoutineView />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.Search}/:params*`}
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    {...props}
                >
                    <SearchView />
                </NavRoute>
                <NavRoute path={LINKS.Settings} mustBeLoggedIn={true} {...props}>
                    <SettingsView />
                </NavRoute>
                <NavRoute path={LINKS.SettingsAuthentication} mustBeLoggedIn={true} {...props}>
                    <SettingsAuthenticationView />
                </NavRoute>
                <NavRoute path={LINKS.SettingsDisplay} mustBeLoggedIn={true} {...props}>
                    <SettingsDisplayView />
                </NavRoute>
                <NavRoute path={LINKS.SettingsNotifications} mustBeLoggedIn={true} {...props}>
                    <SettingsNotificationsView />
                </NavRoute>
                <NavRoute path={LINKS.SettingsProfile} mustBeLoggedIn={true} {...props}>
                    <SettingsProfileView />
                </NavRoute>
                <NavRoute path={LINKS.SettingsPrivacy} mustBeLoggedIn={true} {...props}>
                    <SettingsPrivacyView />
                </NavRoute>
                <NavRoute path={LINKS.SettingsSchedules} mustBeLoggedIn={true} {...props}>
                    <SettingsSchedulesView />
                </NavRoute>
                <NavRoute path={`${LINKS.SmartContract}/add`} mustBeLoggedIn={true} {...props}>
                    <SmartContractCreate />
                </NavRoute>
                <NavRoute path={`${LINKS.SmartContract}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <SmartContractUpdate />
                </NavRoute>
                <NavRoute path={`${LINKS.SmartContract}/:id`} {...props}>
                    <SmartContractView />
                </NavRoute>
                <NavRoute path={`${LINKS.Standard}/add`} mustBeLoggedIn={true} {...props}>
                    <StandardCreate />
                </NavRoute>
                <NavRoute path={`${LINKS.Standard}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <StandardUpdate />
                </NavRoute>
                <NavRoute path={`${LINKS.Standard}/:id`} {...props}>
                    <StandardView />
                </NavRoute>
                <NavRoute
                    path={LINKS.Start}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...props}
                >
                    <StartView />
                </NavRoute>
                <NavRoute
                    path={LINKS.Stats}
                    sitemapIndex
                    priority={0.5}
                    changeFreq="weekly"
                    {...props}
                >
                    <StatsView />
                </NavRoute>
                <NavRoute
                    path={LINKS.Terms}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...props}
                >
                    <TermsView />
                </NavRoute>
                <NavRoute
                    path={LINKS.Tutorial}
                    sitemapIndex
                    priority={0.5}
                    changeFreq="monthly"
                    {...props}
                >
                    <TutorialView />
                </NavRoute>
                <NavRoute
                    path={LINKS.Welcome}
                    sitemapIndex
                    priority={0.5}
                    changeFreq="monthly"
                    {...props}
                >
                    <WelcomeView />
                </NavRoute>
                <NavRoute {...props}>
                    <NotFoundView />
                </NavRoute>
            </Switch>
        </>
    );
}