import { endpointsResource } from "../../api/pairs.js";
import { shapeResourceVersion } from "../../shape/models/models.js";
import { 
    resourceVersionTranslationValidation,
    resourceVersionValidation,
} from "../../validation/models/resourceVersion.js";
import type { 
    Resource,
    ResourceVersionCreateInput,
    ResourceVersion,
    ResourceVersionUpdateInput,
} from "../../api/types.js";

/**
 * Base form configuration for ResourceVersion-based forms
 * Used by DataConverter, DataStructure, Prompt, SmartContract, etc.
 */
export const resourceVersionFormConfig = {
    objectType: "ResourceVersion",
    validation: resourceVersionValidation,
    translationValidation: resourceVersionTranslationValidation,
    transformFunction: (values: ResourceVersion, existing: ResourceVersion, isCreate: boolean) => {
        return isCreate 
            ? shapeResourceVersion.create(values) 
            : shapeResourceVersion.update(existing, values);
    },
    endpoints: endpointsResource,
};

/**
 * Specialized form config for DataConverter
 */
export const dataConverterFormConfig = {
    ...resourceVersionFormConfig,
    objectType: "DataConverter" as const,
};

/**
 * Specialized form config for DataStructure
 */
export const dataStructureFormConfig = {
    ...resourceVersionFormConfig,
    objectType: "DataStructure" as const,
};

/**
 * Specialized form config for Prompt
 */
export const promptFormConfig = {
    ...resourceVersionFormConfig,
    objectType: "Prompt" as const,
};

/**
 * Specialized form config for SmartContract
 */
export const smartContractFormConfig = {
    ...resourceVersionFormConfig,
    objectType: "SmartContract" as const,
};
