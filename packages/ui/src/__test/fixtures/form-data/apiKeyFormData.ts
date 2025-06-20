// Note: Using simple types since ApiKeyCreateInput/UpdateInput are not re-exported from shared
// These represent the form data structure before API submission

/**
 * Form data fixtures for API key-related forms
 * These represent data as it appears in form state before submission
 */

/**
 * API key creation form data
 */
export const minimalApiKeyCreateFormInput = {
    name: "Development API Key",
    limitHard: "1000000",
    stopAtLimit: true,
    permissions: "read,write",
};

export const completeApiKeyCreateFormInput = {
    name: "Production API Key",
    disabled: false,
    limitHard: "5000000",
    limitSoft: "4000000",
    stopAtLimit: false,
    permissions: "read,write,delete",
    teamConnect: "123456789012345678", // Optional team connection
};

/**
 * API key update form data
 */
export const minimalApiKeyUpdateFormInput = {
    id: "123456789012345678",
    name: "Updated API Key Name",
};

export const completeApiKeyUpdateFormInput = {
    id: "123456789012345678",
    name: "Production API Key - Updated",
    disabled: false,
    limitHard: "10000000",
    limitSoft: "8000000",
    stopAtLimit: true,
    permissions: "read,write,delete,admin",
};

/**
 * Form validation scenarios
 */
export const invalidApiKeyCreateFormInputs = {
    missingName: {
        limitHard: "1000000",
        stopAtLimit: true,
        permissions: "read",
    },
    missingLimitHard: {
        name: "Test Key",
        stopAtLimit: true,
        permissions: "read",
    },
    missingPermissions: {
        name: "Test Key",
        limitHard: "1000000",
        stopAtLimit: true,
    },
};

/**
 * Helper functions
 */
export const transformApiKeyFormToApiInput = (formData: any) => {
    return {
        ...formData,
        // Ensure BigInt fields are strings
        limitHard: formData.limitHard?.toString(),
        limitSoft: formData.limitSoft?.toString(),
    };
};