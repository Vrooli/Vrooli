import { LINKS } from "@local/shared";
import { Box } from "@mui/material";
import { lazily } from "react-lazily";
import { Page } from "./components/Page/Page.js";
import { ScrollToTop } from "./components/ScrollToTop.js";
import { FullPageSpinner } from "./components/Spinners.js";
import { NavbarProps } from "./components/navigation/types.js";
import { Route, RouteProps, Switch } from "./route/router.js";
import { useLayoutStore } from "./stores/layoutStore.js";
import { PageProps, ViewDisplayType } from "./types.js";
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

export interface RoutesProps {
    /** Whether the session has been checked */
    sessionChecked: boolean;
    /** How the routes should be displayed */
    display: `${ViewDisplayType}`;
}

export function Routes({ sessionChecked, display }: RoutesProps) {
    // Get the current position from the layout store
    const { positions, routeControlledPosition } = useLayoutStore();

    // Only render routes if this component is in the main position
    // This prevents routes from rendering in side panels
    if (positions[routeControlledPosition] !== "primary") {
        return null;
    }

    // Common props for all routes
    const commonProps = { sessionChecked };

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
                    {...commonProps}
                >
                    <AboutView display={display} />
                </NavRoute>
                <NavRoute path={`${LINKS.Api}/add`} excludePageContainer mustBeLoggedIn={true} {...commonProps}>
                    <ApiUpsert display={display} isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Api}/edit/:rootId/:versionId`} excludePageContainer mustBeLoggedIn={true} {...commonProps}>
                    <ApiUpsert display={display} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Api}/:rootId/:versionId?`} excludePageContainer {...commonProps}>
                    <ApiView display={display} />
                </NavRoute>
                <NavRoute path={LINKS.Awards} excludePageContainer {...commonProps}>
                    <AwardsView display={display} />
                </NavRoute>
                <NavRoute path={`${LINKS.BookmarkList}/add`} mustBeLoggedIn={true} {...commonProps}>
                    <BookmarkListUpsert display={display} isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.BookmarkList}/edit/:id`} mustBeLoggedIn={true} {...commonProps}>
                    <BookmarkListUpsert display={display} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.BookmarkList}/:id`} {...commonProps}>
                    <BookmarkListView display={display} />
                </NavRoute>
                <NavRoute path={LINKS.Calendar} excludePageContainer {...commonProps}>
                    <CalendarView display={display} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Create}
                    excludePageContainer
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    mustBeLoggedIn={true}
                    {...commonProps}
                >
                    <CreateView display={display} />
                </NavRoute>
                <NavRoute path={`${LINKS.Chat}/add`} excludePageContainer {...commonProps}>
                    <ChatCrud display={display} isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Chat}/:id`} excludePageContainer {...commonProps}>
                    <ChatCrud display={display} isCreate={false} />
                </NavRoute>
                <NavRoute
                    path={LINKS.ForgotPassword}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...commonProps}
                >
                    <ForgotPasswordView display={display} />
                </NavRoute>
                <NavRoute path={LINKS.History} excludePageContainer mustBeLoggedIn={true} {...commonProps}>
                    <HistoryView display={display} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Home}
                    sitemapIndex
                    priority={1.0}
                    changeFreq="weekly"
                    excludePageContainer
                    {...commonProps}
                >
                    <HomeView display={display} />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.Login}/:code?`}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...commonProps}
                >
                    <LoginView display={display} />
                </NavRoute>
                <NavRoute path={LINKS.MyStuff} excludePageContainer mustBeLoggedIn={true} {...commonProps}>
                    <MyStuffView display={display} />
                </NavRoute>
                <NavRoute path={`${LINKS.Note}/add`} excludePageContainer mustBeLoggedIn={true} {...commonProps}>
                    <NoteCrud display={display} isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Note}/edit/:rootId/:versionId`} excludePageContainer mustBeLoggedIn={true} {...commonProps}>
                    <NoteCrud display={display} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Note}/:rootId/:versionId?`} excludePageContainer {...commonProps}>
                    <NoteCrud display={display} isCreate={false} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Inbox}
                    excludePageContainer
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    mustBeLoggedIn={true}
                    {...commonProps}
                >
                    <InboxView display={display} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Pro}
                    excludePageContainer
                    sitemapIndex
                    priority={0.5}
                    changeFreq="weekly"
                    {...commonProps}
                >
                    <ProView display={display} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Privacy}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    excludePageContainer
                    {...commonProps}
                >
                    <PrivacyPolicyView display={display} />
                </NavRoute>
                <NavRoute path={`${LINKS.Profile}/add`} mustBeLoggedIn={true} {...commonProps}>
                    <BotUpsert display={display} isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Profile}/edit/:id`} mustBeLoggedIn={true} {...commonProps}>
                    <BotUpsert display={display} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Profile}/:id?`} sx={noSidePadding} {...commonProps}>
                    <UserView display={display} />
                </NavRoute>
                <NavRoute path={`${LINKS.DataConverter}/add`} excludePageContainer mustBeLoggedIn={true} {...commonProps}>
                    <DataConverterUpsert display={display} isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.DataConverter}/edit/:rootId/:versionId`} excludePageContainer mustBeLoggedIn={true} {...commonProps}>
                    <DataConverterUpsert display={display} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.DataConverter}/:rootId/:versionId?`} {...commonProps}>
                    <DataConverterView display={display} />
                </NavRoute>
                <NavRoute path={`${LINKS.DataStructure}/add`} excludePageContainer mustBeLoggedIn={true} {...commonProps}>
                    <DataStructureUpsert display={display} isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.DataStructure}/edit/:rootId/:versionId`} excludePageContainer mustBeLoggedIn={true} {...commonProps}>
                    <DataStructureUpsert display={display} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.DataStructure}/:rootId/:versionId?`} {...commonProps}>
                    <DataStructureView display={display} />
                </NavRoute>
                <NavRoute path={`${LINKS.Project}/add`} mustBeLoggedIn={true} {...commonProps}>
                    <ProjectCrud display={display} isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Project}/edit/:rootId/:versionId`} mustBeLoggedIn={true} {...commonProps}>
                    <ProjectCrud display={display} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Project}/:rootId/:versionId?`} {...commonProps}>
                    <ProjectCrud display={display} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Prompt}/add`} mustBeLoggedIn={true} {...commonProps}>
                    <PromptUpsert display={display} isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Prompt}/edit/:rootId/:versionId`} mustBeLoggedIn={true} {...commonProps}>
                    <PromptUpsert display={display} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Prompt}/:rootId/:versionId?`} {...commonProps}>
                    <PromptView display={display} />
                </NavRoute>
                <NavRoute path={`${LINKS.Reminder}/add`} mustBeLoggedIn={true} {...commonProps}>
                    <ReminderCrud display={display} isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Reminder}/edit/:id`} mustBeLoggedIn={true} {...commonProps}>
                    <ReminderCrud display={display} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Reminder}/:id`} {...commonProps}>
                    <ReminderCrud display={display} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Reports}/:objectType/:objectOrRootId/:versionId?`} excludePageContainer {...commonProps}>
                    <ReportsView display={display} />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.ResetPassword}/:params*`}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...commonProps}
                >
                    <ResetPasswordView display={display} />
                </NavRoute>
                <NavRoute path={`${LINKS.RoutineSingleStep}/add`} mustBeLoggedIn={true} {...commonProps}>
                    <RoutineSingleStepUpsert display={display} isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.RoutineSingleStep}/edit/:rootId/:versionId`} mustBeLoggedIn={true} {...commonProps}>
                    <RoutineSingleStepUpsert display={display} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.RoutineSingleStep}/:rootId/:versionId?`} {...commonProps}>
                    <RoutineSingleStepView display={display} />
                </NavRoute>
                <NavRoute path={`${LINKS.RoutineMultiStep}/add`} excludePageContainer mustBeLoggedIn={true} {...commonProps}>
                    <RoutineMultiStepCrud display={display} isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.RoutineMultiStep}/edit/:rootId/:versionId`} excludePageContainer mustBeLoggedIn={true} {...commonProps}>
                    <RoutineMultiStepCrud display={display} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.RoutineMultiStep}/:rootId/:versionId?`} excludePageContainer {...commonProps}>
                    <RoutineMultiStepCrud display={display} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Run}/:objectType/:id`} {...commonProps}>
                    <RunView display={display} />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.Search}/:params*`}
                    excludePageContainer
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    {...commonProps}
                >
                    <SearchView display={display} />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.SearchVersion}/:params*`}
                    excludePageContainer
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    {...commonProps}
                >
                    <SearchVersionView display={display} />
                </NavRoute>
                <NavRoute path={LINKS.Settings} mustBeLoggedIn={true} excludePageContainer {...commonProps}>
                    <SettingsView display={display} />
                </NavRoute>
                <NavRoute path={LINKS.SettingsApi} mustBeLoggedIn={true} excludePageContainer {...commonProps}>
                    <SettingsApiView display={display} />
                </NavRoute>
                <NavRoute path={LINKS.SettingsAuthentication} mustBeLoggedIn={true} excludePageContainer {...commonProps}>
                    <SettingsAuthenticationView display={display} />
                </NavRoute>
                <NavRoute path={LINKS.SettingsData} mustBeLoggedIn={true} excludePageContainer {...commonProps}>
                    <SettingsDataView display={display} />
                </NavRoute>
                <NavRoute path={LINKS.SettingsDisplay} mustBeLoggedIn={true} excludePageContainer {...commonProps}>
                    <SettingsDisplayView display={display} />
                </NavRoute>
                <NavRoute path={LINKS.SettingsNotifications} mustBeLoggedIn={true} excludePageContainer {...commonProps}>
                    <SettingsNotificationsView display={display} />
                </NavRoute>
                <NavRoute path={LINKS.SettingsPayments} mustBeLoggedIn={true} excludePageContainer {...commonProps}>
                    <SettingsPaymentView display={display} />
                </NavRoute>
                <NavRoute path={LINKS.SettingsProfile} mustBeLoggedIn={true} excludePageContainer {...commonProps}>
                    <SettingsProfileView display={display} />
                </NavRoute>
                <NavRoute path={LINKS.SettingsPrivacy} mustBeLoggedIn={true} excludePageContainer {...commonProps}>
                    <SettingsPrivacyView display={display} />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.Signup}/:code?`}
                    excludePageContainer
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    {...commonProps}
                >
                    <SignupView display={display} />
                </NavRoute>
                <NavRoute path={`${LINKS.SmartContract}/add`} mustBeLoggedIn={true} {...commonProps}>
                    <SmartContractUpsert display={display} isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.SmartContract}/edit/:rootId/:versionId`} mustBeLoggedIn={true} {...commonProps}>
                    <SmartContractUpsert display={display} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.SmartContract}/:rootId/:versionId?`} {...commonProps}>
                    <SmartContractView display={display} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Stats}
                    excludePageContainer
                    sitemapIndex
                    priority={0.5}
                    changeFreq="weekly"
                    {...commonProps}
                >
                    <StatsView display={display} />
                </NavRoute>
                <NavRoute path={`${LINKS.Team}/add`} sx={noSidePadding} mustBeLoggedIn={true} {...commonProps}>
                    <TeamUpsert display={display} isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Team}/edit/:id`} sx={noSidePadding} mustBeLoggedIn={true} {...commonProps}>
                    <TeamUpsert display={display} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Team}/:id`} sx={noSidePadding} {...commonProps}>
                    <TeamView display={display} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Terms}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    excludePageContainer
                    {...commonProps}
                >
                    <TermsView display={display} />
                </NavRoute>
                <NavRoute {...commonProps}>
                    <NotFoundView display={display} />
                </NavRoute>
            </Switch>
        </>
    );
}
