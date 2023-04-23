import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { LINKS } from "@local/consts";
import { Box } from "@mui/material";
import { lazily } from "react-lazily";
import { FullPageSpinner } from "./components/FullPageSpinner/FullPageSpinner";
import { Page } from "./components/Page/Page";
import { ScrollToTop } from "./components/ScrollToTop";
import { ForgotPasswordForm, ResetPasswordForm } from "./forms/auth";
import { Route, Switch } from "./utils/route";
const { HomeView, HistoryView, CreateView, MyStuffView, NotificationsView, } = lazily(() => import("./views/main"));
const { SettingsView, SettingsAuthenticationView, SettingsDisplayView, SettingsNotificationsView, SettingsProfileView, SettingsPrivacyView, SettingsFocusModesView, } = lazily(() => import("./views/settings"));
const { PrivacyPolicyView, TermsView, } = lazily(() => import("./views/legal"));
const { TutorialView } = lazily(() => import("./views/tutorial"));
const { WelcomeView } = lazily(() => import("./views/WelcomeView/WelcomeView"));
const { AboutView } = lazily(() => import("./views/AboutView/AboutView"));
const { AwardsView } = lazily(() => import("./views/AwardsView/AwardsView"));
const { CalendarView } = lazily(() => import("./views/CalendarView/CalendarView"));
const { FormView } = lazily(() => import("./views/wrapper/FormView"));
const { NotFoundView } = lazily(() => import("./views/NotFoundView/NotFoundView"));
const { PremiumView } = lazily(() => import("./views/PremiumView/PremiumView"));
const { SearchView } = lazily(() => import("./views/SearchView/SearchView"));
const { StartView } = lazily(() => import("./views/StartView/StartView"));
const { StatsView } = lazily(() => import("./views/StatsView/StatsView"));
const { ApiUpsert, ApiView } = lazily(() => import("./views/api"));
const { BookmarkListView } = lazily(() => import("./views/bookmarkList"));
const { NoteUpsert, NoteView } = lazily(() => import("./views/note"));
const { OrganizationUpsert, OrganizationView } = lazily(() => import("./views/organization"));
const { ProjectUpsert, ProjectView } = lazily(() => import("./views/project"));
const { QuestionUpsert, QuestionView } = lazily(() => import("./views/question"));
const { ReminderUpsert, ReminderView } = lazily(() => import("./views/reminder"));
const { RoutineUpsert, RoutineView } = lazily(() => import("./views/routine"));
const { SmartContractUpsert, SmartContractView } = lazily(() => import("./views/smartContract"));
const { StandardUpsert, StandardView } = lazily(() => import("./views/standard"));
const { UserView } = lazily(() => import("./views/user"));
const Fallback = _jsxs(Box, { children: [_jsx(Box, { sx: {
                background: (t) => t.palette.primary.dark,
                height: "64px!important",
                zIndex: 100,
                position: "fixed",
                top: 0,
                left: 0,
                right: 0,
            } }), _jsx(FullPageSpinner, {})] });
const NavRoute = (props) => {
    return (_jsx(Route, { ...props, children: _jsx(Page, { ...props, children: props.children }) }));
};
const noSidePadding = {
    paddingLeft: 0,
    paddingRight: 0,
};
export const Routes = (props) => {
    return (_jsxs(_Fragment, { children: [_jsx(ScrollToTop, {}), _jsxs(Switch, { fallback: Fallback, children: [_jsx(NavRoute, { path: LINKS.About, sitemapIndex: true, priority: 0.5, changeFreq: "monthly", ...props, children: _jsx(AboutView, {}) }), _jsx(NavRoute, { path: `${LINKS.Api}/add`, mustBeLoggedIn: true, ...props, children: _jsx(ApiUpsert, { display: 'page', isCreate: true }) }), _jsx(NavRoute, { path: `${LINKS.Api}/edit/:id`, mustBeLoggedIn: true, ...props, children: _jsx(ApiUpsert, { display: 'page', isCreate: false }) }), _jsx(NavRoute, { path: `${LINKS.Api}/:id`, ...props, children: _jsx(ApiView, {}) }), _jsx(NavRoute, { path: LINKS.Awards, ...props, children: _jsx(AwardsView, {}) }), _jsx(NavRoute, { path: `${LINKS.BookmarkList}/:id`, ...props, children: _jsx(BookmarkListView, {}) }), _jsx(NavRoute, { path: LINKS.Calendar, excludePageContainer: true, ...props, children: _jsx(CalendarView, {}) }), _jsx(NavRoute, { path: LINKS.Create, sitemapIndex: true, priority: 0.4, changeFreq: "monthly", mustBeLoggedIn: true, ...props, children: _jsx(CreateView, {}) }), _jsx(NavRoute, { path: `${LINKS.ForgotPassword}/:code?`, sitemapIndex: true, priority: 0.2, changeFreq: "yearly", ...props, children: _jsx(FormView, { title: "Forgot Password", maxWidth: "700px", ...props, children: _jsx(ForgotPasswordForm, { onClose: () => { } }) }) }), _jsx(NavRoute, { path: LINKS.History, mustBeLoggedIn: true, ...props, children: _jsx(HistoryView, {}) }), _jsx(NavRoute, { path: LINKS.Home, sitemapIndex: true, priority: 1.0, changeFreq: "weekly", excludePageContainer: true, ...props, children: _jsx(HomeView, {}) }), _jsx(NavRoute, { path: LINKS.MyStuff, mustBeLoggedIn: true, ...props, children: _jsx(MyStuffView, {}) }), _jsx(NavRoute, { path: `${LINKS.Note}/add`, mustBeLoggedIn: true, ...props, children: _jsx(NoteUpsert, { display: 'page', isCreate: true }) }), _jsx(NavRoute, { path: `${LINKS.Note}/edit/:id`, mustBeLoggedIn: true, ...props, children: _jsx(NoteUpsert, { display: 'page', isCreate: false }) }), _jsx(NavRoute, { path: `${LINKS.Note}/:id`, ...props, children: _jsx(NoteView, {}) }), _jsx(NavRoute, { path: LINKS.Notifications, sitemapIndex: true, priority: 0.4, changeFreq: "monthly", mustBeLoggedIn: true, ...props, children: _jsx(NotificationsView, {}) }), _jsx(NavRoute, { path: `${LINKS.Organization}/add`, sx: noSidePadding, mustBeLoggedIn: true, ...props, children: _jsx(OrganizationUpsert, { display: 'page', isCreate: true }) }), _jsx(NavRoute, { path: `${LINKS.Organization}/edit/:id`, sx: noSidePadding, mustBeLoggedIn: true, ...props, children: _jsx(OrganizationUpsert, { display: 'page', isCreate: false }) }), _jsx(NavRoute, { path: `${LINKS.Organization}/:id`, sx: noSidePadding, ...props, children: _jsx(OrganizationView, {}) }), _jsx(NavRoute, { path: LINKS.Premium, sitemapIndex: true, priority: 0.5, changeFreq: "weekly", ...props, children: _jsx(PremiumView, {}) }), _jsx(NavRoute, { path: LINKS.Privacy, sitemapIndex: true, priority: 0.2, changeFreq: "yearly", ...props, children: _jsx(PrivacyPolicyView, {}) }), _jsx(NavRoute, { path: `${LINKS.Profile}/:id?`, sx: noSidePadding, ...props, children: _jsx(UserView, {}) }), _jsx(NavRoute, { path: `${LINKS.Project}/add`, mustBeLoggedIn: true, ...props, children: _jsx(ProjectUpsert, { display: 'page', isCreate: true }) }), _jsx(NavRoute, { path: `${LINKS.Project}/edit/:id`, mustBeLoggedIn: true, ...props, children: _jsx(ProjectUpsert, { display: 'page', isCreate: false }) }), _jsx(NavRoute, { path: `${LINKS.Project}/:id`, ...props, children: _jsx(ProjectView, {}) }), _jsx(NavRoute, { path: `${LINKS.Question}/add`, mustBeLoggedIn: true, ...props, children: _jsx(QuestionUpsert, { display: 'page', isCreate: true }) }), _jsx(NavRoute, { path: `${LINKS.Question}/edit/:id`, mustBeLoggedIn: true, ...props, children: _jsx(QuestionUpsert, { display: 'page', isCreate: false }) }), _jsx(NavRoute, { path: `${LINKS.Question}/:id`, ...props, children: _jsx(QuestionView, {}) }), _jsx(NavRoute, { path: `${LINKS.Reminder}/add`, mustBeLoggedIn: true, ...props, children: _jsx(ReminderUpsert, { display: 'page', handleDelete: () => { }, isCreate: true }) }), _jsx(NavRoute, { path: `${LINKS.Reminder}/edit/:id`, mustBeLoggedIn: true, ...props, children: _jsx(ReminderUpsert, { display: 'page', handleDelete: () => { }, isCreate: false }) }), _jsx(NavRoute, { path: `${LINKS.Reminder}/:id`, ...props, children: _jsx(ReminderView, {}) }), _jsx(NavRoute, { path: `${LINKS.ResetPassword}/:params*`, sitemapIndex: true, priority: 0.2, changeFreq: "yearly", ...props, children: _jsx(FormView, { title: "Reset Password", maxWidth: "700px", ...props, children: _jsx(ResetPasswordForm, { onClose: () => { } }) }) }), _jsx(NavRoute, { path: `${LINKS.Routine}/add`, mustBeLoggedIn: true, ...props, children: _jsx(RoutineUpsert, { display: 'page', isCreate: true }) }), _jsx(NavRoute, { path: `${LINKS.Routine}/edit/:id`, mustBeLoggedIn: true, ...props, children: _jsx(RoutineUpsert, { display: 'page', isCreate: false }) }), _jsx(NavRoute, { path: `${LINKS.Routine}/:id`, ...props, children: _jsx(RoutineView, {}) }), _jsx(NavRoute, { path: `${LINKS.Search}/:params*`, sitemapIndex: true, priority: 0.4, changeFreq: "monthly", ...props, children: _jsx(SearchView, {}) }), _jsx(NavRoute, { path: LINKS.Settings, mustBeLoggedIn: true, ...props, children: _jsx(SettingsView, {}) }), _jsx(NavRoute, { path: LINKS.SettingsAuthentication, mustBeLoggedIn: true, ...props, children: _jsx(SettingsAuthenticationView, {}) }), _jsx(NavRoute, { path: LINKS.SettingsDisplay, mustBeLoggedIn: true, ...props, children: _jsx(SettingsDisplayView, {}) }), _jsx(NavRoute, { path: LINKS.SettingsNotifications, mustBeLoggedIn: true, ...props, children: _jsx(SettingsNotificationsView, {}) }), _jsx(NavRoute, { path: LINKS.SettingsProfile, mustBeLoggedIn: true, ...props, children: _jsx(SettingsProfileView, {}) }), _jsx(NavRoute, { path: LINKS.SettingsPrivacy, mustBeLoggedIn: true, ...props, children: _jsx(SettingsPrivacyView, {}) }), _jsx(NavRoute, { path: LINKS.SettingsFocusModes, mustBeLoggedIn: true, ...props, children: _jsx(SettingsFocusModesView, {}) }), _jsx(NavRoute, { path: `${LINKS.SmartContract}/add`, mustBeLoggedIn: true, ...props, children: _jsx(SmartContractUpsert, { display: 'page', isCreate: true }) }), _jsx(NavRoute, { path: `${LINKS.SmartContract}/edit/:id`, mustBeLoggedIn: true, ...props, children: _jsx(SmartContractUpsert, { display: 'page', isCreate: false }) }), _jsx(NavRoute, { path: `${LINKS.SmartContract}/:id`, ...props, children: _jsx(SmartContractView, {}) }), _jsx(NavRoute, { path: `${LINKS.Standard}/add`, mustBeLoggedIn: true, ...props, children: _jsx(StandardUpsert, { display: 'page', isCreate: true }) }), _jsx(NavRoute, { path: `${LINKS.Standard}/edit/:id`, mustBeLoggedIn: true, ...props, children: _jsx(StandardUpsert, { display: 'page', isCreate: false }) }), _jsx(NavRoute, { path: `${LINKS.Standard}/:id`, ...props, children: _jsx(StandardView, {}) }), _jsx(NavRoute, { path: LINKS.Start, sitemapIndex: true, priority: 0.2, changeFreq: "yearly", ...props, children: _jsx(StartView, {}) }), _jsx(NavRoute, { path: LINKS.Stats, sitemapIndex: true, priority: 0.5, changeFreq: "weekly", ...props, children: _jsx(StatsView, {}) }), _jsx(NavRoute, { path: LINKS.Terms, sitemapIndex: true, priority: 0.2, changeFreq: "yearly", ...props, children: _jsx(TermsView, {}) }), _jsx(NavRoute, { path: LINKS.Tutorial, sitemapIndex: true, priority: 0.5, changeFreq: "monthly", ...props, children: _jsx(TutorialView, {}) }), _jsx(NavRoute, { path: LINKS.Welcome, sitemapIndex: true, priority: 0.5, changeFreq: "monthly", ...props, children: _jsx(WelcomeView, {}) }), _jsx(NavRoute, { ...props, children: _jsx(NotFoundView, {}) })] })] }));
};
//# sourceMappingURL=Routes.js.map