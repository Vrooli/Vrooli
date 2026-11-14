import type { RouteObject } from "react-router-dom";
import { SettingsIndex, SettingsPage } from "../../pages/SettingsPage";
import { ReachPage } from "../../pages/ReachPage";
import { ChannelPage } from "../reach/ChannelPage";
import { DevicePage } from "../reach/DevicePage";
import { AppearancePage } from "./AppearancePage";
import { EscalationPage } from "./EscalationPage";
import { QuietHoursPage } from "./QuietHoursPage";
import { SessionPage } from "./SessionPage";
import { SourcesPage } from "./SourcesPage";

/**
 * Every settings screen is its own route, so the back button, a reload and a
 * shared link all land where the reader was. The route table mounts this under
 * `/settings`.
 */
export const settingsRoute: RouteObject = {
  path: "settings",
  element: <SettingsPage />,
  children: [
    { index: true, element: <SettingsIndex /> },
    { path: "appearance", element: <AppearancePage /> },
    { path: "session", element: <SessionPage /> },
    { path: "quiet-hours", element: <QuietHoursPage /> },
    { path: "escalation", element: <EscalationPage /> },
    { path: "sources", element: <SourcesPage /> },
    { path: "reach", element: <ReachPage /> },
    { path: "reach/devices/:deviceId", element: <DevicePage /> },
    { path: "reach/devices/:deviceId/channels/:channel", element: <ChannelPage /> },
  ],
};
