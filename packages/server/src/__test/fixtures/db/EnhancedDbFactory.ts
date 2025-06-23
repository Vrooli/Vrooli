import { generatePK, generatePublicId } from "@vrooli/shared";
import { nanoid } from "@vrooli/shared";
import type { 
    DbTestFixtures, 
    DbFactoryConfig, 
    DbFactoryResult, 
    DbFixtureValidation, 
    EnhancedDbFactory as IEnhancedDbFactory,
    DbErrorScenarios,
} from "./types.js";

/**
 * Abstract base class for enhanced database factories
 * Provides common functionality and enforces consistent patterns
 */
export abstract class EnhancedDbFactory<TCreate, TUpdate = Partial<TCreate>> 
    implements IEnhancedDbFactory<TCreate, TUpdate> {
    
    /**
     * Get the test fixtures for this factory
     * Must be implemented by subclasses
     */
    protected abstract getFixtures(): DbTestFixtures<TCreate, TUpdate>;

    /**
     * Get error scenarios for this model
     * Can be overridden by subclasses for model-specific errors
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {},
                foreignKeyViolation: {},
                checkConstraintViolation: {},
            },
            validation: {
                requiredFieldMissing: {},
                invalidDataType: {},
                outOfRange: {},
            },
            businessLogic: {},
        };
    }

    /**
     * Generate fresh IDs and handles to avoid conflicts
     * Can be overridden by subclasses to customize the shape
     */
    protected generateFreshIdentifiers(): Record<string, any> {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            handle: `test_${nanoid(8)}`,
        };
    }

    /**
     * Create minimal fixture with fresh identifiers
     */
    createMinimal(overrides?: Partial<TCreate>): TCreate {
        const fixtures = this.getFixtures();
        const fresh = this.generateFreshIdentifiers();
        
        return {
            ...fixtures.minimal,
            ...fresh,
            ...overrides,
        } as TCreate;
    }

    /**
     * Create complete fixture with fresh identifiers
     */
    createComplete(overrides?: Partial<TCreate>): TCreate {
        const fixtures = this.getFixtures();
        const fresh = this.generateFreshIdentifiers();
        
        return {
            ...fixtures.complete,
            ...fresh,
            ...overrides,
        } as TCreate;
    }

    /**
     * Create fixture with automatic relationship setup
     */
    createWithRelationships(config: DbFactoryConfig<TCreate>): DbFactoryResult<TCreate> {
        let data = this.createMinimal(config.overrides);
        let hasAuth = false;
        let teamCount = 0;
        let roleCount = 0;
        let relationCount = 0;

        // Add authentication if requested
        if (config.withAuth) {
            data = this.addAuthentication(data);
            hasAuth = true;
            relationCount++;
        }

        // Add team memberships if requested
        if (config.withTeams && config.withTeams.length > 0) {
            data = this.addTeamMemberships(data, config.withTeams);
            teamCount = config.withTeams.length;
            relationCount += teamCount;
        }

        // Add roles if requested
        if (config.withRoles && config.withRoles.length > 0) {
            data = this.addRoles(data, config.withRoles);
            roleCount = config.withRoles.length;
            relationCount += roleCount;
        }

        // Add other relations if requested
        if (config.withRelations) {
            data = this.addOtherRelations(data);
            relationCount++;
        }

        return {
            data,
            metadata: {
                hasAuth,
                teamCount,
                roleCount,
                relationCount,
            },
        };
    }

    /**
     * Validate fixture against basic constraints
     */
    validateFixture(data: TCreate): DbFixtureValidation {
        const errors: string[] = [];
        const warnings: string[] = [];

        try {
            // Basic validation - check for required fields
            const dataObj = data as Record<string, any>;
            
            if (!dataObj.id) {
                errors.push("Missing required field: id");
            }

            // Check for common patterns
            if (dataObj.handle && typeof dataObj.handle !== "string") {
                errors.push("Handle must be a string");
            }

            if (dataObj.name && typeof dataObj.name !== "string") {
                errors.push("Name must be a string");
            }

            // Model-specific validation
            const customValidation = this.validateSpecific(data);
            errors.push(...customValidation.errors);
            warnings.push(...customValidation.warnings);

        } catch (error) {
            errors.push(`Validation error: ${error instanceof Error ? error.message : "Unknown error"}`);
        }

        return {
            isValid: errors.length === 0,
            errors,
            warnings,
        };
    }

    /**
     * Create invalid fixture for error testing
     */
    createInvalid(scenario: string): any {
        const fixtures = this.getFixtures();
        const errorScenarios = this.getErrorScenarios();

        switch (scenario) {
            case "missingRequired":
                return fixtures.invalid.missingRequired;
            case "invalidTypes":
                return fixtures.invalid.invalidTypes;
            case "uniqueViolation":
                return errorScenarios.constraints.uniqueViolation;
            case "foreignKeyViolation":
                return errorScenarios.constraints.foreignKeyViolation;
            default:
                return fixtures.invalid[scenario] || fixtures.invalid.missingRequired;
        }
    }

    /**
     * Create edge case fixture
     */
    createEdgeCase(scenario: string): TCreate {
        const fixtures = this.getFixtures();
        const edgeCase = fixtures.edgeCases[scenario];
        
        if (!edgeCase) {
            throw new Error(`Unknown edge case scenario: ${scenario}`);
        }

        const fresh = this.generateFreshIdentifiers();
        return {
            ...edgeCase,
            ...fresh,
        } as TCreate;
    }

    /**
     * Get all available edge case scenarios
     */
    getAvailableEdgeCases(): string[] {
        const fixtures = this.getFixtures();
        return Object.keys(fixtures.edgeCases);
    }

    /**
     * Get all available invalid scenarios
     */
    getAvailableInvalidScenarios(): string[] {
        const fixtures = this.getFixtures();
        const errorScenarios = this.getErrorScenarios();
        
        return [
            ...Object.keys(fixtures.invalid),
            ...Object.keys(errorScenarios.constraints),
            ...Object.keys(errorScenarios.validation),
            ...Object.keys(errorScenarios.businessLogic),
        ];
    }

    // Protected methods for subclasses to override

    /**
     * Add authentication to a fixture
     * Override in subclasses that support authentication
     */
    protected addAuthentication(data: TCreate): TCreate {
        // Default implementation - no authentication added
        return data;
    }

    /**
     * Add team memberships to a fixture
     * Override in subclasses that support team membership
     */
    protected addTeamMemberships(data: TCreate, teams: Array<{ teamId: string; role: string }>): TCreate {
        // Default implementation - no teams added
        return data;
    }

    /**
     * Add roles to a fixture
     * Override in subclasses that support roles
     */
    protected addRoles(data: TCreate, roles: Array<{ id: string; name: string }>): TCreate {
        // Default implementation - no roles added
        return data;
    }

    /**
     * Add other relations to a fixture
     * Override in subclasses for model-specific relations
     */
    protected addOtherRelations(data: TCreate): TCreate {
        // Default implementation - no other relations added
        return data;
    }

    /**
     * Model-specific validation
     * Override in subclasses for custom validation logic
     */
    protected validateSpecific(data: TCreate): { errors: string[]; warnings: string[] } {
        return { errors: [], warnings: [] };
    }

    /**
     * Utility method to create a fixture with specific overrides
     */
    createCustom<K extends keyof TCreate>(
        base: "minimal" | "complete" | string,
        overrides: Partial<TCreate>,
    ): TCreate {
        let baseData: TCreate;

        switch (base) {
            case "minimal":
                baseData = this.createMinimal();
                break;
            case "complete":
                baseData = this.createComplete();
                break;
            default:
                // Try to find edge case
                try {
                    baseData = this.createEdgeCase(base);
                } catch {
                    baseData = this.createMinimal();
                }
        }

        return {
            ...baseData,
            ...overrides,
        };
    }

    /**
     * Create multiple fixtures with different configurations
     */
    createBatch(configs: Array<{
        type: "minimal" | "complete" | string;
        overrides?: Partial<TCreate>;
    }>): TCreate[] {
        return configs.map(config => 
            this.createCustom(config.type, config.overrides || {}),
        );
    }
}
