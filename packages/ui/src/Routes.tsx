import { LINKS } from "@local/shared";
import { Box, useTheme } from "@mui/material";
import { FullPageSpinner } from "components/FullPageSpinner/FullPageSpinner";
import { NavbarProps } from "components/navigation/types";
import { ScrollToTop } from "components/ScrollToTop";
import { ForgotPasswordForm, ResetPasswordForm } from "forms/auth";
import { lazily } from "react-lazily";
import { Route, RouteProps, Switch } from "route";
import { BotUpsert } from "views/objects/bot";
import { PageProps } from "views/wrapper/types";
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
    SettingsView,
    SettingsAuthenticationView,
    SettingsDisplayView,
    SettingsNotificationsView,
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
const { ChatView } = lazily(() => import("./views/ChatView/ChatView"));
const { FormView } = lazily(() => import("./views/wrapper/FormView"));
const { NotFoundView } = lazily(() => import("./views/NotFoundView/NotFoundView"));
const { PremiumView } = lazily(() => import("./views/PremiumView/PremiumView"));
const { SearchView } = lazily(() => import("./views/SearchView/SearchView"));
const { StartView } = lazily(() => import("./views/StartView/StartView"));
const { StatsView } = lazily(() => import("./views/StatsView/StatsView"));
const { ApiUpsert, ApiView } = lazily(() => import("./views/objects/api"));
const { BookmarkListUpsert, BookmarkListView } = lazily(() => import("./views/objects/bookmarkList"));
const { NoteUpsert, NoteView } = lazily(() => import("./views/objects/note"));
const { OrganizationUpsert, OrganizationView } = lazily(() => import("./views/objects/organization"));
const { ProjectUpsert, ProjectView } = lazily(() => import("./views/objects/project"));
const { QuestionUpsert, QuestionView } = lazily(() => import("./views/objects/question"));
const { ReminderUpsert, ReminderView } = lazily(() => import("./views/objects/reminder"));
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
        zIndex: 100,
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

const zIndex = 200;

/** Style for pages that don't use left/right padding */
const noSidePadding = {
    paddingLeft: 0,
    paddingRight: 0,
};

const viewProps = ({
    display: "page" as const,
    zIndex,
});

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
                    <AboutView {...viewProps} />
                </NavRoute>
                <NavRoute path={`${LINKS.Api}/add`} mustBeLoggedIn={true} {...props}>
                    <ApiUpsert {...viewProps} isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Api}/edit/:rootId/:versionId`} mustBeLoggedIn={true} {...props}>
                    <ApiUpsert {...viewProps} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Api}/:rootId/:versionId?`} {...props}>
                    <ApiView {...viewProps} />
                </NavRoute>
                <NavRoute path={LINKS.Awards} {...props}>
                    <AwardsView {...viewProps} />
                </NavRoute>
                <NavRoute path={`${LINKS.BookmarkList}/add`} mustBeLoggedIn={true} {...props}>
                    <BookmarkListUpsert {...viewProps} isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.BookmarkList}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <BookmarkListUpsert {...viewProps} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.BookmarkList}/:id`} {...props}>
                    <BookmarkListView {...viewProps} />
                </NavRoute>
                <NavRoute path={LINKS.Calendar} excludePageContainer {...props}>
                    <CalendarView {...viewProps} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Create}
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    mustBeLoggedIn={true}
                    {...props}
                >
                    <CreateView {...viewProps} />
                </NavRoute>
                <NavRoute path={`${LINKS.Chat}/:id`} {...props}>
                    <ChatView {...viewProps} />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.ForgotPassword}/:code?`}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...props}
                >
                    <FormView title="Forgot Password" maxWidth="700px" zIndex={zIndex} {...props}>
                        <ForgotPasswordForm zIndex={zIndex} />
                    </FormView>
                </NavRoute>
                <NavRoute path={LINKS.History} mustBeLoggedIn={true} {...props}>
                    <HistoryView {...viewProps} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Home}
                    sitemapIndex
                    priority={1.0}
                    changeFreq="weekly"
                    excludePageContainer
                    {...props}
                >
                    <HomeView {...viewProps} />
                </NavRoute>
                <NavRoute path={LINKS.MyStuff} mustBeLoggedIn={true} {...props}>
                    <MyStuffView {...viewProps} />
                </NavRoute>
                <NavRoute path={`${LINKS.Note}/add`} excludePageContainer mustBeLoggedIn={true} {...props}>
                    <NoteUpsert {...viewProps} isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Note}/edit/:rootId/:versionId`} excludePageContainer mustBeLoggedIn={true} {...props}>
                    <NoteUpsert {...viewProps} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Note}/:rootId/:versionId?`} excludePageContainer {...props}>
                    <NoteView {...viewProps} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Inbox}
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    mustBeLoggedIn={true}
                    {...props}
                >
                    <InboxView {...viewProps} />
                </NavRoute>
                <NavRoute path={`${LINKS.Organization}/add`} sx={noSidePadding} mustBeLoggedIn={true} {...props}>
                    <OrganizationUpsert {...viewProps} isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Organization}/edit/:id`} sx={noSidePadding} mustBeLoggedIn={true} {...props}>
                    <OrganizationUpsert {...viewProps} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Organization}/:id`} sx={noSidePadding} {...props}>
                    <OrganizationView {...viewProps} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Premium}
                    sitemapIndex
                    priority={0.5}
                    changeFreq="weekly"
                    {...props}
                >
                    <PremiumView {...viewProps} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Privacy}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    sx={{ background: palette.background.paper }}
                    {...props}
                >
                    <PrivacyPolicyView {...viewProps} />
                </NavRoute>
                <NavRoute path={`${LINKS.Profile}/add`} mustBeLoggedIn={true} {...props}>
                    <BotUpsert {...viewProps} isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Profile}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <BotUpsert {...viewProps} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Profile}/:id?`} sx={noSidePadding} {...props}>
                    <UserView {...viewProps} />
                </NavRoute>
                <NavRoute path={`${LINKS.Project}/add`} mustBeLoggedIn={true} {...props}>
                    <ProjectUpsert {...viewProps} isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Project}/edit/:rootId/:versionId`} mustBeLoggedIn={true} {...props}>
                    <ProjectUpsert {...viewProps} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Project}/:rootId/:versionId?`} {...props}>
                    <ProjectView {...viewProps} />
                </NavRoute>
                <NavRoute path={`${LINKS.Question}/add`} mustBeLoggedIn={true} {...props}>
                    <QuestionUpsert {...viewProps} isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Question}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <QuestionUpsert {...viewProps} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Question}/:id`} {...props}>
                    <QuestionView {...viewProps} />
                </NavRoute>
                <NavRoute path={`${LINKS.Reminder}/add`} mustBeLoggedIn={true} {...props}>
                    <ReminderUpsert {...viewProps} isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Reminder}/edit/:id`} mustBeLoggedIn={true} {...props}>
                    <ReminderUpsert {...viewProps} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Reminder}/:id`} {...props}>
                    <ReminderView {...viewProps} />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.ResetPassword}/:params*`}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...props}
                >
                    <FormView title="Reset Password" maxWidth="700px" zIndex={zIndex} {...props}>
                        <ResetPasswordForm zIndex={zIndex} />
                    </FormView>
                </NavRoute>
                <NavRoute path={`${LINKS.Routine}/add`} mustBeLoggedIn={true} {...props}>
                    <RoutineUpsert {...viewProps} isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Routine}/edit/:rootId/:versionId`} mustBeLoggedIn={true} {...props}>
                    <RoutineUpsert {...viewProps} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Routine}/:rootId/:versionId?`} {...props}>
                    <RoutineView {...viewProps} />
                </NavRoute>
                <NavRoute
                    path={`${LINKS.Search}/:params*`}
                    sitemapIndex
                    priority={0.4}
                    changeFreq="monthly"
                    {...props}
                >
                    <SearchView {...viewProps} />
                </NavRoute>
                <NavRoute path={LINKS.Settings} mustBeLoggedIn={true} {...props}>
                    <SettingsView {...viewProps} />
                </NavRoute>
                <NavRoute path={LINKS.SettingsAuthentication} mustBeLoggedIn={true} {...props}>
                    <SettingsAuthenticationView {...viewProps} />
                </NavRoute>
                <NavRoute path={LINKS.SettingsDisplay} mustBeLoggedIn={true} {...props}>
                    <SettingsDisplayView {...viewProps} />
                </NavRoute>
                <NavRoute path={LINKS.SettingsNotifications} mustBeLoggedIn={true} {...props}>
                    <SettingsNotificationsView {...viewProps} />
                </NavRoute>
                <NavRoute path={LINKS.SettingsProfile} mustBeLoggedIn={true} {...props}>
                    <SettingsProfileView {...viewProps} />
                </NavRoute>
                <NavRoute path={LINKS.SettingsPrivacy} mustBeLoggedIn={true} {...props}>
                    <SettingsPrivacyView {...viewProps} />
                </NavRoute>
                <NavRoute path={LINKS.SettingsFocusModes} mustBeLoggedIn={true} {...props}>
                    <SettingsFocusModesView {...viewProps} />
                </NavRoute>
                <NavRoute path={`${LINKS.SmartContract}/add`} mustBeLoggedIn={true} {...props}>
                    <SmartContractUpsert {...viewProps} isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.SmartContract}/edit/:rootId/:versionId`} mustBeLoggedIn={true} {...props}>
                    <SmartContractUpsert {...viewProps} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.SmartContract}/:rootId/:versionId?`} {...props}>
                    <SmartContractView {...viewProps} />
                </NavRoute>
                <NavRoute path={`${LINKS.Standard}/add`} mustBeLoggedIn={true} {...props}>
                    <StandardUpsert {...viewProps} isCreate={true} />
                </NavRoute>
                <NavRoute path={`${LINKS.Standard}/edit/:rootId/:versionId`} mustBeLoggedIn={true} {...props}>
                    <StandardUpsert {...viewProps} isCreate={false} />
                </NavRoute>
                <NavRoute path={`${LINKS.Standard}/:rootId/:versionId?`} {...props}>
                    <StandardView {...viewProps} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Start}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    {...props}
                >
                    <StartView {...viewProps} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Stats}
                    sitemapIndex
                    priority={0.5}
                    changeFreq="weekly"
                    {...props}
                >
                    <StatsView {...viewProps} />
                </NavRoute>
                <NavRoute
                    path={LINKS.Terms}
                    sitemapIndex
                    priority={0.2}
                    changeFreq="yearly"
                    sx={{ background: palette.background.paper }}
                    {...props}
                >
                    <TermsView {...viewProps} />
                </NavRoute>
                <NavRoute {...props}>
                    <NotFoundView />
                </NavRoute>
            </Switch>
        </>
    );
};
