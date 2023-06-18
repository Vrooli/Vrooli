export const smallButtonProps = (isEditing: boolean, canPressWhenNotEditing: boolean) => ({
    width: { xs: "58px", md: "69px" },
    height: { xs: "58px", md: "69px" },
    overflow: "hidden",
    boxShadow: !isEditing && !canPressWhenNotEditing ? 0 : 4,
    pointerEvents: !isEditing && !canPressWhenNotEditing ? "none" : "auto",
}) as const;

export const largeButtonProps = (isEditing: boolean, canPressWhenNotEditing: boolean) => ({
    minWidth: { xs: "58px", md: "69px" },
    height: { xs: "58px", md: "69px" },
    overflow: "hidden",
    boxShadow: !isEditing && !canPressWhenNotEditing ? 0 : 4,
    pointerEvents: !isEditing && !canPressWhenNotEditing ? "none" : "auto",
}) as const;

export const commonIconProps = () => ({
    width: "69px",
    height: "69px",
    fill: "white",
}) as const;

export const commonLabelProps = () => ({
    width: { xs: "58px", md: "69px" },
    textAlign: "center",
}) as const;
