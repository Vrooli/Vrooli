import { Box } from "@mui/material";
import { DiagonalWaveLoader } from "../DiagonalWaveLoader/DiagonalWaveLoader";

export const FullPageSpinner = () => {
    return (
        <Box sx={{
            position: "absolute",
            top: "50%",
            left: "50%",
            transform: "translate(-50%, -50%)",
            zIndex: 100000,
        }}>
            <DiagonalWaveLoader size={100} />
        </Box>
    );
};
