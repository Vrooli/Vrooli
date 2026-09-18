import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { getProxyInfo } from "@vrooli/api-base";
import { BoardController } from "./components/BoardController";
import FocusPage from "./pages/FocusPage";
import OpenLoopPage from "./pages/OpenLoopPage";
import RoomPage from "./pages/RoomPage";
import SettingsPage from "./pages/SettingsPage";
import { fetchCatalogs } from "./lib/api";
import { configureEnabledPacks } from "./scenes";

function CatalogThemeLoader() {
  const { data } = useQuery({ queryKey: ["catalogs"], queryFn: fetchCatalogs, staleTime: 30_000 });
  useEffect(() => {
    if (!data?.themes) return;
    const connectorEntries = data.connectors ?? [];
    const enabledPacks = new Set(connectorEntries.filter((connector) => connector.enabled !== false && connector.pack === true).map((connector) => connector.id));
    configureEnabledPacks(connectorEntries.some((connector) => connector.pack === true) ? enabledPacks : new Set(["vrooli"]));
    const styleID = "command-center-catalog-themes";
    document.getElementById(styleID)?.remove();
    const style = document.createElement("style"); style.id = styleID;
    style.textContent = data.themes.map((theme) => {
      const tokens = theme.tokens && typeof theme.tokens === "object" ? theme.tokens as Record<string, unknown> : {};
      return `[data-theme="${theme.id}"]{${Object.entries(tokens).filter(([, value]) => typeof value === "string").map(([key, value]) => `${key}:${value}`).join(";")}}`;
    }).join("\n");
    document.head.appendChild(style);
    return () => document.getElementById(styleID)?.remove();
  }, [data]);
  return null;
}

/** Routes are derived: every room the board reports is reachable at /:roomId. */
export default function App() {
  const proxyInfo = getProxyInfo();
  const basename = (proxyInfo?.primary.path ?? "").replace(/\/+$/, "");
  return (
    <BrowserRouter basename={basename}>
      <BoardController>
        <CatalogThemeLoader />
        <Routes>
          <Route path="/" element={<Navigate to="/mission-control" replace />} />
          <Route path="/focus" element={<FocusPage />} />
          <Route path="/open-loop" element={<OpenLoopPage />} />
          <Route path="/settings" element={<SettingsPage />} />
          <Route path="/:roomId" element={<RoomPage />} />
          <Route path="*" element={<Navigate to="/mission-control" replace />} />
        </Routes>
      </BoardController>
    </BrowserRouter>
  );
}
