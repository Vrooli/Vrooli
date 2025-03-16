import { LINKS } from "@local/shared";
import { Box } from "@mui/material";
import { lazily } from "react-lazily";
import { Page } from "./components/Page/Page.js";
import { ScrollToTop } from "./components/ScrollToTop.js";
import { FullPageSpinner } from "./components/Spinners/Spinners.js";
import { NavbarProps } from "./components/navigation/types.js";
import { Route, RouteProps, Switch } from "./route/router.js";
import { PageProps } from "./types.js";
import { BotUpsert } from "./views/objects/bot/BotUpsert.js";

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
} = lazily(() => import("./views/PolicyView/PolicyView"));
const { AboutView } = lazily(() => import("./views/AboutView/AboutView"));
const { AwardsView } = lazily(() => import("./views/AwardsView/AwardsView"));
const { CalendarView } = lazily(() => import("./views/CalendarView/CalendarView"));
const { ChatCrud } = lazily(() => import("./views/objects/chat"));
const { DataConverterUpsert, DataConverterView } = lazily(() => import("./views/objects/dataConverter"));
const { DataStructureUpsert, DataStructureView } = lazily(() => import("./views/objects/dataStructure"));
const { NotFoundView } = lazily(() => import("./views/NotFoundView/NotFoundView"));
const { ProView } = lazily(() => import("./views/ProView/ProView"));
const { SearchView } = lazily(() => import("./views/SearchView/SearchView"));
const { SearchVersionView } = lazily(() => import("./views/SearchVersionView/SearchVersionView"));
const { StatsSiteView: StatsView } = lazily(() => import("./views/StatsSiteView/StatsSiteView"));
const { ApiUpsert, ApiView } = lazily(() => import("./views/objects/api"));
const { BookmarkListUpsert, BookmarkListView } = lazily(() => import("./views/objects/bookmarkList"));
const { NoteCrud } = lazily(() => import("./views/objects/note"));
const { TeamUpsert, TeamView } = lazily(() => import("./views/objects/team"));
const { ProjectCrud } = lazily(() => import("./views/objects/project"));
const { PromptUpsert, PromptView } = lazily(() => import("./views/objects/prompt"));
const { QuestionUpsert, QuestionView } = lazily(() => import("./views/objects/question"));
const { ReminderCrud } = lazily(() => import("./views/objects/reminder"));
const { RoutineUpsert, RoutineView } = lazily(() => import("./views/objects/routine"));
const { SmartContractUpsert, SmartContractView } = lazily(() => import("./views/objects/smartContract"));
const { UserView } = lazily(() => import("./views/objects/user"));
const { RunView } = lazily(() => import("./views/runs/RunView/RunView"));
const { ReportsView } = lazily(() => import("./views/ReportsView/ReportsView"));

const fallbackNavbarStyle = {
    background: (t) => t.palette.primary.dark,
    height: "64px!important",
    zIndex: 1000,
    position: "fixed",
    top: 0,
    left: 0,
    right: 0,
} as const;

/**
 * Fallback displayed while route is being loaded.
 */
const Fallback = <Box>
    {/* A blank Navbar to display before the actual one (which is dynamic depending on the page) is rendered. */}
    <Box sx={fallbackNavbarStyle} />
    {/* Loading spinner */}
    <FullPageSpinner />
</Box>;

/**
 * Wrapper to define Navbar within each route. This allows us to 
 * customize the Navbar for each route, without it flashing on
 * the screen before the route is loaded.
 */
function NavRoute(props: PageProps & RouteProps & NavbarProps) {
    return (
        <Route {...props}>
            <Page {...props}>
                {props.children}
            </Page>
        </Route>
    );
}

/** Style for pages that don't use left/right padding */
const noSidePadding = {
    paddingLeft: 0,
    paddingRight: 0,
};

export function Routes(props: { sessionChecked: boolean }) {
    return (
        <>
            <ScrollToTop />
            <Switch fallback={Fallback}>
                <NavRoute
                    path={LINKS.About}
                    sitemapIndex
                    priority={0.5}
                    changeFreq="monthly"
                    excludePageContainer
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
                    <ProView display="page" />
                </NavRoute>
                <NavRoute
                    path={LINKS.Privacy}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    excludePageContainer
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
                <NavRoute path={`${LINKS.DataConverter}/add`} mustBeLoggedIn={true} {...props}>
                    <DataConverterUpsert display="page" isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.DataConverter}/edit/:rootId/:versionId`} mustBeLoggedIn={true} {...props}>
                    <DataConverterUpsert display="page" isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.DataConverter}/:rootId/:versionId?`} {...props}>
                    <DataConverterView display="page" />
                </NavRoute>
                <NavRoute path={`${LINKS.DataStructure}/add`} mustBeLoggedIn={true} {...props}>
                    <DataStructureUpsert display="page" isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.DataStructure}/edit/:rootId/:versionId`} mustBeLoggedIn={true} {...props}>
                    <DataStructureUpsert display="page" isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.DataStructure}/:rootId/:versionId?`} {...props}>
                    <DataStructureView display="page" />
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
                <NavRoute path={`${LINKS.Prompt}/add`} mustBeLoggedIn={true} {...props}>
                    <PromptUpsert display="page" isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Prompt}/edit/:rootId/:versionId`} mustBeLoggedIn={true} {...props}>
                    <PromptUpsert display="page" isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Prompt}/:rootId/:versionId?`} {...props}>
                    <PromptView display="page" />
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
                <NavRoute path={`${LINKS.Reports}/:objectType/:objectOrRootId/:versionId?`} {...props}>
                    <ReportsView />
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
                <NavRoute path={`${LINKS.Run}/:objectType/:id`} {...props}>
                    <RunView display="page" />
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
                <NavRoute
                    path={`${LINKS.SearchVersion}/:params*`}
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    {...props}
                >
                    <SearchVersionView display="page" />
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
                <NavRoute path={`${LINKS.SmartContract}/add`} mustBeLoggedIn={true} {...props}>
                    <SmartContractUpsert display="page" isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.SmartContract}/edit/:rootId/:versionId`} mustBeLoggedIn={true} {...props}>
                    <SmartContractUpsert display="page" isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.SmartContract}/:rootId/:versionId?`} {...props}>
                    <SmartContractView display="page" />
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
                    excludePageContainer
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
}
