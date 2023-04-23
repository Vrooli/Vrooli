import { jsx as _jsx } from "react/jsx-runtime";
import { Box } from "@mui/material";
import { DiagonalWaveLoader } from "../DiagonalWaveLoader/DiagonalWaveLoader";
export const FullPageSpinner = () => {
    return (_jsx(Box, { sx: {
            position: "absolute",
            top: "50%",
            left: "50%",
            transform: "translate(-50%, -50%)",
            zIndex: 100000,
        }, children: _jsx(DiagonalWaveLoader, { size: 100 }) }));
};
//# sourceMappingURL=FullPageSpinner.js.map