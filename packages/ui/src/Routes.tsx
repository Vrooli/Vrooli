import { useCallback } from 'react';
import { lazily } from 'react-lazily';
import { Route, RouteProps, Switch } from '@shared/route';
import { APP_LINKS as LINKS, BUSINESS_NAME } from '@shared/consts';
import {
    ForgotPasswordForm,
    ResetPasswordForm
} from 'forms';
import { Navbar, ScrollToTop } from 'components';
import { CommonProps } from 'types';
import { Page } from './components/Page/Page';
import { Box, CircularProgress } from '@mui/material';
import { guestSession } from 'utils/authentication';
import { PageProps } from 'views/wrapper/types';
import { NavbarProps } from 'components/navigation/types';

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

/**
 * Loading spinner displayed when route is being loaded.
 */
const Fallback = <Box sx={{
    position: 'absolute',
    top: '50%',
    left: '50%',
    transform: 'translate(-50%, -50%)',
    zIndex: 100000,
}}>
    <CircularProgress size={100} />
</Box>

/**
 * Wrapper to define Navbar within each route. This allows us to 
 * customize the Navbar for each route, without it flashing on
 * the screen before the route is loaded.
 */
//TODO once this is verified to work (including sitemap generation), pass in ViewUI instead of normal children
const NavRoute = (props: PageProps & RouteProps & NavbarProps) => {
    return (
        <Route {...props}>
            <>
                <Navbar session={props.session} sessionChecked={props.session !== undefined} />
                <Page {...props}>
                    {props.children}
                </Page>
            </>
        </Route>
    )
}

export const Routes = (props: CommonProps) => {
    // Tab title for static (non-dynamic) pages (e.g. Home, Search, Create, Notifications).
    const title = useCallback((page: string) => `${page} | ${BUSINESS_NAME}`, []);

    return (
        <>
            <ScrollToTop />
            <Switch fallback={Fallback}>
                <NavRoute path={`${LINKS.Api}/add`} mustBeLoggedIn={true} {...props}>
                    <ApiCreate session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.Api}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <ApiUpdate session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.Api}/:id`} {...props}>
                    <ApiView session={props.session} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Awards}
                    sitemapIndex
                    priority={0.5}
                    changeFreq="weekly"
                    {...props}
                >
                    <AwardsView {...props} />
                </NavRoute>
                <NavRoute path={LINKS.Calendar} {...props}>
                    <CalendarView session={props.session} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Create}
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    mustBeLoggedIn={true}
                    {...props}
                >
                    <CreateView session={props.session} />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.ForgotPassword}/:code?`}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...props}
                >
                    <FormView title="Forgot Password" maxWidth="700px">
                        <ForgotPasswordForm />
                    </FormView>
                </NavRoute>
                <NavRoute path={LINKS.History} mustBeLoggedIn={true} {...props}>
                    <HistoryView session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.HistorySearch}/:params*`} mustBeLoggedIn={true} {...props}>
                    <HistorySearchView session={props.session} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Home}
                    sitemapIndex
                    priority={1.0}
                    changeFreq="weekly"
                    {...props}
                >
                    <HomeView session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.Note}/add`} mustBeLoggedIn={true} {...props}>
                    <NoteCreate session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.Note}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <NoteUpdate session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.Note}/:id`} {...props}>
                    <NoteView session={props.session} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Notifications}
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    mustBeLoggedIn={true}
                    {...props}
                >
                    <NotificationsView session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.Organization}/add`} mustBeLoggedIn={true} {...props}>
                    <OrganizationCreate session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.Organization}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <OrganizationUpdate session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.Organization}/:id`} {...props}>
                    <OrganizationView session={props.session} />
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
                <NavRoute path={`${LINKS.Profile}/:id?`} {...props}>
                    <UserView session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.Project}/add`} mustBeLoggedIn={true} {...props}>
                    <ProjectCreate session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.Project}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <ProjectUpdate session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.Project}/:id`} {...props}>
                    <ProjectView session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.Question}/add`} mustBeLoggedIn={true} {...props}>
                    <QuestionCreate session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.Question}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <QuestionUpdate session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.Question}/:id`} {...props}>
                    <QuestionView session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.Reminder}/add`} mustBeLoggedIn={true} {...props}>
                    <ReminderCreate session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.Reminder}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <ReminderUpdate session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.Reminder}/:id`} {...props}>
                    <ReminderView session={props.session} />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.ResetPassword}/:params*`}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...props}
                >
                    <FormView title="Reset Password" maxWidth="700px">
                        <ResetPasswordForm />
                    </FormView>
                </NavRoute>
                <NavRoute path={`${LINKS.Routine}/add`} mustBeLoggedIn={true} {...props}>
                    <RoutineCreate session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.Routine}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <RoutineUpdate session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.Routine}/:id`} {...props}>
                    <RoutineView session={props.session} />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.Search}/:params*`}
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    {...props}
                >
                    <SearchView session={props.session} />
                </NavRoute>
                <NavRoute path={LINKS.Settings} mustBeLoggedIn={true} {...props}>
                    <SettingsView session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.SmartContract}/add`} mustBeLoggedIn={true} {...props}>
                    <SmartContractCreate session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.SmartContract}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <SmartContractUpdate session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.SmartContract}/:id`} {...props}>
                    <SmartContractView session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.Standard}/add`} mustBeLoggedIn={true} {...props}>
                    <StandardCreate session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.Standard}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <StandardUpdate session={props.session} />
                </NavRoute>
                <NavRoute path={`${LINKS.Standard}/:id`} {...props}>
                    <StandardView session={props.session} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Start}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...props}
                >
                    <StartView session={props.session} />
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
                    sx={{
                        display: 'flex',
                        justifyContent: 'center',
                        alignItems: 'center',
                        textAlign: 'center',
                    }}
                    {...props}
                >
                    <WelcomeView {...props} />
                </NavRoute>
                <NavRoute  {...props}>
                    <NotFoundView />
                </NavRoute>
            </Switch>
        </>
    );
}