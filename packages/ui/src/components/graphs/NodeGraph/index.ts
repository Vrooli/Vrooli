import { NodeType } from "@local/shared";

export * from "./AddAfterLinkDialog/AddAfterLinkDialog";
export * from "./AddBeforeLinkDialog/AddBeforeLinkDialog";
export * from "./edges";
export * from "./GraphActions/GraphActions";
export * from "./NodeColumn/NodeColumn";
export * from "./NodeContextMenu/NodeContextMenu";
export * from "./NodeEndDialog/NodeEndDialog";
export * from "./NodeGraph/NodeGraph";
export * from "./NodeRoutineListDialog/NodeRoutineListDialog";
export * from "./nodes";

export const NodeWidth = {
    [NodeType.End]: 100,
    [NodeType.Redirect]: 100,
    [NodeType.RoutineList]: 350,
    [NodeType.Start]: 100,
};
