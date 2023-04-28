import { Button, keyframes, styled } from "@mui/material";

const pulse = keyframes`
    0% {
        box-shadow: 0 0 0 0 rgba(0, 255, 170, 0.7);
    }
    70% {
        box-shadow: 0 0 0 10px rgba(0, 255, 170, 0);
    }
    100% {
        box-shadow: 0 0 0 0 rgba(0, 255, 170, 0);
    }
`;

export const PulseButton = styled(Button)(({ theme }) => ({
    fontSize: "1.8rem",
    // Button border has neon green glow animation
    animation: `${pulse} 3s infinite ease`,
    borderColor: "#0fa",
    borderWidth: "2px",
    color: "#0fa",
    fontWeight: 550,
    width: "fit-content",
    // On hover, brighten and grow
    "&:hover": {
        borderColor: "#0fa",
        color: "#0fa",
        background: "transparent",
        filter: "brightness(1.2)",
        transform: "scale(1.05)",
    },
    transition: "all 0.2s ease",
}));
