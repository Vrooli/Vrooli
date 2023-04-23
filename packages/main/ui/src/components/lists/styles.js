export const smallHorizontalScrollbar = (palette) => ({
    overflowX: "auto",
    "&::-webkit-scrollbar": {
        height: 5,
    },
    "&::-webkit-scrollbar-track": {
        backgroundColor: "transparent",
    },
    "&::-webkit-scrollbar-thumb": {
        borderRadius: "100px",
        backgroundColor: palette.background.textSecondary,
    },
});
export const cardRoot = {
    boxShadow: 6,
    background: (t) => t.palette.primary.light,
    color: (t) => t.palette.primary.contrastText,
    borderRadius: "16px",
    margin: 0,
    cursor: "pointer",
    maxWidth: "500px",
    "&:hover": {
        filter: "brightness(120%)",
        transition: "filter 0.2s",
    },
};
//# sourceMappingURL=styles.js.map