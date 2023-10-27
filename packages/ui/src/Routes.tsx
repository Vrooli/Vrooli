import { LINKS } from "@local/shared";
import { Box, useTheme } from "@mui/material";
import { FullPageSpinner } from "components/FullPageSpinner/FullPageSpinner";
import { NavbarProps } from "components/navigation/types";
import { ScrollToTop } from "components/ScrollToTop";
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
const { OrganizationUpsert, OrganizationView } = lazily(() => import("./views/objects/organization"));
const { ProjectUpsert, ProjectView } = lazily(() => import("./views/objects/project"));
const { QuestionUpsert, QuestionView } = lazily(() => import("./views/objects/question"));
const { ReminderCrud } = lazily(() => import("./views/objects/reminder"));
const { RoutineUpsert, RoutineView } = lazily(() => import("./views/objects/routine"));
const { SmartContractUpsert, SmartContractView } = lazily(() => import("./views/objects/smartContract"));
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
                    <AboutView />
                </NavRoute>
                <NavRoute path={`${LINKS.Api}/add`} mustBeLoggedIn={true} {...props}>
                    <ApiUpsert isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Api}/edit/:rootId/:versionId`} mustBeLoggedIn={true} {...props}>
                    <ApiUpsert isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Api}/:rootId/:versionId?`} {...props}>
                    <ApiView />
                </NavRoute>
                <NavRoute path={LINKS.Awards} {...props}>
                    <AwardsView />
                </NavRoute>
                <NavRoute path={`${LINKS.BookmarkList}/add`} mustBeLoggedIn={true} {...props}>
                    <BookmarkListUpsert isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.BookmarkList}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <BookmarkListUpsert isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.BookmarkList}/:id`} {...props}>
                    <BookmarkListView />
                </NavRoute>
                <NavRoute path={LINKS.Calendar} excludePageContainer {...props}>
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
                <NavRoute path={`${LINKS.Chat}/add`} excludePageContainer {...props}>
                    <ChatCrud isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Chat}/:id`} excludePageContainer {...props}>
                    <ChatCrud isCreate={false} />
                </NavRoute>
                <NavRoute
                    path={LINKS.ForgotPassword}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...props}
                >
                    <ForgotPasswordView />
                </NavRoute>
                <NavRoute path={LINKS.History} mustBeLoggedIn={true} {...props}>
                    <HistoryView />
                </NavRoute>
                <NavRoute
                    path={LINKS.Home}
                    sitemapIndex
                    priority={1.0}
                    changeFreq="weekly"
                    excludePageContainer
                    {...props}
                >
                    <HomeView />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.Login}/:code?`}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...props}
                >
                    <LoginView />
                </NavRoute>
                <NavRoute path={LINKS.MyStuff} mustBeLoggedIn={true} {...props}>
                    <MyStuffView />
                </NavRoute>
                <NavRoute path={`${LINKS.Note}/add`} excludePageContainer mustBeLoggedIn={true} {...props}>
                    <NoteCrud isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Note}/edit/:rootId/:versionId`} excludePageContainer mustBeLoggedIn={true} {...props}>
                    <NoteCrud isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Note}/:rootId/:versionId?`} excludePageContainer {...props}>
                    <NoteCrud isCreate={false} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Inbox}
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    mustBeLoggedIn={true}
                    {...props}
                >
                    <InboxView />
                </NavRoute>
                <NavRoute path={`${LINKS.Organization}/add`} sx={noSidePadding} mustBeLoggedIn={true} {...props}>
                    <OrganizationUpsert isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Organization}/edit/:id`} sx={noSidePadding} mustBeLoggedIn={true} {...props}>
                    <OrganizationUpsert isCreate={false} />
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
                    sx={{ background: palette.background.paper }}
                    {...props}
                >
                    <PrivacyPolicyView />
                </NavRoute>
                <NavRoute path={`${LINKS.Profile}/add`} mustBeLoggedIn={true} {...props}>
                    <BotUpsert isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Profile}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <BotUpsert isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Profile}/:id?`} sx={noSidePadding} {...props}>
                    <UserView />
                </NavRoute>
                <NavRoute path={`${LINKS.Project}/add`} mustBeLoggedIn={true} {...props}>
                    <ProjectUpsert isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Project}/edit/:rootId/:versionId`} mustBeLoggedIn={true} {...props}>
                    <ProjectUpsert isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Project}/:rootId/:versionId?`} {...props}>
                    <ProjectView />
                </NavRoute>
                <NavRoute path={`${LINKS.Question}/add`} mustBeLoggedIn={true} {...props}>
                    <QuestionUpsert isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Question}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <QuestionUpsert isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Question}/:id`} {...props}>
                    <QuestionView />
                </NavRoute>
                <NavRoute path={`${LINKS.Reminder}/add`} mustBeLoggedIn={true} {...props}>
                    <ReminderCrud isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Reminder}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <ReminderCrud isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Reminder}/:id`} {...props}>
                    <ReminderCrud isCreate={false} />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.ResetPassword}/:params*`}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...props}
                >
                    <ResetPasswordView />
                </NavRoute>
                <NavRoute path={`${LINKS.Routine}/add`} mustBeLoggedIn={true} {...props}>
                    <RoutineUpsert isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Routine}/edit/:rootId/:versionId`} mustBeLoggedIn={true} {...props}>
                    <RoutineUpsert isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Routine}/:rootId/:versionId?`} {...props}>
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
                <NavRoute path={LINKS.SettingsApi} mustBeLoggedIn={true} {...props}>
                    <SettingsApiView />
                </NavRoute>
                <NavRoute path={LINKS.SettingsAuthentication} mustBeLoggedIn={true} {...props}>
                    <SettingsAuthenticationView />
                </NavRoute>
                <NavRoute path={LINKS.SettingsData} mustBeLoggedIn={true} {...props}>
                    <SettingsDataView />
                </NavRoute>
                <NavRoute path={LINKS.SettingsDisplay} mustBeLoggedIn={true} {...props}>
                    <SettingsDisplayView />
                </NavRoute>
                <NavRoute path={LINKS.SettingsNotifications} mustBeLoggedIn={true} {...props}>
                    <SettingsNotificationsView />
                </NavRoute>
                <NavRoute path={LINKS.SettingsPayments} mustBeLoggedIn={true} {...props}>
                    <SettingsPaymentView />
                </NavRoute>
                <NavRoute path={LINKS.SettingsProfile} mustBeLoggedIn={true} {...props}>
                    <SettingsProfileView />
                </NavRoute>
                <NavRoute path={LINKS.SettingsPrivacy} mustBeLoggedIn={true} {...props}>
                    <SettingsPrivacyView />
                </NavRoute>
                <NavRoute path={LINKS.SettingsFocusModes} mustBeLoggedIn={true} {...props}>
                    <SettingsFocusModesView />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.Signup}/:code?`}
                    excludePageContainer
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    {...props}
                >
                    <SignupView />
                </NavRoute>
                <NavRoute path={`${LINKS.SmartContract}/add`} mustBeLoggedIn={true} {...props}>
                    <SmartContractUpsert isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.SmartContract}/edit/:rootId/:versionId`} mustBeLoggedIn={true} {...props}>
                    <SmartContractUpsert isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.SmartContract}/:rootId/:versionId?`} {...props}>
                    <SmartContractView />
                </NavRoute>
                <NavRoute path={`${LINKS.Standard}/add`} mustBeLoggedIn={true} {...props}>
                    <StandardUpsert isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Standard}/edit/:rootId/:versionId`} mustBeLoggedIn={true} {...props}>
                    <StandardUpsert isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Standard}/:rootId/:versionId?`} {...props}>
                    <StandardView />
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
                    sx={{ background: palette.background.paper }}
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
};
