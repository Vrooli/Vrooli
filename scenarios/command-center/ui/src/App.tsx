import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { getProxyInfo } from "@vrooli/api-base";
import RoomPage from "./pages/RoomPage";
import FocusPage from "./pages/FocusPage";
import OpenLoopPage from "./pages/OpenLoopPage";
import { BoardController } from "./components/BoardController";

export default function App() {
  const proxyInfo = getProxyInfo();
  const basename = (proxyInfo?.primary?.path ?? proxyInfo?.basePath ?? "").replace(/\/+$/, "");
  return (
    <BrowserRouter basename={basename}>
      <BoardController><Routes>
        <Route path="/" element={<Navigate to="/mission-control" replace />} />
        <Route path="/focus" element={<FocusPage />} />
        <Route path="/open-loop" element={<OpenLoopPage />} />
        <Route path="/:roomId" element={<RoomPage />} />
        <Route path="*" element={<Navigate to="/mission-control" replace />} />
      </Routes></BoardController>
    </BrowserRouter>
  );
}
