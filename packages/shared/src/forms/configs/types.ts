/**
 * Unified Form Configuration Types
 * 
 * This module defines a unified structure for form configurations that can be
 * used across UI components and API endpoints. Test fixtures are kept separate
 * and reference these configurations.
 */

import type { Session } from "../../api/types.js";
import type { YupModel } from "../../validation/utils/types.js";

/** Endpoint definition from pairs.ts */
export interface EndpointDefinition {
    endpoint: string;
    method: string;
}

/**
 * Core form configuration for production use
 * 
 * @template TShape - The shape type (internal form data structure)
 * @template TCreateInput - API input type for create operations
 * @template TUpdateInput - API input type for update operations
 * @template TApiResult - API response type
 */
export interface FormConfig<
    TShape extends { __typename: string; id: string },
    TCreateInput extends Record<string, any>,
    TUpdateInput extends Record<string, any>,
    TApiResult extends { __typename: string; id: string }
> {
    /** Object type being configured */
    objectType: string;

    /** Validation schemas */
    validation: {
        schema: YupModel<["create", "update"]>;
        translationSchema?: YupModel<["create", "update"]>;
    };

    /** Data transformation functions */
    transformations: {
        /** Convert raw form data to internal shape format */
        formToShape: (formData: any) => TShape;

        /** Shape transformation object with create/update methods */
        shapeToInput: {
            create?: (shape: TShape) => TCreateInput;
            update?: (existing: TShape, shape: TShape) => TUpdateInput | undefined;
        };

        /** Convert API response back to shape (for round-trip testing) */
        apiResultToShape: (apiResult: TApiResult) => TShape;

        /** Generate initial form values */
        getInitialValues: (session?: Session, existing?: Partial<TShape>) => TShape;
    };

    /** API endpoint configuration - use endpoint objects from pairs.ts directly */
    endpoints?: {
        findOne?: EndpointDefinition;
        findMany?: EndpointDefinition;
        createOne?: EndpointDefinition;
        createMany?: EndpointDefinition;
        updateOne?: EndpointDefinition;
        updateMany?: EndpointDefinition;
        deleteOne?: EndpointDefinition;
        deleteMany?: EndpointDefinition;
        [key: string]: EndpointDefinition | undefined;
    };
}

/**
 * Standardized test fixture structure for forms
 * This keeps test data separate from production code and provides
 * a consistent structure for testing valid/invalid cases.
 */
export interface FormFixtures<TShape extends { __typename: string; id: string }> {
    /** Reference to the form config this fixture is for */
    configType: string;

    /** Valid test scenarios that should pass validation */
    valid: {
        /** Minimal valid data with only required fields */
        minimal: TShape;
        /** Complete valid data with all fields populated */
        complete: TShape;
        /** Additional valid scenarios */
        [key: string]: TShape;
    };

    /** Invalid test scenarios that should fail validation */
    invalid: {
        /** Missing required fields */
        missingRequired: TShape;
        /** Invalid field values */
        invalidValues: TShape;
        /** Additional invalid scenarios */
        [key: string]: TShape;
    };

    /** Edge case scenarios for boundary testing */
    edge: {
        /** Maximum allowed values (e.g., max length strings) */
        maxValues: TShape;
        /** Minimum allowed values */
        minValues: TShape;
        /** Additional edge cases */
        [key: string]: TShape;
    };
}
