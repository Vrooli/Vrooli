// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-18
import { type FormSchema } from "@vrooli/shared";
import { afterEach, describe, expect, it, vi } from "vitest";
import { normalizeFormContainers } from "./FormView.js";

describe("normalizeFormContainers", () => {
    afterEach(() => {
        vi.clearAllMocks();
    });

    it("renders all form elements in single container by default", () => {
        // GIVEN: A form with 3 elements but no container configuration
        const formSchema = {
            elements: [{}, {}, {}],
        } as unknown as FormSchema;

        // WHEN: Form is rendered
        const containers = normalizeFormContainers(formSchema);

        // THEN: All elements are grouped in one container
        expect(containers).toEqual([{
            totalItems: 3,
        }]);
    });

    it("groups form elements according to container configuration", () => {
        // GIVEN: A form with 5 elements and specific grouping
        const formSchema = {
            elements: [{}, {}, {}, {}, {}],
            containers: [
                { totalItems: 3, disableCollapse: true }, // First 3 elements
                { totalItems: 2 },                         // Next 2 elements
            ],
        } as unknown as FormSchema;

        // WHEN: Form layout is calculated
        const containers = normalizeFormContainers(formSchema);

        // THEN: Elements are grouped as specified
        expect(containers).toEqual([
            { totalItems: 3, disableCollapse: true },
            { totalItems: 2, disableCollapse: undefined },
        ]);
    });

    it("limits container size when specified size exceeds available elements", () => {
        // GIVEN: Container requests 5 items but only 4 elements exist
        const formSchema = {
            elements: [{}, {}, {}, {}],
            containers: [
                { totalItems: 5 }, // Wants 5 but only 4 available
                { totalItems: 2 }, // Would overflow, so ignored
            ],
        } as unknown as FormSchema;

        // WHEN: Containers are normalized
        const containers = normalizeFormContainers(formSchema);

        // THEN: Container size is capped at available elements
        expect(containers).toEqual([
            { totalItems: 4, disableCollapse: undefined },
        ]);
    });

    it("creates additional container for remaining elements", () => {
        // GIVEN: 5 elements but containers only account for 4
        const formSchema: FormSchema = {
            elements: [{}, {}, {}, {}, {}],
            containers: [
                { totalItems: 2 }, // Elements 0-1
                { totalItems: 2 }, // Elements 2-3
                // Element 4 is unaccounted for
            ],
        } as unknown as FormSchema;

        // WHEN: Form layout is calculated
        const containers = normalizeFormContainers(formSchema);

        // THEN: Extra container is added for the orphaned element
        expect(containers).toEqual([
            { totalItems: 2 },
            { totalItems: 2 },
            { totalItems: 1 }, // Auto-generated for element 4
        ]);
    });

    it("returns empty containers when form has no elements", () => {
        // GIVEN: A form with no elements to display
        const formSchema = {
            elements: [],
            containers: [
                { totalItems: 3 },
                { totalItems: 2 },
            ],
        } as unknown as FormSchema;

        // WHEN: Form is rendered
        const containers = normalizeFormContainers(formSchema);

        // THEN: No containers are created (nothing to display)
        expect(containers).toEqual([]);
    });

    it("creates default container when no layout specified", () => {
        // GIVEN: Form elements with no container configuration
        const formSchema = {
            elements: [{}, {}, {}, {}, {}],
            containers: [],
        } as unknown as FormSchema;

        // WHEN: Form needs to render
        const containers = normalizeFormContainers(formSchema);

        // THEN: Single container holds all elements
        expect(containers).toEqual([{
            totalItems: 5,
        }]);
    });

    it("preserves collapse settings for form sections", () => {
        // GIVEN: Form sections with different collapse behaviors
        const formSchema = {
            elements: [{}, {}, {}, {}, {}],
            containers: [
                { totalItems: 2, disableCollapse: true },  // Always expanded section
                { totalItems: 3, disableCollapse: false }, // Collapsible section
            ],
        } as unknown as FormSchema;

        // WHEN: Form containers are normalized
        const containers = normalizeFormContainers(formSchema);

        // THEN: Collapse settings are preserved
        expect(containers).toEqual([
            { totalItems: 2, disableCollapse: true },
            { totalItems: 3, disableCollapse: false },
        ]);
    });
});
