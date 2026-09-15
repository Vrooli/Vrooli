import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { getProxyInfo } from "@vrooli/api-base";
import { BoardController } from "./components/BoardController";
import FocusPage from "./pages/FocusPage";
import OpenLoopPage from "./pages/OpenLoopPage";
import RoomPage from "./pages/RoomPage";

/** Routes are derived: every room the board reports is reachable at /:roomId. */
export default function App() {
  const proxyInfo = getProxyInfo();
  const basename = (proxyInfo?.primary.path ?? "").replace(/\/+$/, "");
  return (
    <BrowserRouter basename={basename}>
      <BoardController>
        <Routes>
          <Route path="/" element={<Navigate to="/mission-control" replace />} />
          <Route path="/focus" element={<FocusPage />} />
          <Route path="/open-loop" element={<OpenLoopPage />} />
          <Route path="/:roomId" element={<RoomPage />} />
          <Route path="*" element={<Navigate to="/mission-control" replace />} />
        </Routes>
      </BoardController>
    </BrowserRouter>
  );
}
