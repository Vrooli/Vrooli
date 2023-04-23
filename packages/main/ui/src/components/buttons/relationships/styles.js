export const commonButtonProps = (isEditing, canPressWhenNotEditing) => ({
    width: { xs: "58px", md: "69px" },
    height: { xs: "58px", md: "69px" },
    overflow: "hidden",
    boxShadow: !isEditing && !canPressWhenNotEditing ? 0 : 4,
    pointerEvents: !isEditing && !canPressWhenNotEditing ? "none" : "auto",
});
export const commonIconProps = () => ({
    width: "69px",
    height: "69px",
    color: "white",
});
export const commonLabelProps = () => ({
    width: { xs: "58px", md: "69px" },
    textAlign: "center",
});
//# sourceMappingURL=styles.js.map