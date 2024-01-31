import { LRUCache } from './lruCache';

describe('LRUCache', () => {
    let cache: LRUCache<string, number>;

    beforeEach(() => {
        // Initialize a new cache before each test with a limit of 3 items
        cache = new LRUCache<string, number>(3);
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        jest.spyOn(console, "warn").mockImplementation(() => { });
    });
    afterAll(() => {
        jest.restoreAllMocks();
    });

    test('should add items to the cache', () => {
        cache.set('item1', 1);
        expect(cache.get('item1')).toBe(1);
    });

    test('should return undefined for missing items', () => {
        expect(cache.get('missingItem')).toBeUndefined();
    });

    test('should update the value if the same key is added to the cache', () => {
        cache.set('item', 1);
        cache.set('item', 2);
        expect(cache.get('item')).toBe(2);
    });

    test('should maintain the max limit by evicting the least recently used item', () => {
        cache.set('item1', 1);
        cache.set('item2', 2);
        cache.set('item3', 3);
        cache.set('item4', 4); // This should evict 'item1'

        expect(cache.get('item1')).toBeUndefined(); // 'item1' should have been evicted
        expect(cache.get('item2')).toBe(2);
        expect(cache.get('item3')).toBe(3);
        expect(cache.get('item4')).toBe(4);
    });

    test('should move the recently accessed item to the end', () => {
        cache.set('item1', 1);
        cache.set('item2', 2);
        cache.set('item3', 3);

        // Access 'item1' so it becomes the most recently used
        expect(cache.get('item1')).toBe(1);

        // Add another item, which should evict 'item2' instead of 'item1'
        cache.set('item4', 4);

        expect(cache.get('item1')).toBe(1);
        expect(cache.get('item2')).toBeUndefined();
        expect(cache.get('item3')).toBe(3);
        expect(cache.get('item4')).toBe(4);
    });

    test('should handle a large number of items', () => {
        const largeCache = new LRUCache<string, number>(1000);
        for (let i = 0; i < 2000; i++) {
            largeCache.set('item' + i, i);
        }

        // The cache should not grow beyond 1000 items
        expect(largeCache.size()).toBe(1000);

        // The first 1000 items should be evicted
        for (let i = 0; i < 1000; i++) {
            expect(largeCache.get('item' + i)).toBeUndefined();
        }

        // The last 1000 items should be present
        for (let i = 1000; i < 2000; i++) {
            expect(largeCache.get('item' + i)).toBe(i);
        }
    });

    test('should not add items over a certain size', () => {
        // Initialize a cache with a max size of 10 bytes
        const smallCache = new LRUCache<string, string>(3, 10);

        smallCache.set('smallItem', 'small'); // This should be added
        smallCache.set('largeItem', 'This is a large item'); // This should be skipped

        expect(smallCache.get('smallItem')).toBe('small');
        expect(smallCache.get('largeItem')).toBeUndefined();
    });

    test('should evict the least recently used item, respecting size limits', () => {
        // Initialize a cache with a max size of 10 bytes
        const smallCache = new LRUCache<string, string>(3, 10);

        smallCache.set('item1', '12345'); // 5 bytes
        smallCache.set('item2', '1234567890'); // 10 bytes, at the limit
        smallCache.set('item3', '12345'); // 5 bytes, 'item1' should be evicted next if needed

        // Access 'item2' to make it recently used
        expect(smallCache.get('item2')).toBe('1234567890');

        // Attempt to add an item that would exceed the size limit
        smallCache.set('largeItem', 'This is a large item'); // This should be skipped

        // 'item1' should still be evicted as 'largeItem' wasn't added
        smallCache.set('item4', '12345'); // 'item1' should be evicted

        expect(smallCache.get('item1')).toBeUndefined();
        expect(smallCache.get('item2')).toBe('1234567890');
        expect(smallCache.get('item3')).toBe('12345');
        expect(smallCache.get('item4')).toBe('12345');
        expect(smallCache.get('largeItem')).toBeUndefined(); // 'largeItem' was never added
    });
});
