import { describe, it, expect } from 'vitest';
import {
  filterByProperty,
  filterByProperties,
  sortBy,
  sortByDate,
  toggleInSet,
  selectAll,
  removeFromSet,
  removeById,
  removeByIds,
  updateById,
  searchByProperty,
  getUniqueValues,
  countByProperty,
  groupBy,
} from './collections';

interface TestItem {
  id: number;
  name: string;
  status: string;
  created_at: string;
}

const createTestItems = (): TestItem[] => [
  { id: 1, name: 'Alice', status: 'active', created_at: '2024-01-15T10:00:00Z' },
  { id: 2, name: 'Bob', status: 'pending', created_at: '2024-01-16T10:00:00Z' },
  { id: 3, name: 'Charlie', status: 'active', created_at: '2024-01-14T10:00:00Z' },
  { id: 4, name: 'Diana', status: 'archived', created_at: '2024-01-17T10:00:00Z' },
];

describe('collections', () => {
  describe('filterByProperty', () => {
    it('filters items by property value', () => {
      const items = createTestItems();
      const result = filterByProperty(items, 'status', 'active');
      expect(result).toHaveLength(2);
      expect(result.every((item) => item.status === 'active')).toBe(true);
    });

    it('returns all items when value is "all"', () => {
      const items = createTestItems();
      const result = filterByProperty(items, 'status', 'all');
      expect(result).toHaveLength(4);
    });

    it('returns empty array when no items match', () => {
      const items = createTestItems();
      const result = filterByProperty(items, 'status', 'nonexistent');
      expect(result).toHaveLength(0);
    });

    it('handles empty array', () => {
      const result = filterByProperty([] as TestItem[], 'status', 'active');
      expect(result).toHaveLength(0);
    });

    it('does not mutate original array', () => {
      const items = createTestItems();
      filterByProperty(items, 'status', 'active');
      expect(items).toHaveLength(4);
    });
  });

  describe('filterByProperties', () => {
    it('filters by multiple properties', () => {
      const items = createTestItems();
      const result = filterByProperties(items, { status: 'active', name: 'Alice' });
      expect(result).toHaveLength(1);
      expect(result[0]?.id).toBe(1);
    });

    it('returns all when all filters are "all"', () => {
      const items = createTestItems();
      const result = filterByProperties(items, { status: 'all' });
      expect(result).toHaveLength(4);
    });

    it('handles mix of actual values and "all"', () => {
      const items = createTestItems();
      const result = filterByProperties(items, { status: 'active', name: 'all' as TestItem['name'] | 'all' });
      expect(result).toHaveLength(2);
    });
  });

  describe('sortBy', () => {
    it('sorts by property ascending', () => {
      const items = createTestItems();
      const result = sortBy(items, 'name', 'asc');
      expect(result[0]?.name).toBe('Alice');
      expect(result[3]?.name).toBe('Diana');
    });

    it('sorts by property descending', () => {
      const items = createTestItems();
      const result = sortBy(items, 'name', 'desc');
      expect(result[0]?.name).toBe('Diana');
      expect(result[3]?.name).toBe('Alice');
    });

    it('sorts by numeric property', () => {
      const items = createTestItems();
      const result = sortBy(items, 'id', 'desc');
      expect(result[0]?.id).toBe(4);
      expect(result[3]?.id).toBe(1);
    });

    it('does not mutate original array', () => {
      const items = createTestItems();
      const originalFirst = items[0];
      sortBy(items, 'name', 'desc');
      expect(items[0]).toBe(originalFirst);
    });

    it('defaults to ascending order', () => {
      const items = createTestItems();
      const result = sortBy(items, 'id');
      expect(result[0]?.id).toBe(1);
    });

    it('handles null values (sorts to end for asc)', () => {
      const items: Array<{ id: number; value: number | null }> = [
        { id: 1, value: 5 },
        { id: 2, value: null },
        { id: 3, value: 3 },
      ];
      const result = sortBy(items, 'value', 'asc');
      expect(result[0]?.value).toBe(3);
      expect(result[1]?.value).toBe(5);
      expect(result[2]?.value).toBeNull();
    });
  });

  describe('sortByDate', () => {
    it('sorts by date descending (newest first) by default', () => {
      const items = createTestItems();
      const result = sortByDate(items, 'created_at');
      expect(result[0]?.name).toBe('Diana'); // Jan 17
      expect(result[3]?.name).toBe('Charlie'); // Jan 14
    });

    it('sorts by date ascending (oldest first)', () => {
      const items = createTestItems();
      const result = sortByDate(items, 'created_at', 'asc');
      expect(result[0]?.name).toBe('Charlie'); // Jan 14
      expect(result[3]?.name).toBe('Diana'); // Jan 17
    });

    it('handles null date values', () => {
      const items: Array<{ id: number; date: string | null }> = [
        { id: 1, date: '2024-01-15' },
        { id: 2, date: null },
        { id: 3, date: '2024-01-10' },
      ];
      const result = sortByDate(items, 'date', 'desc');
      expect(result[0]?.date).toBe('2024-01-15');
      expect(result[2]?.date).toBeNull();
    });
  });

  describe('toggleInSet', () => {
    it('adds item to set if not present', () => {
      const set = new Set([1, 2]);
      const result = toggleInSet(set, 3);
      expect(result.has(3)).toBe(true);
      expect(result.size).toBe(3);
    });

    it('removes item from set if present', () => {
      const set = new Set([1, 2, 3]);
      const result = toggleInSet(set, 2);
      expect(result.has(2)).toBe(false);
      expect(result.size).toBe(2);
    });

    it('does not mutate original set', () => {
      const set = new Set([1, 2]);
      toggleInSet(set, 3);
      expect(set.has(3)).toBe(false);
    });
  });

  describe('selectAll', () => {
    it('selects all items when not all selected', () => {
      const items = createTestItems();
      const selection = new Set([1]);
      const result = selectAll(items, selection);
      expect(result.size).toBe(4);
      expect(result.has(1)).toBe(true);
      expect(result.has(4)).toBe(true);
    });

    it('clears selection when all items selected', () => {
      const items = createTestItems();
      const selection = new Set([1, 2, 3, 4]);
      const result = selectAll(items, selection);
      expect(result.size).toBe(0);
    });

    it('uses custom id key', () => {
      const items = [{ key: 'a' }, { key: 'b' }];
      const selection = new Set<string>();
      const result = selectAll(items, selection, 'key');
      expect(result.has('a')).toBe(true);
      expect(result.has('b')).toBe(true);
    });
  });

  describe('removeFromSet', () => {
    it('removes item from set', () => {
      const set = new Set([1, 2, 3]);
      const result = removeFromSet(set, 2);
      expect(result.has(2)).toBe(false);
      expect(result.size).toBe(2);
    });

    it('returns unchanged set if item not present', () => {
      const set = new Set([1, 2]);
      const result = removeFromSet(set, 5);
      expect(result.size).toBe(2);
    });

    it('does not mutate original set', () => {
      const set = new Set([1, 2]);
      removeFromSet(set, 1);
      expect(set.has(1)).toBe(true);
    });
  });

  describe('removeById', () => {
    it('removes item by id', () => {
      const items = createTestItems();
      const result = removeById(items, 2);
      expect(result).toHaveLength(3);
      expect(result.find((item) => item.id === 2)).toBeUndefined();
    });

    it('returns unchanged array if id not found', () => {
      const items = createTestItems();
      const result = removeById(items, 999);
      expect(result).toHaveLength(4);
    });

    it('does not mutate original array', () => {
      const items = createTestItems();
      removeById(items, 1);
      expect(items).toHaveLength(4);
    });

    it('does not mutate frozen array', () => {
      const items = Object.freeze([...createTestItems()]);
      const result = removeById(items as TestItem[], 1);
      expect(items).toHaveLength(4);
      expect(result).toHaveLength(3);
    });

    it('uses custom id key', () => {
      const items = [{ key: 'a', name: 'A' }, { key: 'b', name: 'B' }];
      const result = removeById(items, 'a', 'key');
      expect(result).toHaveLength(1);
      expect(result[0]?.key).toBe('b');
    });
  });

  describe('removeByIds', () => {
    it('removes multiple items by ids', () => {
      const items = createTestItems();
      const result = removeByIds(items, new Set([1, 3]));
      expect(result).toHaveLength(2);
      expect(result.map((item) => item.id)).toEqual([2, 4]);
    });

    it('handles empty set', () => {
      const items = createTestItems();
      const result = removeByIds(items, new Set());
      expect(result).toHaveLength(4);
    });

    it('ignores non-existent ids', () => {
      const items = createTestItems();
      const result = removeByIds(items, new Set([999, 1000]));
      expect(result).toHaveLength(4);
    });

    it('does not mutate original array', () => {
      const items = createTestItems();
      removeByIds(items, new Set([1, 2]));
      expect(items).toHaveLength(4);
    });

    it('does not mutate frozen array', () => {
      const items = Object.freeze([...createTestItems()]);
      const result = removeByIds(items as TestItem[], new Set([1, 2]));
      expect(items).toHaveLength(4);
      expect(result).toHaveLength(2);
    });
  });

  describe('updateById', () => {
    it('updates item by id', () => {
      const items = createTestItems();
      const item1 = items[1];
      expect(item1).toBeDefined();
      const updated = { ...item1, name: 'Bobby' };
      const result = updateById(items, 2, updated);
      expect(result[1]?.name).toBe('Bobby');
    });

    it('returns unchanged array if id not found', () => {
      const items = createTestItems();
      const updated = { id: 999, name: 'Nobody', status: 'x', created_at: '' };
      const result = updateById(items, 999, updated);
      expect(result).toEqual(items);
    });

    it('does not mutate original array', () => {
      const items = createTestItems();
      const item0 = items[0];
      expect(item0).toBeDefined();
      const updated = { ...item0, name: 'New' };
      updateById(items, 1, updated);
      expect(items[0]?.name).toBe('Alice');
    });

    it('does not mutate frozen array', () => {
      const items = Object.freeze([...createTestItems()]);
      const testItem = createTestItems()[0];
      expect(testItem).toBeDefined();
      const updated = { ...testItem, name: 'New' };
      const result = updateById(items as TestItem[], 1, updated);
      expect(items[0]?.name).toBe('Alice');
      expect(result[0]?.name).toBe('New');
    });
  });

  describe('searchByProperty', () => {
    it('finds items by substring match', () => {
      const items = createTestItems();
      const result = searchByProperty(items, 'name', 'li');
      expect(result).toHaveLength(2);
      expect(result.map((item) => item.name).sort()).toEqual(['Alice', 'Charlie']);
    });

    it('is case insensitive', () => {
      const items = createTestItems();
      const result = searchByProperty(items, 'name', 'ALICE');
      expect(result).toHaveLength(1);
      expect(result[0]?.name).toBe('Alice');
    });

    it('returns all items for empty query', () => {
      const items = createTestItems();
      const result = searchByProperty(items, 'name', '');
      expect(result).toHaveLength(4);
    });

    it('returns all items for whitespace-only query', () => {
      const items = createTestItems();
      const result = searchByProperty(items, 'name', '   ');
      expect(result).toHaveLength(4);
    });

    it('returns empty array when no matches', () => {
      const items = createTestItems();
      const result = searchByProperty(items, 'name', 'xyz');
      expect(result).toHaveLength(0);
    });
  });

  describe('getUniqueValues', () => {
    it('returns unique values sorted', () => {
      const items = createTestItems();
      const result = getUniqueValues(items, 'status');
      expect(result).toEqual(['active', 'archived', 'pending']);
    });

    it('handles empty array', () => {
      const result = getUniqueValues([], 'status');
      expect(result).toEqual([]);
    });

    it('handles single item', () => {
      const testItem = createTestItems()[0];
      expect(testItem).toBeDefined();
      if (!testItem) throw new Error('Expected testItem to be defined');
      const items = [testItem];
      const result = getUniqueValues(items, 'status');
      expect(result).toEqual(['active']);
    });
  });

  describe('countByProperty', () => {
    it('counts items matching value', () => {
      const items = createTestItems();
      expect(countByProperty(items, 'status', 'active')).toBe(2);
      expect(countByProperty(items, 'status', 'pending')).toBe(1);
      expect(countByProperty(items, 'status', 'archived')).toBe(1);
    });

    it('returns 0 for non-matching value', () => {
      const items = createTestItems();
      expect(countByProperty(items, 'status', 'nonexistent')).toBe(0);
    });

    it('returns 0 for empty array', () => {
      expect(countByProperty([] as TestItem[], 'status', 'active')).toBe(0);
    });
  });

  describe('groupBy', () => {
    it('groups items by property', () => {
      const items = createTestItems();
      const result = groupBy(items, 'status');
      expect(Object.keys(result).sort()).toEqual(['active', 'archived', 'pending']);
      expect(result['active']).toHaveLength(2);
      expect(result['pending']).toHaveLength(1);
      expect(result['archived']).toHaveLength(1);
    });

    it('handles empty array', () => {
      const result = groupBy([], 'status');
      expect(result).toEqual({});
    });

    it('preserves item order within groups', () => {
      const items = createTestItems();
      const result = groupBy(items, 'status');
      expect(result['active']?.[0]?.name).toBe('Alice');
      expect(result['active']?.[1]?.name).toBe('Charlie');
    });
  });
});
