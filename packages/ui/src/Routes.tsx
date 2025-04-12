import { LINKS } from "@local/shared";
import { Box } from "@mui/material";
import { lazily } from "react-lazily";
import { Page } from "./components/Page/Page.js";
import { ScrollToTop } from "./components/ScrollToTop.js";
import { FullPageSpinner } from "./components/Spinners.js";
import { NavbarProps } from "./components/navigation/types.js";
import { Route, RouteProps, Switch } from "./route/router.js";
import { useLayoutStore } from "./stores/layoutStore.js";
import { PageProps } from "./types.js";
import { BotUpsert } from "./views/objects/bot/BotUpsert.js";

// Lazy loading in the Routes component is a recommended way to improve performance. See https://reactjs.org/docs/code-splitting.html#route-based-code-splitting
const {
    HomeView,
    HistoryView,
    CreateView,
    MyStuffView,
    InboxView,
} = lazily(() => import("./views/main/index.js"));
const {
    ForgotPasswordView,
    LoginView,
    ResetPasswordView,
    SignupView,
} = lazily(() => import("./views/auth/index.js"));
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
} = lazily(() => import("./views/settings/index.js"));
const {
    PrivacyPolicyView,
    TermsView,
} = lazily(() => import("./views/PolicyView/PolicyView.js"));
const { AboutView } = lazily(() => import("./views/AboutView/AboutView.js"));
const { AwardsView } = lazily(() => import("./views/AwardsView/AwardsView.js"));
const { CalendarView } = lazily(() => import("./views/CalendarView/CalendarView.js"));
const { ChatCrud } = lazily(() => import("./views/objects/chat/ChatCrud.js"));
const { DataConverterUpsert, DataConverterView } = lazily(() => import("./views/objects/dataConverter/index.js"));
const { DataStructureUpsert, DataStructureView } = lazily(() => import("./views/objects/dataStructure/index.js"));
const { NotFoundView } = lazily(() => import("./views/NotFoundView/NotFoundView.js"));
const { ProView } = lazily(() => import("./views/ProView/ProView.js"));
const { SearchView } = lazily(() => import("./views/SearchView/SearchView.js"));
const { SearchVersionView } = lazily(() => import("./views/SearchVersionView/SearchVersionView.js"));
const { StatsSiteView: StatsView } = lazily(() => import("./views/StatsSiteView/StatsSiteView.js"));
const { ApiUpsert, ApiView } = lazily(() => import("./views/objects/api/index.js"));
const { BookmarkListUpsert, BookmarkListView } = lazily(() => import("./views/objects/bookmarkList/index.js"));
const { NoteCrud } = lazily(() => import("./views/objects/note/index.js"));
const { TeamUpsert, TeamView } = lazily(() => import("./views/objects/team/index.js"));
const { ProjectCrud } = lazily(() => import("./views/objects/project/index.js"));
const { PromptUpsert, PromptView } = lazily(() => import("./views/objects/prompt/index.js"));
const { QuestionUpsert, QuestionView } = lazily(() => import("./views/objects/question/index.js"));
const { ReminderCrud } = lazily(() => import("./views/objects/reminder/index.js"));
const { RoutineSingleStepUpsert, RoutineMultiStepCrud, RoutineSingleStepView } = lazily(() => import("./views/objects/routine/index.js"));
const { SmartContractUpsert, SmartContractView } = lazily(() => import("./views/objects/smartContract/index.js"));
const { UserView } = lazily(() => import("./views/objects/user/UserView.js"));
const { RunView } = lazily(() => import("./views/runs/RunView.js"));
const { ReportsView } = lazily(() => import("./views/ReportsView/ReportsView.js"));

/**
 * Fallback displayed while route is being loaded.
 */
const Fallback = <Box>
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
    // Get the current position from the layout store
    const { positions, routeControlledPosition } = useLayoutStore();

    // Only render routes if this component is in the main position
    // This prevents routes from rendering in side panels
    if (positions[routeControlledPosition] !== "primary") {
        return null;
    }

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
                    <AboutView />
                </NavRoute>
                <NavRoute path={`${LINKS.Api}/add`} mustBeLoggedIn={true} {...props}>
                    <ApiUpsert display="page" isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Api}/edit/:rootId/:versionId`} mustBeLoggedIn={true} {...props}>
                    <ApiUpsert display="page" isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Api}/:rootId/:versionId?`} excludePageContainer {...props}>
                    <ApiView display="page" />
                </NavRoute>
                <NavRoute path={LINKS.Awards} excludePageContainer {...props}>
                    <AwardsView />
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
                    excludePageContainer
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    mustBeLoggedIn={true}
                    {...props}
                >
                    <CreateView />
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
                <NavRoute path={LINKS.History} excludePageContainer mustBeLoggedIn={true} {...props}>
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
                <NavRoute path={LINKS.MyStuff} excludePageContainer mustBeLoggedIn={true} {...props}>
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
                    excludePageContainer
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
                    <ProView />
                </NavRoute>
                <NavRoute
                    path={LINKS.Privacy}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    excludePageContainer
                    {...props}
                >
                    <PrivacyPolicyView />
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
                <NavRoute path={`${LINKS.Reports}/:objectType/:objectOrRootId/:versionId?`} excludePageContainer {...props}>
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
                <NavRoute path={`${LINKS.RoutineSingleStep}/add`} mustBeLoggedIn={true} {...props}>
                    <RoutineSingleStepUpsert display="page" isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.RoutineSingleStep}/edit/:rootId/:versionId`} mustBeLoggedIn={true} {...props}>
                    <RoutineSingleStepUpsert display="page" isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.RoutineSingleStep}/:rootId/:versionId?`} {...props}>
                    <RoutineSingleStepView display="page" />
                </NavRoute>
                <NavRoute path={`${LINKS.RoutineMultiStep}/add`} excludePageContainer mustBeLoggedIn={true} {...props}>
                    <RoutineMultiStepCrud display="page" isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.RoutineMultiStep}/edit/:rootId/:versionId`} excludePageContainer mustBeLoggedIn={true} {...props}>
                    <RoutineMultiStepCrud display="page" isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.RoutineMultiStep}/:rootId/:versionId?`} excludePageContainer {...props}>
                    <RoutineMultiStepCrud display="page" isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Run}/:objectType/:id`} {...props}>
                    <RunView display="page" />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.Search}/:params*`}
                    excludePageContainer
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    {...props}
                >
                    <SearchView display="page" />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.SearchVersion}/:params*`}
                    excludePageContainer
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    {...props}
                >
                    <SearchVersionView display="page" />
                </NavRoute>
                <NavRoute path={LINKS.Settings} mustBeLoggedIn={true} excludePageContainer {...props}>
                    <SettingsView />
                </NavRoute>
                <NavRoute path={LINKS.SettingsApi} mustBeLoggedIn={true} excludePageContainer {...props}>
                    <SettingsApiView display="page" />
                </NavRoute>
                <NavRoute path={LINKS.SettingsAuthentication} mustBeLoggedIn={true} excludePageContainer {...props}>
                    <SettingsAuthenticationView display="page" />
                </NavRoute>
                <NavRoute path={LINKS.SettingsData} mustBeLoggedIn={true} excludePageContainer {...props}>
                    <SettingsDataView display="page" />
                </NavRoute>
                <NavRoute path={LINKS.SettingsDisplay} mustBeLoggedIn={true} excludePageContainer {...props}>
                    <SettingsDisplayView display="page" />
                </NavRoute>
                <NavRoute path={LINKS.SettingsNotifications} mustBeLoggedIn={true} excludePageContainer {...props}>
                    <SettingsNotificationsView display="page" />
                </NavRoute>
                <NavRoute path={LINKS.SettingsPayments} mustBeLoggedIn={true} excludePageContainer {...props}>
                    <SettingsPaymentView display="page" />
                </NavRoute>
                <NavRoute path={LINKS.SettingsProfile} mustBeLoggedIn={true} excludePageContainer {...props}>
                    <SettingsProfileView display="page" />
                </NavRoute>
                <NavRoute path={LINKS.SettingsPrivacy} mustBeLoggedIn={true} excludePageContainer {...props}>
                    <SettingsPrivacyView display="page" />
                </NavRoute>
                <NavRoute path={LINKS.SettingsFocusModes} mustBeLoggedIn={true} excludePageContainer {...props}>
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
                    excludePageContainer
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
                    <TermsView />
                </NavRoute>
                <NavRoute {...props}>
                    <NotFoundView />
                </NavRoute>
            </Switch>
        </>
    );
}
