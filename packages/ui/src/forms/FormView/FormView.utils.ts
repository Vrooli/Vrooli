import type { GridSpacing } from "@mui/material";
import type { FormBuildViewProps, FormSchema, GridContainer } from "@vrooli/shared";
import type { PopoverListInput, PopoverListStructure } from "./FormView.types.js";

/**
 * Function to convert a FormSchema into an array of containers with start and end indices for rendering.
 * @param formSchema The schema defining the form layout, containers, and elements.
 * @returns An array of containers with start and end indices for the elements in each container. 
 * If no containers are provided, returns one container covering all elements. 
 * Ensures valid ranges for each container.
 */
export function normalizeFormContainers(
    formSchema: Partial<Pick<FormSchema, "containers" | "elements">>,
): GridContainer[] {
    if (typeof formSchema !== "object" || formSchema === null) {
        // If the schema is not an object, return an empty array
        return [];
    }

    const { elements, containers } = formSchema;

    if (!Array.isArray(elements) || elements.length === 0) {
        // If there are no elements, return an empty array
        return [];
    }

    if (!Array.isArray(containers) || containers.length === 0) {
        // If no containers are provided, return one container for all elements
        return [{
            totalItems: elements.length,
        }];
    }

    const gridContainers: GridContainer[] = [];
    let currentIndex = 0;

    containers.forEach(container => {
        if (currentIndex >= elements.length) {
            // If the current index is beyond the elements length, break the loop
            return;
        }

        const endIndex = Math.min(currentIndex + container.totalItems - 1, elements.length - 1);

        gridContainers.push({
            ...container,
            totalItems: endIndex - currentIndex + 1,
        });

        currentIndex = endIndex + 1;
    });

    // Ensure all elements are covered in case totalItems were more than elements length
    if (currentIndex < elements.length) {
        gridContainers.push({
            totalItems: elements.length - currentIndex,
        });
    }

    return gridContainers;
}

/**
 * Applies restrictions to the available form element types in 
 * one of the form builder popover categories
 */
export function filterCategory<T extends PopoverListInput | PopoverListStructure>(
    category: T[0],
    limits: FormBuildViewProps["limits"],
    popover: "inputs" | "structures",
) {
    if (limits?.[popover]?.types) {
        // Filter items based on the limits.input array
        const filteredItems = category.items.filter(item => limits?.[popover]?.types?.includes(item.type as never));

        // If filteredItems is not empty, return category with filtered items
        if (filteredItems.length > 0) {
            return { ...category, items: filteredItems };
        }
        // If filteredItems is empty, return null
        return null;
    }
    // Otherwise, return the category as-is
    return category;
}

/**
 * Calculates size of grid item based on the number of items in the grid. 
 * 1 item is { xs: 12 }, 
 * 2 items is { xs: 12, sm: 6 },
 * 3 items is { xs: 12, sm: 6, md: 4 },
 * 4+ items is { xs: 12, sm: 6, md: 4, lg: 3 }
 * @returns Size of grid item
 */
export function calculateGridItemSize(numItems: number): { [key: string]: GridSpacing } {
    switch (numItems) {
        case 1:
            return { xs: 12 };
        // eslint-disable-next-line no-magic-numbers
        case 2:
            return { xs: 12, sm: 6 };
        // eslint-disable-next-line no-magic-numbers
        case 3:
            return { xs: 12, sm: 6, md: 4 };
        default:
            return { xs: 12, sm: 6, md: 4, lg: 3 };
    }
}
