import type { RouteObject } from "react-router-dom";

import { AppShell } from "../layout/AppShell";
import { RouteError } from "./RouteError";
import { DashboardPage } from "../pages/DashboardPage";
import { PacketPage } from "../features/packet/PacketPage";
import { AsksPage } from "../features/asks/AsksPage";
import { HistoryPage } from "../features/asks/HistoryPage";
import { DeliveryPage } from "../pages/DeliveryPage";
import { DeliveryDetailPage } from "../pages/DeliveryDetailPage";
import { NotFoundPage } from "../pages/NotFoundPage";
import { settingsRoute } from "../features/settings/routes";

/**
 * Canonical route table. Exported so tests can construct an in-memory router
 * from the same config the production app uses.
 *
 * Add new pages by appending to the `children` array.
 */
export const routes: RouteObject[] = [
  {
    path: "/",
    element: <AppShell />,
    errorElement: <RouteError />,
    children: [
      // Start on the useful read-only feed; the decision queue is explicitly
      // available at /asks and requires an owner session.
      { index: true, element: <DashboardPage /> },
      { path: "updates", element: <DashboardPage /> },
      { path: "asks", element: <AsksPage /> },
      { path: "asks/:askId", element: <PacketPage /> },
      { path: "history", element: <HistoryPage /> },
      { path: "delivery", element: <DeliveryPage /> },
      { path: "delivery/:notificationId", element: <DeliveryDetailPage /> },
      settingsRoute,
      { path: "*", element: <NotFoundPage /> },
    ],
  },
];
