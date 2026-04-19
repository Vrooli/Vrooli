import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import MissionControl from "./pages/MissionControl";
import Hive from "./pages/Hive";
import Forge from "./pages/Forge";
import Ledger from "./pages/Ledger";
import Broadcast from "./pages/Broadcast";
import Panorama from "./pages/Panorama";

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Navigate to="/mission-control" replace />} />
        <Route path="/mission-control" element={<MissionControl />} />
        <Route path="/hive" element={<Hive />} />
        <Route path="/forge" element={<Forge />} />
        <Route path="/ledger" element={<Ledger />} />
        <Route path="/broadcast" element={<Broadcast />} />
        <Route path="/panorama" element={<Panorama />} />
        <Route path="*" element={<Navigate to="/mission-control" replace />} />
      </Routes>
    </BrowserRouter>
  );
}
