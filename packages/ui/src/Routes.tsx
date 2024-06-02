import { LINKS } from "@local/shared";
import { Box, useTheme } from "@mui/material";
import { FullPageSpinner } from "components/FullPageSpinner/FullPageSpinner";
import { ScrollToTop } from "components/ScrollToTop";
import { NavbarProps } from "components/navigation/types";
import { lazily } from "react-lazily";
import { Route, RouteProps, Switch } from "route";
import { BotUpsert } from "views/objects/bot";
import { PageProps } from "views/types";
import { Page } from "./components/Page/Page";

// Lazy loading in the Routes component is a recommended way to improve performance. See https://reactjs.org/docs/code-splitting.html#route-based-code-splitting
const {
    HomeView,
    HistoryView,
    CreateView,
    MyStuffView,
    InboxView,
} = lazily(() => import("./views/main"));
const {
    ForgotPasswordView,
    LoginView,
    ResetPasswordView,
    SignupView,
} = lazily(() => import("./views/auth"));
const {
    SettingsView,
    SettingsApiView,
    SettingsAuthenticationView,
    SettingsDataView,
    SettingsDisplayView,
    SettingsNotificationsView,
    SettingsPaymentView,
    SettingsProfileView,
    SettingsPrivacyView,
    SettingsFocusModesView,
} = lazily(() => import("./views/settings"));
const {
    PrivacyPolicyView,
    TermsView,
} = lazily(() => import("./views/legal"));
const { AboutView } = lazily(() => import("./views/AboutView/AboutView"));
const { AwardsView } = lazily(() => import("./views/AwardsView/AwardsView"));
const { CalendarView } = lazily(() => import("./views/CalendarView/CalendarView"));
const { ChatCrud } = lazily(() => import("./views/objects/chat"));
const { NotFoundView } = lazily(() => import("./views/NotFoundView/NotFoundView"));
const { PremiumView } = lazily(() => import("./views/PremiumView/PremiumView"));
const { SearchView } = lazily(() => import("./views/SearchView/SearchView"));
const { StatsSiteView: StatsView } = lazily(() => import("./views/StatsSiteView/StatsSiteView"));
const { ApiUpsert, ApiView } = lazily(() => import("./views/objects/api"));
const { BookmarkListUpsert, BookmarkListView } = lazily(() => import("./views/objects/bookmarkList"));
const { NoteCrud } = lazily(() => import("./views/objects/note"));
const { TeamUpsert, TeamView } = lazily(() => import("./views/objects/team"));
const { ProjectCrud } = lazily(() => import("./views/objects/project"));
const { QuestionUpsert, QuestionView } = lazily(() => import("./views/objects/question"));
const { ReminderCrud } = lazily(() => import("./views/objects/reminder"));
const { RoutineUpsert, RoutineView } = lazily(() => import("./views/objects/routine"));
const { CodeUpsert, CodeView } = lazily(() => import("./views/objects/code"));
const { StandardUpsert, StandardView } = lazily(() => import("./views/objects/standard"));
const { UserView } = lazily(() => import("./views/objects/user"));

/**
 * Fallback displayed while route is being loaded.
 */
const Fallback = <Box>
    {/* A blank Navbar to display before the actual one (which is dynamic depending on the page) is rendered. */}
    <Box sx={{
        background: (t) => t.palette.primary.dark,
        height: "64px!important",
        zIndex: 1000,
        position: "fixed",
        top: 0,
        left: 0,
        right: 0,
    }} />
    {/* Loading spinner */}
    <FullPageSpinner />
</Box>;

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
    );
};

/** Style for pages that don't use left/right padding */
const noSidePadding = {
    paddingLeft: 0,
    paddingRight: 0,
};

export const Routes = (props: { sessionChecked: boolean }) => {
    const { palette } = useTheme();

    return (
        <>
            <ScrollToTop />
            <Switch fallback={Fallback}>
                <NavRoute
                    path={LINKS.About}
                    sitemapIndex
                    priority={0.5}
                    changeFreq="monthly"
                    sx={{ background: palette.background.paper }}
                    {...props}
                >
                    <AboutView display="page" />
                </NavRoute>
                <NavRoute path={`${LINKS.Api}/add`} mustBeLoggedIn={true} {...props}>
                    <ApiUpsert display="page" isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Api}/edit/:rootId/:versionId`} mustBeLoggedIn={true} {...props}>
                    <ApiUpsert display="page" isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Api}/:rootId/:versionId?`} {...props}>
                    <ApiView display="page" />
                </NavRoute>
                <NavRoute path={LINKS.Awards} {...props}>
                    <AwardsView display="page" />
                </NavRoute>
                <NavRoute path={`${LINKS.BookmarkList}/add`} mustBeLoggedIn={true} {...props}>
                    <BookmarkListUpsert display="page" isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.BookmarkList}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <BookmarkListUpsert display="page" isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.BookmarkList}/:id`} {...props}>
                    <BookmarkListView display="page" />
                </NavRoute>
                <NavRoute path={LINKS.Calendar} excludePageContainer {...props}>
                    <CalendarView display="page" />
                </NavRoute>
                <NavRoute
                    path={LINKS.Create}
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    mustBeLoggedIn={true}
                    {...props}
                >
                    <CreateView display="page" />
                </NavRoute>
                <NavRoute path={`${LINKS.Chat}/add`} excludePageContainer {...props}>
                    <ChatCrud display="page" isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Chat}/:id`} excludePageContainer {...props}>
                    <ChatCrud display="page" isCreate={false} />
                </NavRoute>
                <NavRoute
                    path={LINKS.ForgotPassword}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...props}
                >
                    <ForgotPasswordView display="page" />
                </NavRoute>
                <NavRoute path={LINKS.History} mustBeLoggedIn={true} {...props}>
                    <HistoryView display="page" />
                </NavRoute>
                <NavRoute
                    path={LINKS.Home}
                    sitemapIndex
                    priority={1.0}
                    changeFreq="weekly"
                    excludePageContainer
                    {...props}
                >
                    <HomeView display="page" />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.Login}/:code?`}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...props}
                >
                    <LoginView display="page" />
                </NavRoute>
                <NavRoute path={LINKS.MyStuff} mustBeLoggedIn={true} {...props}>
                    <MyStuffView display="page" />
                </NavRoute>
                <NavRoute path={`${LINKS.Note}/add`} excludePageContainer mustBeLoggedIn={true} {...props}>
                    <NoteCrud display="page" isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Note}/edit/:rootId/:versionId`} excludePageContainer mustBeLoggedIn={true} {...props}>
                    <NoteCrud display="page" isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Note}/:rootId/:versionId?`} excludePageContainer {...props}>
                    <NoteCrud display="page" isCreate={false} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Inbox}
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    mustBeLoggedIn={true}
                    {...props}
                >
                    <InboxView display="page" />
                </NavRoute>
                <NavRoute
                    path={LINKS.Pro}
                    excludePageContainer
                    sitemapIndex
                    priority={0.5}
                    changeFreq="weekly"
                    {...props}
                >
                    <PremiumView display="page" />
                </NavRoute>
                <NavRoute
                    path={LINKS.Privacy}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...props}
                >
                    <PrivacyPolicyView display="page" />
                </NavRoute>
                <NavRoute path={`${LINKS.Profile}/add`} mustBeLoggedIn={true} {...props}>
                    <BotUpsert display="page" isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Profile}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <BotUpsert display="page" isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Profile}/:id?`} sx={noSidePadding} {...props}>
                    <UserView display="page" />
                </NavRoute>
                <NavRoute path={`${LINKS.Code}/add`} mustBeLoggedIn={true} {...props}>
                    <CodeUpsert display="page" isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Code}/edit/:rootId/:versionId`} mustBeLoggedIn={true} {...props}>
                    <CodeUpsert display="page" isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Code}/:rootId/:versionId?`} {...props}>
                    <CodeView display="page" />
                </NavRoute>
                <NavRoute path={`${LINKS.Project}/add`} mustBeLoggedIn={true} {...props}>
                    <ProjectCrud display="page" isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Project}/edit/:rootId/:versionId`} mustBeLoggedIn={true} {...props}>
                    <ProjectCrud display="page" isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Project}/:rootId/:versionId?`} {...props}>
                    <ProjectCrud display="page" isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Question}/add`} mustBeLoggedIn={true} {...props}>
                    <QuestionUpsert display="page" isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Question}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <QuestionUpsert display="page" isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Question}/:id`} {...props}>
                    <QuestionView display="page" />
                </NavRoute>
                <NavRoute path={`${LINKS.Reminder}/add`} mustBeLoggedIn={true} {...props}>
                    <ReminderCrud display="page" isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Reminder}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <ReminderCrud display="page" isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Reminder}/:id`} {...props}>
                    <ReminderCrud display="page" isCreate={false} />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.ResetPassword}/:params*`}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...props}
                >
                    <ResetPasswordView display="page" />
                </NavRoute>
                <NavRoute path={`${LINKS.Routine}/add`} mustBeLoggedIn={true} {...props}>
                    <RoutineUpsert display="page" isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Routine}/edit/:rootId/:versionId`} mustBeLoggedIn={true} {...props}>
                    <RoutineUpsert display="page" isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Routine}/:rootId/:versionId?`} {...props}>
                    <RoutineView display="page" />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.Search}/:params*`}
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    {...props}
                >
                    <SearchView display="page" />
                </NavRoute>
                <NavRoute path={LINKS.Settings} mustBeLoggedIn={true} {...props}>
                    <SettingsView display="page" />
                </NavRoute>
                <NavRoute path={LINKS.SettingsApi} mustBeLoggedIn={true} {...props}>
                    <SettingsApiView display="page" />
                </NavRoute>
                <NavRoute path={LINKS.SettingsAuthentication} mustBeLoggedIn={true} {...props}>
                    <SettingsAuthenticationView display="page" />
                </NavRoute>
                <NavRoute path={LINKS.SettingsData} mustBeLoggedIn={true} {...props}>
                    <SettingsDataView display="page" />
                </NavRoute>
                <NavRoute path={LINKS.SettingsDisplay} mustBeLoggedIn={true} {...props}>
                    <SettingsDisplayView display="page" />
                </NavRoute>
                <NavRoute path={LINKS.SettingsNotifications} mustBeLoggedIn={true} {...props}>
                    <SettingsNotificationsView display="page" />
                </NavRoute>
                <NavRoute path={LINKS.SettingsPayments} mustBeLoggedIn={true} {...props}>
                    <SettingsPaymentView display="page" />
                </NavRoute>
                <NavRoute path={LINKS.SettingsProfile} mustBeLoggedIn={true} {...props}>
                    <SettingsProfileView display="page" />
                </NavRoute>
                <NavRoute path={LINKS.SettingsPrivacy} mustBeLoggedIn={true} {...props}>
                    <SettingsPrivacyView display="page" />
                </NavRoute>
                <NavRoute path={LINKS.SettingsFocusModes} mustBeLoggedIn={true} {...props}>
                    <SettingsFocusModesView display="page" />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.Signup}/:code?`}
                    excludePageContainer
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    {...props}
                >
                    <SignupView display="page" />
                </NavRoute>
                <NavRoute path={`${LINKS.Standard}/add`} mustBeLoggedIn={true} {...props}>
                    <StandardUpsert display="page" isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Standard}/edit/:rootId/:versionId`} mustBeLoggedIn={true} {...props}>
                    <StandardUpsert display="page" isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Standard}/:rootId/:versionId?`} {...props}>
                    <StandardView display="page" />
                </NavRoute>
                <NavRoute
                    path={LINKS.Stats}
                    sitemapIndex
                    priority={0.5}
                    changeFreq="weekly"
                    {...props}
                >
                    <StatsView display="page" />
                </NavRoute>
                <NavRoute path={`${LINKS.Team}/add`} sx={noSidePadding} mustBeLoggedIn={true} {...props}>
                    <TeamUpsert display="page" isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Team}/edit/:id`} sx={noSidePadding} mustBeLoggedIn={true} {...props}>
                    <TeamUpsert display="page" isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Team}/:id`} sx={noSidePadding} {...props}>
                    <TeamView display="page" />
                </NavRoute>
                <NavRoute
                    path={LINKS.Terms}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...props}
                >
                    <TermsView display="page" />
                </NavRoute>
                <NavRoute {...props}>
                    <NotFoundView />
                </NavRoute>
            </Switch>
        </>
    );
};
