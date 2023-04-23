import { multiLineEllipsis, textShadow } from "../../../../styles";
export const nodeLabel = {
    ...multiLineEllipsis(3),
    ...textShadow,
    position: "absolute",
    textAlign: "center",
    margin: "0",
    top: "50%",
    left: "50%",
    transform: "translate(-50%, -50%)",
    width: "100%",
    lineBreak: "anywhere",
};
export const routineNodeCheckboxLabel = {
    marginLeft: "0",
};
export const routineNodeCheckboxOption = {
    padding: "4px",
};
//# sourceMappingURL=styles.js.map