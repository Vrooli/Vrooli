/**
 * Yup augmentations that add custom methods to yup schemas.
 * This file must be imported before any validation schemas that use these methods.
 */
import * as yup from "yup";

// Add removeEmptyString method to yup.string
yup.addMethod(yup.string, "removeEmptyString", function transformRemoveEmptyString() {
    return this.transform((value: unknown) => {
        if (typeof value === "string") {
            return value.trim() !== "" ? value : undefined;
        }
        // Convert non-strings to their string representation, then check if empty
        if (value !== null && value !== undefined) {
            const stringValue = String(value);
            // Special case: arrays convert to "" which should be undefined
            if (Array.isArray(value) && stringValue === "") {
                return undefined;
            }
            return stringValue.trim() !== "" ? stringValue : undefined;
        }
        return undefined;
    });
});

// Add toBool method to yup.bool  
yup.addMethod(yup.bool, "toBool", function transformToBool() {
    return this.transform((value: unknown) => {
        if (typeof value === "boolean") return value;
        if (typeof value === "string") {
            const trimmed = value.trim().toLowerCase();
            return trimmed === "true" || trimmed === "yes" || trimmed === "1";
        }
        if (typeof value === "number") return value === 1;
        
        // Handle objects with valueOf or toString methods
        if (value && typeof value === "object") {
            // Try valueOf first
            if (value && typeof value === "object" && "valueOf" in value && typeof value.valueOf === "function") {
                const valueOfResult = value.valueOf();
                if (typeof valueOfResult === "number") {
                    return valueOfResult === 1;
                }
            }
            
            // Try toString
            if (value && typeof value === "object" && "toString" in value && typeof value.toString === "function") {
                const stringResult = value.toString();
                if (typeof stringResult === "string") {
                    const trimmed = stringResult.trim().toLowerCase();
                    return trimmed === "true" || trimmed === "yes" || trimmed === "1";
                }
            }
        }
        
        return false;
    });
});

// Augment TypeScript declarations for the new methods
declare module "yup" {
    interface StringSchema {
        removeEmptyString(): StringSchema;
    }
    
    interface BooleanSchema {
        toBool(): BooleanSchema;
    }
}
