export const smallButtonProps = (isEditing: boolean, canPressWhenNotEditing: boolean) => ({
    width: { xs: "48px", md: "56px" },
    height: { xs: "48px", md: "56px" },
    overflow: "hidden",
    boxShadow: !isEditing && !canPressWhenNotEditing ? 0 : 4,
    pointerEvents: !isEditing && !canPressWhenNotEditing ? "none" : "auto",
}) as const;

export const largeButtonProps = (isEditing: boolean, canPressWhenNotEditing: boolean) => ({
    minWidth: { xs: "48px", md: "56px" },
    height: { xs: "48px", md: "56px" },
    overflow: "hidden",
    boxShadow: !isEditing && !canPressWhenNotEditing ? 0 : 4,
    pointerEvents: !isEditing && !canPressWhenNotEditing ? "none" : "auto",
}) as const;

export const commonIconProps = () => ({
    width: "56px",
    height: "56px",
    fill: "white",
}) as const;

export const commonLabelProps = () => ({
    width: { xs: "48px", md: "56px" },
    textAlign: "center",
}) as const;
