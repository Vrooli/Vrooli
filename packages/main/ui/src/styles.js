export const centeredDiv = {
    display: "flex",
    justifyContent: "center",
    alignItems: "center",
};
export const textShadow = {
    textShadow: `-0.5px -0.5px 0 black,  
        0.5px -0.5px 0 black,
        -0.5px 0.5px 0 black,
        0.5px 0.5px 0 black`,
};
export const clickSize = {
    minHeight: "48px",
};
export const multiLineEllipsis = (lines) => ({
    display: "-webkit-box",
    WebkitLineClamp: lines,
    WebkitBoxOrient: "vertical",
    overflow: "hidden",
    textOverflow: "ellipsis",
});
export const noSelect = {
    WebkitTouchCallout: "none",
    WebkitUserSelect: "none",
    MozUserSelect: "none",
    msUserSelect: "none",
    userSelect: "none",
};
export const linkColors = (palette) => ({
    a: {
        color: palette.mode === "light" ? "#001cd3" : "#dd86db",
        "&:visited": {
            color: palette.mode === "light" ? "#001cd3" : "#f551ef",
        },
        "&:active": {
            color: palette.mode === "light" ? "#001cd3" : "#f551ef",
        },
        "&:hover": {
            color: palette.mode === "light" ? "#5a6ff6" : "#f3d4f2",
        },
        textDecoration: "none",
    },
});
export const greenNeonText = {
    color: "#fff",
    filter: "drop-shadow(0 0 2px #fff) drop-shadow(0 0 4px #0fa) drop-shadow(0 0 4px #0fa) drop-shadow(0 0 32px #0fa) drop-shadow(0 0 21px #0fa)",
};
export const iconButtonProps = {
    background: "transparent",
    border: "1px solid #0fa",
    "&:hover": {
        background: "transparent",
        filter: "brightness(1.2)",
        transform: "scale(1.2)",
    },
    transition: "all 0.2s ease",
};
export const slideText = {
    margin: "auto",
    textAlign: { xs: "left", md: "justify" }, fontSize: { xs: "1.25rem", md: "1.5rem" },
    zIndex: 10,
};
export const slideTitle = {
    textAlign: "center",
    fontSize: { xs: "2.4em", sm: "3rem", md: "3.75rem" },
    zIndex: 10,
};
export const slideImageContainer = {
    justifyContent: "center",
    height: "100%",
    display: "flex",
    "& > img": {
        maxWidth: {
            xs: "min(225px, 90%)",
            sm: "min(300px, 90%)",
        },
        maxHeight: "100%",
        objectFit: "contain",
        zIndex: "3",
    },
};
export const textPop = {
    padding: "0",
    color: "white",
    textAlign: "center",
    fontWeight: 600,
    textShadow: `-1px -1px 0 black,  
                1px -1px 0 black,
                -1px 1px 0 black,
                1px 1px 0 black`,
};
//# sourceMappingURL=styles.js.map