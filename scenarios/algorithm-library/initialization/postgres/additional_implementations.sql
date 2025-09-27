-- Additional Algorithm Implementations
-- This file adds JavaScript and Go implementations for existing algorithms

-- Get algorithm IDs
WITH algo_ids AS (
    SELECT id, name FROM algorithms
)

-- Add JavaScript implementations for sorting algorithms
INSERT INTO implementations (algorithm_id, language, code, is_primary, validated, version) 
SELECT id, 'javascript', 
CASE name
    WHEN 'quicksort' THEN 'function quicksort(arr) {
    if (arr.length <= 1) {
        return arr;
    }
    
    const pivot = arr[Math.floor(arr.length / 2)];
    const left = arr.filter(x => x < pivot);
    const middle = arr.filter(x => x === pivot);
    const right = arr.filter(x => x > pivot);
    
    return [...quicksort(left), ...middle, ...quicksort(right)];
}'

    WHEN 'mergesort' THEN 'function mergesort(arr) {
    if (arr.length <= 1) {
        return arr;
    }
    
    const mid = Math.floor(arr.length / 2);
    const left = mergesort(arr.slice(0, mid));
    const right = mergesort(arr.slice(mid));
    
    return merge(left, right);
}

function merge(left, right) {
    const result = [];
    let i = 0, j = 0;
    
    while (i < left.length && j < right.length) {
        if (left[i] <= right[j]) {
            result.push(left[i]);
            i++;
        } else {
            result.push(right[j]);
            j++;
        }
    }
    
    return [...result, ...left.slice(i), ...right.slice(j)];
}'

    WHEN 'binary_search' THEN 'function binarySearch(arr, target) {
    let left = 0;
    let right = arr.length - 1;
    
    while (left <= right) {
        const mid = Math.floor((left + right) / 2);
        
        if (arr[mid] === target) {
            return mid;
        } else if (arr[mid] < target) {
            left = mid + 1;
        } else {
            right = mid - 1;
        }
    }
    
    return -1;
}'

    WHEN 'dfs' THEN 'function dfs(graph, start, visited = new Set()) {
    visited.add(start);
    const result = [start];
    
    for (const neighbor of graph[start] || []) {
        if (!visited.has(neighbor)) {
            result.push(...dfs(graph, neighbor, visited));
        }
    }
    
    return result;
}'

    WHEN 'bfs' THEN 'function bfs(graph, start) {
    const visited = new Set();
    const queue = [start];
    const result = [];
    
    visited.add(start);
    
    while (queue.length > 0) {
        const vertex = queue.shift();
        result.push(vertex);
        
        for (const neighbor of graph[vertex] || []) {
            if (!visited.has(neighbor)) {
                visited.add(neighbor);
                queue.push(neighbor);
            }
        }
    }
    
    return result;
}'

    WHEN 'fibonacci' THEN 'function fibonacci(n) {
    if (n <= 1) {
        return n;
    }
    
    let prev = 0;
    let curr = 1;
    
    for (let i = 2; i <= n; i++) {
        const next = prev + curr;
        prev = curr;
        curr = next;
    }
    
    return curr;
}'

    WHEN 'heapsort' THEN 'function heapsort(arr) {
    const result = [...arr];
    const n = result.length;
    
    // Build max heap
    for (let i = Math.floor(n / 2) - 1; i >= 0; i--) {
        heapify(result, n, i);
    }
    
    // Extract elements from heap
    for (let i = n - 1; i > 0; i--) {
        [result[0], result[i]] = [result[i], result[0]];
        heapify(result, i, 0);
    }
    
    return result;
}

function heapify(arr, n, i) {
    let largest = i;
    const left = 2 * i + 1;
    const right = 2 * i + 2;
    
    if (left < n && arr[left] > arr[largest]) {
        largest = left;
    }
    
    if (right < n && arr[right] > arr[largest]) {
        largest = right;
    }
    
    if (largest !== i) {
        [arr[i], arr[largest]] = [arr[largest], arr[i]];
        heapify(arr, n, largest);
    }
}'

    WHEN 'dijkstra' THEN 'function dijkstra(graph, start) {
    const distances = {};
    const visited = new Set();
    const pq = [[0, start]]; // [distance, node]
    
    // Initialize distances
    for (const node in graph) {
        distances[node] = Infinity;
    }
    distances[start] = 0;
    
    while (pq.length > 0) {
        // Sort to simulate priority queue
        pq.sort((a, b) => a[0] - b[0]);
        const [currentDist, current] = pq.shift();
        
        if (visited.has(current)) continue;
        visited.add(current);
        
        for (const [neighbor, weight] of Object.entries(graph[current] || {})) {
            const distance = currentDist + weight;
            
            if (distance < distances[neighbor]) {
                distances[neighbor] = distance;
                pq.push([distance, neighbor]);
            }
        }
    }
    
    return distances;
}'

    ELSE NULL
END AS code,
false, true, '1.0.0'
FROM algo_ids
WHERE name IN ('quicksort', 'mergesort', 'binary_search', 'dfs', 'bfs', 'fibonacci', 'heapsort', 'dijkstra')
AND code IS NOT NULL;

-- Add Go implementations
INSERT INTO implementations (algorithm_id, language, code, is_primary, validated, version) 
SELECT id, 'go', 
CASE name
    WHEN 'quicksort' THEN 'func quicksort(arr []int) []int {
    if len(arr) <= 1 {
        return arr
    }
    
    pivot := arr[len(arr)/2]
    var left, middle, right []int
    
    for _, v := range arr {
        if v < pivot {
            left = append(left, v)
        } else if v == pivot {
            middle = append(middle, v)
        } else {
            right = append(right, v)
        }
    }
    
    result := quicksort(left)
    result = append(result, middle...)
    result = append(result, quicksort(right)...)
    
    return result
}'

    WHEN 'binary_search' THEN 'func binarySearch(arr []int, target int) int {
    left, right := 0, len(arr)-1
    
    for left <= right {
        mid := (left + right) / 2
        
        if arr[mid] == target {
            return mid
        } else if arr[mid] < target {
            left = mid + 1
        } else {
            right = mid - 1
        }
    }
    
    return -1
}'

    WHEN 'fibonacci' THEN 'func fibonacci(n int) int {
    if n <= 1 {
        return n
    }
    
    prev, curr := 0, 1
    
    for i := 2; i <= n; i++ {
        next := prev + curr
        prev = curr
        curr = next
    }
    
    return curr
}'

    ELSE NULL
END AS code,
false, true, '1.0.0'
FROM algo_ids
WHERE name IN ('quicksort', 'binary_search', 'fibonacci')
AND code IS NOT NULL;

-- Add Python implementations for algorithms that don't have them yet
INSERT INTO implementations (algorithm_id, language, code, is_primary, validated, version) 
SELECT id, 'python', 
CASE name
    WHEN 'heapsort' THEN 'def heapsort(arr):
    import heapq
    result = arr.copy()
    heapq.heapify(result)
    return [heapq.heappop(result) for _ in range(len(result))]'

    WHEN 'dijkstra' THEN 'import heapq

def dijkstra(graph, start):
    distances = {node: float("inf") for node in graph}
    distances[start] = 0
    pq = [(0, start)]
    visited = set()
    
    while pq:
        current_dist, current = heapq.heappop(pq)
        
        if current in visited:
            continue
        visited.add(current)
        
        for neighbor, weight in graph[current].items():
            distance = current_dist + weight
            
            if distance < distances[neighbor]:
                distances[neighbor] = distance
                heapq.heappush(pq, (distance, neighbor))
    
    return distances'

    WHEN 'insertion_sort' THEN 'def insertion_sort(arr):
    result = arr.copy()
    
    for i in range(1, len(result)):
        key = result[i]
        j = i - 1
        
        while j >= 0 and result[j] > key:
            result[j + 1] = result[j]
            j -= 1
        
        result[j + 1] = key
    
    return result'

    WHEN 'selection_sort' THEN 'def selection_sort(arr):
    result = arr.copy()
    n = len(result)
    
    for i in range(n):
        min_idx = i
        for j in range(i + 1, n):
            if result[j] < result[min_idx]:
                min_idx = j
        
        result[i], result[min_idx] = result[min_idx], result[i]
    
    return result'

    WHEN 'counting_sort' THEN 'def counting_sort(arr):
    if not arr:
        return []
    
    max_val = max(arr)
    min_val = min(arr)
    range_val = max_val - min_val + 1
    
    count = [0] * range_val
    output = [0] * len(arr)
    
    # Count occurrences
    for num in arr:
        count[num - min_val] += 1
    
    # Cumulative count
    for i in range(1, len(count)):
        count[i] += count[i - 1]
    
    # Build output array
    for num in reversed(arr):
        output[count[num - min_val] - 1] = num
        count[num - min_val] -= 1
    
    return output'

    WHEN 'bucket_sort' THEN 'def bucket_sort(arr):
    if not arr:
        return []
    
    # Create buckets
    num_buckets = len(arr)
    max_val = max(arr)
    min_val = min(arr)
    bucket_range = (max_val - min_val) / num_buckets + 1
    
    buckets = [[] for _ in range(num_buckets)]
    
    # Distribute elements into buckets
    for num in arr:
        index = int((num - min_val) / bucket_range)
        buckets[index].append(num)
    
    # Sort individual buckets and concatenate
    result = []
    for bucket in buckets:
        result.extend(sorted(bucket))
    
    return result'

    WHEN 'radix_sort' THEN 'def radix_sort(arr):
    if not arr:
        return []
    
    # Handle negative numbers by adding offset
    min_val = min(arr)
    offset = -min_val if min_val < 0 else 0
    arr = [x + offset for x in arr]
    
    max_val = max(arr)
    exp = 1
    
    while max_val // exp > 0:
        counting_sort_by_digit(arr, exp)
        exp *= 10
    
    # Remove offset
    return [x - offset for x in arr]

def counting_sort_by_digit(arr, exp):
    n = len(arr)
    output = [0] * n
    count = [0] * 10
    
    for i in range(n):
        index = (arr[i] // exp) % 10
        count[index] += 1
    
    for i in range(1, 10):
        count[i] += count[i - 1]
    
    for i in range(n - 1, -1, -1):
        index = (arr[i] // exp) % 10
        output[count[index] - 1] = arr[i]
        count[index] -= 1
    
    for i in range(n):
        arr[i] = output[i]'

    ELSE NULL
END AS code,
false, true, '1.0.0'
FROM algo_ids
WHERE name IN ('heapsort', 'dijkstra', 'insertion_sort', 'selection_sort', 'counting_sort', 'bucket_sort', 'radix_sort')
AND code IS NOT NULL;

-- Update implementation counts
UPDATE algorithms a
SET language_count = (
    SELECT COUNT(DISTINCT language) 
    FROM implementations i 
    WHERE i.algorithm_id = a.id
),
has_validated_impl = EXISTS (
    SELECT 1 FROM implementations i 
    WHERE i.algorithm_id = a.id AND i.validated = true
);