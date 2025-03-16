import { getActionFromFieldName } from "./getActionFromFieldName.js";
import { AuthDataById } from "./getAuthenticatedData.js";
import { InputsById } from "./types.js";

/**
 * Combines input data with existing data to estimate what the updated data will look like. 
 * This can be used with validator functions (e.g. isPublic) to determine if an object *will* be public 
 * once it's created or updated.
 * 
 * NOTE: When creating, this does not guarantee that the validator function will work as expected. If 
 * you do not explicitly provide a field used by the validator (e.g. don't provide `isPrivate`, and let the shape function's 
 * default handle it), then unexpected results may occur. This means it's safer to provide as much 
 * data as possible when creating an object.
 */
export function authDataWithInput(
    input: string | object,
    existingData: object,
    inputsById: InputsById,
    authDataById: AuthDataById,
): object {
    // If input is a string, it's an ID, so get the input object
    if (typeof input === "string") {
        input = inputsById[input]?.input ?? {};
    }
    // Overwrite existingData with input
    const combined = { ...existingData, ...input };
    // Loop through fields to handle relationships
    for (const field in combined) {
        // If a relationship, find the action and process accordingly
        const action = getActionFromFieldName(field);
        if (!action) continue;
        const inputValue = combined[field];
        const fieldName = field.substring(0, field.length - action.length);
        const existingValue = combined[fieldName];
        delete combined[field];

        switch (action) {
            case "Connect": {
                if (Array.isArray(inputValue)) {
                    const connectedDataArray = inputValue.map(id => authDataById[id] ?? {});
                    combined[fieldName] = connectedDataArray;
                } else {
                    combined[fieldName] = authDataById[inputValue] ?? {};
                }
                break;
            }
            case "Create": {
                const isArrayOfInputs = Array.isArray(inputValue);
                if (isArrayOfInputs) {
                    combined[fieldName] = inputValue.map((item, index) => {
                        return authDataWithInput(item, existingValue?.[index] || {}, inputsById, authDataById);
                    });
                } else {
                    combined[fieldName] = authDataWithInput(inputValue, existingValue || {}, inputsById, authDataById);
                }
                break;
            }
            case "Delete":
            case "Disconnect": {
                if (Array.isArray(inputValue)) {
                    combined[fieldName] = (existingValue || []).filter(item => !inputValue.includes(item.id));
                } else {
                    combined[fieldName] = null;
                }
                break;
            }
            case "Update": {
                if (Array.isArray(inputValue)) {
                    combined[fieldName] = (existingValue || []).map(item => {
                        // Find the item in inputValue by ID for updating
                        const updatingItem = inputValue.find(update => update.id === item.id);
                        if (updatingItem) {
                            return authDataWithInput(updatingItem, item, inputsById, authDataById);
                        } else {
                            return item;
                        }
                    });
                } else {
                    combined[fieldName] = authDataWithInput(inputValue, existingValue || {}, inputsById, authDataById);
                }
                break;
            }
        }
    }
    return combined;
}
