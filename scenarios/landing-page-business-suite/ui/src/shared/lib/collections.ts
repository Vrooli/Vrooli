/**
 * Generic collection utility functions for filtering, sorting, and selection operations.
 *
 * These functions standardize common collection operations across the application,
 * reducing duplication and ensuring consistent behavior.
 */

/**
 * Sort direction for sortBy operations
 */
export type SortDirection = 'asc' | 'desc';

/**
 * Filter items by a property value.
 *
 * Returns all items where the specified property equals the given value.
 * Pass 'all' as value to return all items (no filtering).
 *
 * @param items - Array of items to filter
 * @param property - Property key to filter on
 * @param value - Value to match, or 'all' to skip filtering
 * @returns Filtered array
 *
 * @example
 * filterByProperty(users, 'role', 'admin')   // users where role === 'admin'
 * filterByProperty(users, 'role', 'all')     // all users (no filter)
 */
export function filterByProperty<T, K extends keyof T>(
  items: T[],
  property: K,
  value: T[K] | 'all'
): T[] {
  if (value === 'all') {
    return items;
  }
  return items.filter((item) => item[property] === value);
}

/**
 * Filter items by multiple property values (multi-filter).
 *
 * Returns items that match ALL specified filters.
 * Properties with value 'all' are skipped.
 *
 * @param items - Array of items to filter
 * @param filters - Object mapping property keys to filter values
 * @returns Filtered array
 *
 * @example
 * filterByProperties(feedback, { status: 'pending', type: 'bug' })
 */
export function filterByProperties<T>(
  items: T[],
  filters: Partial<Record<keyof T, T[keyof T] | 'all'>>
): T[] {
  return items.filter((item) => {
    for (const [key, value] of Object.entries(filters)) {
      if (value !== 'all' && item[key as keyof T] !== value) {
        return false;
      }
    }
    return true;
  });
}

/**
 * Sort items by a property.
 *
 * @param items - Array of items to sort
 * @param key - Property key to sort by
 * @param direction - Sort direction ('asc' or 'desc', default 'asc')
 * @returns New sorted array (does not mutate original)
 *
 * @example
 * sortBy(users, 'name', 'asc')       // A-Z by name
 * sortBy(products, 'price', 'desc')  // highest price first
 */
export function sortBy<T>(
  items: T[],
  key: keyof T,
  direction: SortDirection = 'asc'
): T[] {
  return [...items].sort((a, b) => {
    const aVal = a[key];
    const bVal = b[key];

    // Handle null/undefined
    if (aVal == null && bVal == null) return 0;
    if (aVal == null) return direction === 'asc' ? 1 : -1;
    if (bVal == null) return direction === 'asc' ? -1 : 1;

    // Compare values
    if (aVal < bVal) return direction === 'asc' ? -1 : 1;
    if (aVal > bVal) return direction === 'asc' ? 1 : -1;
    return 0;
  });
}

/**
 * Sort items by a date property.
 *
 * Convenience function for sorting by date strings.
 *
 * @param items - Array of items to sort
 * @param key - Property key containing date string
 * @param direction - Sort direction ('asc' for oldest first, 'desc' for newest first)
 * @returns New sorted array
 *
 * @example
 * sortByDate(emails, 'created_at', 'desc')  // newest first
 */
export function sortByDate<T>(
  items: T[],
  key: keyof T,
  direction: SortDirection = 'desc'
): T[] {
  return [...items].sort((a, b) => {
    const aVal = a[key];
    const bVal = b[key];

    // Handle null/undefined
    if (aVal == null && bVal == null) return 0;
    if (aVal == null) return 1;
    if (bVal == null) return -1;

    const dateA = new Date(aVal as string).getTime();
    const dateB = new Date(bVal as string).getTime();

    return direction === 'asc' ? dateA - dateB : dateB - dateA;
  });
}

/**
 * Toggle an item in a Set.
 *
 * Returns a new Set with the item added if not present, or removed if present.
 *
 * @param set - Current Set
 * @param item - Item to toggle
 * @returns New Set with item toggled
 *
 * @example
 * const selection = new Set([1, 2]);
 * toggleInSet(selection, 2) // Set([1])
 * toggleInSet(selection, 3) // Set([1, 2, 3])
 */
export function toggleInSet<T>(set: Set<T>, item: T): Set<T> {
  const next = new Set(set);
  if (next.has(item)) {
    next.delete(item);
  } else {
    next.add(item);
  }
  return next;
}

/**
 * Select all items or clear selection.
 *
 * If current selection equals all items, returns empty Set.
 * Otherwise, returns Set containing all item IDs.
 *
 * @param items - Array of items with id property
 * @param currentSelection - Current selection Set
 * @param idKey - Property key for the item ID (default: 'id')
 * @returns New Set with all selected or empty
 *
 * @example
 * const items = [{ id: 1 }, { id: 2 }];
 * selectAll(items, new Set([1]))    // Set([1, 2]) - select all
 * selectAll(items, new Set([1, 2])) // Set() - deselect all
 */
export function selectAll<T, K extends keyof T = 'id' & keyof T>(
  items: T[],
  currentSelection: Set<T[K]>,
  idKey: K = 'id' as K
): Set<T[K]> {
  if (currentSelection.size === items.length) {
    return new Set();
  }
  return new Set(items.map((item) => item[idKey]));
}

/**
 * Remove an item from a Set.
 *
 * @param set - Current Set
 * @param item - Item to remove
 * @returns New Set without the item
 */
export function removeFromSet<T>(set: Set<T>, item: T): Set<T> {
  const next = new Set(set);
  next.delete(item);
  return next;
}

/**
 * Remove an item from an array by ID.
 *
 * @param items - Array of items
 * @param id - ID of item to remove
 * @param idKey - Property key for the item ID (default: 'id')
 * @returns New array without the item
 *
 * @example
 * removeById(users, 5)              // remove user with id 5
 * removeById(items, 'abc', 'key')   // remove item with key 'abc'
 */
export function removeById<T, K extends keyof T = 'id' & keyof T>(
  items: T[],
  id: T[K],
  idKey: K = 'id' as K
): T[] {
  return items.filter((item) => item[idKey] !== id);
}

/**
 * Remove multiple items from an array by their IDs.
 *
 * @param items - Array of items
 * @param ids - Set of IDs to remove
 * @param idKey - Property key for the item ID (default: 'id')
 * @returns New array without the specified items
 *
 * @example
 * removeByIds(users, new Set([1, 2, 3]))
 */
export function removeByIds<T, K extends keyof T = 'id' & keyof T>(
  items: T[],
  ids: Set<T[K]>,
  idKey: K = 'id' as K
): T[] {
  return items.filter((item) => !ids.has(item[idKey]));
}

/**
 * Update an item in an array by ID.
 *
 * @param items - Array of items
 * @param id - ID of item to update
 * @param updated - Updated item (replaces the existing item)
 * @param idKey - Property key for the item ID (default: 'id')
 * @returns New array with the item updated
 *
 * @example
 * updateById(users, 5, { ...user, name: 'New Name' })
 */
export function updateById<T, K extends keyof T = 'id' & keyof T>(
  items: T[],
  id: T[K],
  updated: T,
  idKey: K = 'id' as K
): T[] {
  return items.map((item) => (item[idKey] === id ? updated : item));
}

/**
 * Search items by a text property (case-insensitive substring match).
 *
 * @param items - Array of items
 * @param property - Property key to search in
 * @param query - Search query string
 * @returns Filtered array of matching items
 *
 * @example
 * searchByProperty(users, 'email', 'gmail')  // users with 'gmail' in email
 */
export function searchByProperty<T>(
  items: T[],
  property: keyof T,
  query: string
): T[] {
  const trimmed = query.trim();
  if (!trimmed) {
    return items;
  }
  const lowerQuery = trimmed.toLowerCase();
  return items.filter((item) => {
    const value = item[property];
    if (typeof value !== 'string') return false;
    return value.toLowerCase().includes(lowerQuery);
  });
}

/**
 * Get unique values for a property from an array.
 *
 * @param items - Array of items
 * @param property - Property key to extract values from
 * @returns Sorted array of unique values
 *
 * @example
 * getUniqueValues(emails, 'source')  // ['landing', 'newsletter', 'waitlist']
 */
export function getUniqueValues<T, K extends keyof T>(
  items: T[],
  property: K
): T[K][] {
  const values = new Set(items.map((item) => item[property]));
  return Array.from(values).sort();
}

/**
 * Count items matching a property value.
 *
 * @param items - Array of items
 * @param property - Property key to check
 * @param value - Value to count
 * @returns Number of items matching the value
 *
 * @example
 * countByProperty(feedback, 'status', 'pending')  // 5
 */
export function countByProperty<T, K extends keyof T>(
  items: T[],
  property: K,
  value: T[K]
): number {
  return items.filter((item) => item[property] === value).length;
}

/**
 * Group items by a property value.
 *
 * @param items - Array of items
 * @param property - Property key to group by
 * @returns Object with property values as keys and arrays of items as values
 *
 * @example
 * groupBy(users, 'role')  // { admin: [...], user: [...] }
 */
export function groupBy<T>(
  items: T[],
  property: keyof T
): Record<string, T[]> {
  return items.reduce<Record<string, T[]>>(
    (groups, item) => {
      const key = String(item[property]);
      if (!groups[key]) {
        groups[key] = [];
      }
      groups[key].push(item);
      return groups;
    },
    {}
  );
}
