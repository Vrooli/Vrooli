import { type QueryAction } from "./types.js";

/** If field is a relation, finds action type */
export const getActionFromFieldName = (fieldName: string): QueryAction | null => {
    const actions: QueryAction[] = ["Connect", "Create", "Delete", "Disconnect", "Update"];
    for (const action of actions) {
        if (fieldName.endsWith(action)) {
            return action;
        }
    }
    return null;
};
