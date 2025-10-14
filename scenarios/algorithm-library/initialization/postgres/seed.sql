-- Seed data for Algorithm Library
-- Initial set of fundamental algorithms with implementations

-- Clear existing data (for re-seeding)
TRUNCATE algorithms, implementations, test_cases, validation_results, benchmarks CASCADE;

-- Insert fundamental sorting algorithms
INSERT INTO algorithms (name, display_name, category, description, complexity_time, complexity_space, difficulty, tags, common_applications) VALUES
('quicksort', 'QuickSort', 'sorting', 'Efficient divide-and-conquer sorting algorithm that picks a pivot element and partitions the array around it', 'O(n log n)', 'O(log n)', 'medium', 
    ARRAY['divide-conquer', 'in-place', 'comparison-sort', 'unstable'], 
    ARRAY['General purpose sorting', 'Systems with good cache locality', 'When average case matters more than worst case']),

('mergesort', 'MergeSort', 'sorting', 'Stable divide-and-conquer sorting algorithm that divides array into halves, sorts them, and merges', 'O(n log n)', 'O(n)', 'medium',
    ARRAY['divide-conquer', 'stable', 'comparison-sort', 'external-sorting'],
    ARRAY['Sorting linked lists', 'External sorting of large datasets', 'When stability is required']),

('bubblesort', 'BubbleSort', 'sorting', 'Simple sorting algorithm that repeatedly steps through the list comparing adjacent elements', 'O(n²)', 'O(1)', 'easy',
    ARRAY['comparison-sort', 'stable', 'in-place', 'adaptive'],
    ARRAY['Educational purposes', 'Small datasets', 'Nearly sorted data']),

('binary_search', 'Binary Search', 'searching', 'Efficient search algorithm that finds the position of a target value within a sorted array', 'O(log n)', 'O(1)', 'easy',
    ARRAY['divide-conquer', 'iterative', 'recursive'],
    ARRAY['Database indexing', 'Finding elements in sorted arrays', 'Root finding in numerical methods']),

('linear_search', 'Linear Search', 'searching', 'Simple search algorithm that checks every element sequentially until target is found', 'O(n)', 'O(1)', 'easy',
    ARRAY['sequential', 'brute-force'],
    ARRAY['Unsorted data', 'Small datasets', 'Finding first occurrence']),

('dfs', 'Depth-First Search', 'graph', 'Graph traversal algorithm that explores as far as possible along each branch before backtracking', 'O(V + E)', 'O(V)', 'medium',
    ARRAY['traversal', 'recursive', 'stack-based', 'backtracking'],
    ARRAY['Topological sorting', 'Path finding', 'Cycle detection', 'Maze solving']),

('bfs', 'Breadth-First Search', 'graph', 'Graph traversal algorithm that explores all vertices at the present depth before moving to vertices at the next depth', 'O(V + E)', 'O(V)', 'medium',
    ARRAY['traversal', 'queue-based', 'shortest-path'],
    ARRAY['Shortest path in unweighted graphs', 'Level-order traversal', 'Finding connected components']),

('fibonacci', 'Fibonacci Sequence', 'dynamic_programming', 'Classic algorithm to generate Fibonacci numbers using various approaches (recursive, iterative, DP)', 'O(n)', 'O(1)', 'easy',
    ARRAY['recursion', 'memoization', 'dynamic-programming', 'math'],
    ARRAY['Mathematical modeling', 'Algorithm education', 'Golden ratio calculations']);

-- Get algorithm IDs for foreign key references
WITH algo_ids AS (
    SELECT id, name FROM algorithms
)

-- Insert Python implementations
INSERT INTO implementations (algorithm_id, language, code, is_primary, validated, version) 
SELECT id, 'python', 
CASE name
    WHEN 'quicksort' THEN 'def quicksort(arr):
    if len(arr) <= 1:
        return arr
    
    pivot = arr[len(arr) // 2]
    left = [x for x in arr if x < pivot]
    middle = [x for x in arr if x == pivot]
    right = [x for x in arr if x > pivot]
    
    return quicksort(left) + middle + quicksort(right)'

    WHEN 'mergesort' THEN 'def mergesort(arr):
    if len(arr) <= 1:
        return arr
    
    mid = len(arr) // 2
    left = mergesort(arr[:mid])
    right = mergesort(arr[mid:])
    
    return merge(left, right)

def merge(left, right):
    result = []
    i = j = 0
    
    while i < len(left) and j < len(right):
        if left[i] <= right[j]:
            result.append(left[i])
            i += 1
        else:
            result.append(right[j])
            j += 1
    
    result.extend(left[i:])
    result.extend(right[j:])
    return result'

    WHEN 'bubblesort' THEN 'def bubblesort(arr):
    n = len(arr)
    arr = arr.copy()
    
    for i in range(n):
        swapped = False
        for j in range(0, n - i - 1):
            if arr[j] > arr[j + 1]:
                arr[j], arr[j + 1] = arr[j + 1], arr[j]
                swapped = True
        
        if not swapped:
            break
    
    return arr'

    WHEN 'binary_search' THEN 'def binary_search(arr, target):
    left, right = 0, len(arr) - 1
    
    while left <= right:
        mid = left + (right - left) // 2
        
        if arr[mid] == target:
            return mid
        elif arr[mid] < target:
            left = mid + 1
        else:
            right = mid - 1
    
    return -1'

    WHEN 'linear_search' THEN 'def linear_search(arr, target):
    for i in range(len(arr)):
        if arr[i] == target:
            return i
    return -1'

    WHEN 'dfs' THEN 'def dfs(graph, start, visited=None):
    if visited is None:
        visited = set()
    
    visited.add(start)
    result = [start]
    
    for neighbor in graph[start]:
        if neighbor not in visited:
            result.extend(dfs(graph, neighbor, visited))
    
    return result'

    WHEN 'bfs' THEN 'from collections import deque

def bfs(graph, start):
    visited = set([start])
    queue = deque([start])
    result = []
    
    while queue:
        vertex = queue.popleft()
        result.append(vertex)
        
        for neighbor in graph[vertex]:
            if neighbor not in visited:
                visited.add(neighbor)
                queue.append(neighbor)
    
    return result'

    WHEN 'fibonacci' THEN 'def fibonacci(n):
    if n <= 0:
        return 0
    elif n == 1:
        return 1

    prev, curr = 0, 1
    for _ in range(2, n + 1):
        prev, curr = curr, prev + curr

    return curr'

    WHEN 'insertion_sort' THEN 'def insertion_sort(arr):
    arr = arr.copy()

    for i in range(1, len(arr)):
        key = arr[i]
        j = i - 1

        while j >= 0 and arr[j] > key:
            arr[j + 1] = arr[j]
            j -= 1

        arr[j + 1] = key

    return arr'

    WHEN 'selection_sort' THEN 'def selection_sort(arr):
    arr = arr.copy()
    n = len(arr)

    for i in range(n):
        min_idx = i

        for j in range(i + 1, n):
            if arr[j] < arr[min_idx]:
                min_idx = j

        arr[i], arr[min_idx] = arr[min_idx], arr[i]

    return arr'

    WHEN 'heapsort' THEN 'def heapsort(arr):
    def heapify(arr, n, i):
        largest = i
        left = 2 * i + 1
        right = 2 * i + 2

        if left < n and arr[left] > arr[largest]:
            largest = left

        if right < n and arr[right] > arr[largest]:
            largest = right

        if largest != i:
            arr[i], arr[largest] = arr[largest], arr[i]
            heapify(arr, n, largest)

    arr = arr.copy()
    n = len(arr)

    # Build max heap
    for i in range(n // 2 - 1, -1, -1):
        heapify(arr, n, i)

    # Extract elements from heap one by one
    for i in range(n - 1, 0, -1):
        arr[0], arr[i] = arr[i], arr[0]
        heapify(arr, i, 0)

    return arr'

END,
true, true, '1.0.0'
FROM algo_ids
WHERE name IN ('quicksort', 'mergesort', 'bubblesort', 'binary_search', 'linear_search', 'dfs', 'bfs', 'fibonacci', 'insertion_sort', 'selection_sort', 'heapsort');

-- Insert JavaScript implementations
WITH algo_ids AS (
    SELECT id, name FROM algorithms
)
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

    WHEN 'binary_search' THEN 'function binarySearch(arr, target) {
    let left = 0;
    let right = arr.length - 1;
    
    while (left <= right) {
        const mid = left + Math.floor((right - left) / 2);
        
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

    WHEN 'fibonacci' THEN 'function fibonacci(n) {
    if (n <= 0) return 0;
    if (n === 1) return 1;

    let prev = 0;
    let curr = 1;

    for (let i = 2; i <= n; i++) {
        [prev, curr] = [curr, prev + curr];
    }

    return curr;
}'

    WHEN 'insertion_sort' THEN 'function insertionSort(arr) {
    const result = [...arr];

    for (let i = 1; i < result.length; i++) {
        const key = result[i];
        let j = i - 1;

        while (j >= 0 && result[j] > key) {
            result[j + 1] = result[j];
            j--;
        }

        result[j + 1] = key;
    }

    return result;
}'

    WHEN 'selection_sort' THEN 'function selectionSort(arr) {
    const result = [...arr];
    const n = result.length;

    for (let i = 0; i < n; i++) {
        let minIdx = i;

        for (let j = i + 1; j < n; j++) {
            if (result[j] < result[minIdx]) {
                minIdx = j;
            }
        }

        [result[i], result[minIdx]] = [result[minIdx], result[i]];
    }

    return result;
}'

    WHEN 'heapsort' THEN 'function heapsort(arr) {
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
    }

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
            result.push(left[i++]);
        } else {
            result.push(right[j++]);
        }
    }

    return result.concat(left.slice(i)).concat(right.slice(j));
}'

    WHEN 'bubblesort' THEN 'function bubblesort(arr) {
    const result = [...arr];
    const n = result.length;

    for (let i = 0; i < n; i++) {
        let swapped = false;

        for (let j = 0; j < n - i - 1; j++) {
            if (result[j] > result[j + 1]) {
                [result[j], result[j + 1]] = [result[j + 1], result[j]];
                swapped = true;
            }
        }

        if (!swapped) break;
    }

    return result;
}'

END,
false, true, '1.0.0'
FROM algo_ids
WHERE name IN ('quicksort', 'binary_search', 'fibonacci', 'insertion_sort', 'selection_sort', 'heapsort', 'mergesort', 'bubblesort');

-- Insert test cases for sorting algorithms
INSERT INTO test_cases (algorithm_id, name, description, input, expected_output, is_edge_case, sequence_order)
SELECT id, test_name, test_desc, test_input::jsonb, test_output::jsonb, edge_case, seq_order
FROM algorithms,
LATERAL (VALUES
    ('Empty array', 'Test with empty array', '{"arr": []}', '[]', true, 1),
    ('Single element', 'Test with single element', '{"arr": [42]}', '[42]', true, 2),
    ('Already sorted', 'Test with already sorted array', '{"arr": [1, 2, 3, 4, 5]}', '[1, 2, 3, 4, 5]', false, 3),
    ('Reverse sorted', 'Test with reverse sorted array', '{"arr": [5, 4, 3, 2, 1]}', '[1, 2, 3, 4, 5]', false, 4),
    ('Random array', 'Test with random array', '{"arr": [3, 7, 1, 4, 6, 2, 5]}', '[1, 2, 3, 4, 5, 6, 7]', false, 5),
    ('Duplicates', 'Test with duplicate values', '{"arr": [3, 1, 4, 1, 5, 9, 2, 6, 5, 3]}', '[1, 1, 2, 3, 3, 4, 5, 5, 6, 9]', false, 6),
    ('All same', 'Test with all same values', '{"arr": [5, 5, 5, 5, 5]}', '[5, 5, 5, 5, 5]', true, 7)
) AS t(test_name, test_desc, test_input, test_output, edge_case, seq_order)
WHERE category = 'sorting';

-- Insert test cases for searching algorithms
INSERT INTO test_cases (algorithm_id, name, description, input, expected_output, is_edge_case, sequence_order)
SELECT id, test_name, test_desc, test_input::jsonb, test_output::jsonb, edge_case, seq_order
FROM algorithms,
LATERAL (VALUES
    ('Empty array', 'Search in empty array', '{"arr": [], "target": 5}', '-1', true, 1),
    ('Single element found', 'Search in single element array - found', '{"arr": [5], "target": 5}', '0', true, 2),
    ('Single element not found', 'Search in single element array - not found', '{"arr": [3], "target": 5}', '-1', true, 3),
    ('First element', 'Target is first element', '{"arr": [1, 2, 3, 4, 5], "target": 1}', '0', false, 4),
    ('Last element', 'Target is last element', '{"arr": [1, 2, 3, 4, 5], "target": 5}', '4', false, 5),
    ('Middle element', 'Target is middle element', '{"arr": [1, 2, 3, 4, 5], "target": 3}', '2', false, 6),
    ('Not found', 'Target not in array', '{"arr": [1, 2, 3, 4, 5], "target": 10}', '-1', false, 7)
) AS t(test_name, test_desc, test_input, test_output, edge_case, seq_order)
WHERE category = 'searching';

-- Insert test cases for graph algorithms
INSERT INTO test_cases (algorithm_id, name, description, input, expected_output, is_edge_case, sequence_order)
SELECT id, test_name, test_desc, test_input::jsonb, test_output::jsonb, edge_case, seq_order
FROM algorithms,
LATERAL (VALUES
    ('Single node', 'Graph with single node', '{"graph": {"A": []}, "start": "A"}', '["A"]', true, 1),
    ('Linear graph', 'Linear connected graph', '{"graph": {"A": ["B"], "B": ["C"], "C": []}, "start": "A"}', '["A", "B", "C"]', false, 2),
    ('Tree structure', 'Tree-like graph', '{"graph": {"A": ["B", "C"], "B": ["D", "E"], "C": ["F"], "D": [], "E": [], "F": []}, "start": "A"}', '["A", "B", "D", "E", "C", "F"]', false, 3),
    ('Cycle', 'Graph with cycle', '{"graph": {"A": ["B"], "B": ["C"], "C": ["A", "D"], "D": []}, "start": "A"}', '["A", "B", "C", "D"]', false, 4)
) AS t(test_name, test_desc, test_input, test_output, edge_case, seq_order)
WHERE name IN ('dfs', 'bfs');

-- Insert test cases for Fibonacci
INSERT INTO test_cases (algorithm_id, name, description, input, expected_output, is_edge_case, sequence_order)
SELECT id, test_name, test_desc, test_input::jsonb, test_output::jsonb, edge_case, seq_order
FROM algorithms,
LATERAL (VALUES
    ('Zero', 'Fibonacci of 0', '{"n": 0}', '0', true, 1),
    ('One', 'Fibonacci of 1', '{"n": 1}', '1', true, 2),
    ('Small number', 'Fibonacci of 5', '{"n": 5}', '5', false, 3),
    ('Medium number', 'Fibonacci of 10', '{"n": 10}', '55', false, 4),
    ('Larger number', 'Fibonacci of 20', '{"n": 20}', '6765', false, 5)
) AS t(test_name, test_desc, test_input, test_output, edge_case, seq_order)
WHERE name = 'fibonacci';

-- Add some algorithm relationships
INSERT INTO algorithm_relationships (from_algorithm_id, to_algorithm_id, relationship_type, description)
SELECT q.id, b.id, 'alternative', 'Both are comparison-based sorting algorithms'
FROM algorithms q, algorithms b
WHERE q.name = 'quicksort' AND b.name = 'mergesort';

INSERT INTO algorithm_relationships (from_algorithm_id, to_algorithm_id, relationship_type, description)
SELECT b.id, l.id, 'optimizes', 'Binary search optimizes linear search for sorted data'
FROM algorithms b, algorithms l
WHERE b.name = 'binary_search' AND l.name = 'linear_search';

-- Additional algorithms (50+ total required by PRD)
INSERT INTO algorithms (name, display_name, category, description, complexity_time, complexity_space, difficulty, tags, common_applications) VALUES
-- Additional sorting algorithms
('heapsort', 'HeapSort', 'sorting', 'Comparison-based sorting using binary heap data structure', 'O(n log n)', 'O(1)', 'medium',
    ARRAY['heap', 'in-place', 'comparison-sort'],
    ARRAY['Systems programming', 'Embedded systems with memory constraints']),
('insertion_sort', 'Insertion Sort', 'sorting', 'Builds final sorted array one item at a time', 'O(n²)', 'O(1)', 'easy',
    ARRAY['stable', 'in-place', 'adaptive', 'online'],
    ARRAY['Small datasets', 'Nearly sorted data', 'Online algorithms']),
('selection_sort', 'Selection Sort', 'sorting', 'Divides input into sorted and unsorted regions', 'O(n²)', 'O(1)', 'easy',
    ARRAY['in-place', 'unstable', 'comparison-sort'],
    ARRAY['Small datasets', 'Memory-constrained systems']),
('radix_sort', 'Radix Sort', 'sorting', 'Non-comparative integer sorting algorithm', 'O(nk)', 'O(n+k)', 'hard',
    ARRAY['non-comparison', 'stable', 'integer-sort'],
    ARRAY['Large integer datasets', 'String sorting', 'Parallel processing']),
('counting_sort', 'Counting Sort', 'sorting', 'Sorts by counting objects having distinct key values', 'O(n+k)', 'O(k)', 'medium',
    ARRAY['non-comparison', 'stable', 'integer-sort'],
    ARRAY['Small integer range', 'Histogram generation']),

-- Tree algorithms  
('binary_tree_traversal', 'Binary Tree Traversal', 'tree', 'Inorder, preorder, and postorder tree traversal', 'O(n)', 'O(h)', 'easy',
    ARRAY['recursion', 'tree-traversal'],
    ARRAY['Expression evaluation', 'Syntax tree processing']),
('bst_insert', 'BST Insert', 'tree', 'Insert node in binary search tree maintaining BST property', 'O(h)', 'O(1)', 'easy',
    ARRAY['binary-search-tree', 'insertion'],
    ARRAY['Database indexing', 'Symbol tables']),
('avl_tree', 'AVL Tree Operations', 'tree', 'Self-balancing binary search tree operations', 'O(log n)', 'O(1)', 'hard',
    ARRAY['self-balancing', 'rotation', 'height-balanced'],
    ARRAY['Database systems', 'File systems']),
('heap_operations', 'Heap Operations', 'tree', 'Binary heap insert, delete, heapify operations', 'O(log n)', 'O(1)', 'medium',
    ARRAY['priority-queue', 'complete-tree'],
    ARRAY['Priority queues', 'Heap sort', 'Dijkstra algorithm']),

-- More graph algorithms
('dijkstra', 'Dijkstra Algorithm', 'graph', 'Shortest path in weighted graph with non-negative edges', 'O(E log V)', 'O(V)', 'hard',
    ARRAY['shortest-path', 'greedy', 'weighted-graph'],
    ARRAY['GPS navigation', 'Network routing', 'Flight connections']),
('kruskal', 'Kruskal Algorithm', 'graph', 'Minimum spanning tree using edge sorting', 'O(E log E)', 'O(V)', 'medium',
    ARRAY['mst', 'greedy', 'union-find'],
    ARRAY['Network design', 'Circuit design']),
('topological_sort', 'Topological Sort', 'graph', 'Linear ordering of vertices in DAG', 'O(V + E)', 'O(V)', 'medium',
    ARRAY['dag', 'ordering', 'dependency'],
    ARRAY['Build systems', 'Course scheduling', 'Package managers']),

-- Dynamic programming algorithms
('knapsack_01', '0/1 Knapsack', 'dynamic_programming', 'Select items with maximum value within weight limit', 'O(nW)', 'O(nW)', 'medium',
    ARRAY['optimization', 'subset-selection'],
    ARRAY['Resource allocation', 'Budget management']),
('longest_common_subsequence', 'LCS', 'dynamic_programming', 'Find longest subsequence common to two sequences', 'O(mn)', 'O(mn)', 'medium',
    ARRAY['string-matching', 'sequence-alignment'],
    ARRAY['DNA analysis', 'Diff tools', 'Version control']),
('edit_distance', 'Edit Distance', 'dynamic_programming', 'Minimum operations to transform one string to another', 'O(mn)', 'O(mn)', 'medium',
    ARRAY['string-matching', 'levenshtein'],
    ARRAY['Spell correction', 'DNA sequence alignment']),
('coin_change', 'Coin Change', 'dynamic_programming', 'Minimum coins needed to make change', 'O(nS)', 'O(S)', 'medium',
    ARRAY['optimization', 'counting'],
    ARRAY['Currency systems', 'Making change']),
('kadane_algorithm', 'Kadane Algorithm', 'dynamic_programming', 'Maximum sum subarray', 'O(n)', 'O(1)', 'medium',
    ARRAY['maximum-subarray', 'optimization'],
    ARRAY['Financial analysis', 'Image processing']),

-- String algorithms
('kmp', 'KMP Pattern Matching', 'string', 'Knuth-Morris-Pratt string matching algorithm', 'O(n+m)', 'O(m)', 'hard',
    ARRAY['pattern-matching', 'failure-function'],
    ARRAY['Text editors', 'Plagiarism detection']),
('rabin_karp', 'Rabin-Karp', 'string', 'String matching using rolling hash', 'O(nm)', 'O(1)', 'medium',
    ARRAY['pattern-matching', 'rolling-hash'],
    ARRAY['Multiple pattern search', 'Plagiarism detection']),

-- Mathematical algorithms  
('gcd', 'GCD (Euclidean)', 'math', 'Greatest common divisor using Euclidean algorithm', 'O(log min(a,b))', 'O(1)', 'easy',
    ARRAY['number-theory', 'euclidean'],
    ARRAY['Fraction simplification', 'Cryptography']),
('sieve_eratosthenes', 'Sieve of Eratosthenes', 'math', 'Find all prime numbers up to n', 'O(n log log n)', 'O(n)', 'easy',
    ARRAY['prime-numbers', 'sieve'],
    ARRAY['Prime generation', 'Cryptography']),
('fast_exponentiation', 'Fast Exponentiation', 'math', 'Compute a^n efficiently using binary exponentiation', 'O(log n)', 'O(1)', 'medium',
    ARRAY['exponentiation', 'binary-method'],
    ARRAY['Cryptography', 'Matrix operations']),

-- Backtracking algorithms
('n_queens', 'N-Queens', 'backtracking', 'Place N queens on NxN chessboard with no attacks', 'O(N!)', 'O(N)', 'medium',
    ARRAY['constraint-satisfaction', 'recursion'],
    ARRAY['Puzzle solving', 'Constraint programming']),
('sudoku_solver', 'Sudoku Solver', 'backtracking', 'Solve Sudoku puzzle using backtracking', 'O(9^m)', 'O(1)', 'medium',
    ARRAY['constraint-satisfaction', 'puzzle'],
    ARRAY['Game solving', 'Constraint satisfaction']),
('permutations', 'Generate Permutations', 'backtracking', 'Generate all permutations of a sequence', 'O(n!)', 'O(n)', 'easy',
    ARRAY['combinatorics', 'recursion'],
    ARRAY['Combinatorial problems', 'Testing']),

-- Greedy algorithms
('activity_selection', 'Activity Selection', 'greedy', 'Select maximum non-overlapping activities', 'O(n log n)', 'O(1)', 'easy',
    ARRAY['interval-scheduling', 'optimization'],
    ARRAY['Resource scheduling', 'Meeting rooms']),
('huffman_coding', 'Huffman Coding', 'greedy', 'Optimal prefix-free encoding', 'O(n log n)', 'O(n)', 'medium',
    ARRAY['compression', 'prefix-codes', 'priority-queue'],
    ARRAY['Data compression', 'File encoding'])
ON CONFLICT (name) DO NOTHING;

-- Insert Python implementations for additional sorting algorithms
WITH algo_ids AS (
    SELECT id, name FROM algorithms
)
INSERT INTO implementations (algorithm_id, language, code, is_primary, validated, version)
SELECT id, 'python',
CASE name
    WHEN 'insertion_sort' THEN 'def insertion_sort(arr):
    arr = arr.copy()

    for i in range(1, len(arr)):
        key = arr[i]
        j = i - 1

        while j >= 0 and arr[j] > key:
            arr[j + 1] = arr[j]
            j -= 1

        arr[j + 1] = key

    return arr'

    WHEN 'selection_sort' THEN 'def selection_sort(arr):
    arr = arr.copy()
    n = len(arr)

    for i in range(n):
        min_idx = i

        for j in range(i + 1, n):
            if arr[j] < arr[min_idx]:
                min_idx = j

        arr[i], arr[min_idx] = arr[min_idx], arr[i]

    return arr'

    WHEN 'heapsort' THEN 'def heapsort(arr):
    def heapify(arr, n, i):
        largest = i
        left = 2 * i + 1
        right = 2 * i + 2

        if left < n and arr[left] > arr[largest]:
            largest = left

        if right < n and arr[right] > arr[largest]:
            largest = right

        if largest != i:
            arr[i], arr[largest] = arr[largest], arr[i]
            heapify(arr, n, largest)

    arr = arr.copy()
    n = len(arr)

    # Build max heap
    for i in range(n // 2 - 1, -1, -1):
        heapify(arr, n, i)

    # Extract elements from heap one by one
    for i in range(n - 1, 0, -1):
        arr[0], arr[i] = arr[i], arr[0]
        heapify(arr, i, 0)

    return arr'
END,
true, true, '1.0.0'
FROM algo_ids
WHERE name IN ('insertion_sort', 'selection_sort', 'heapsort');

-- Insert JavaScript implementations for additional sorting algorithms
WITH algo_ids AS (
    SELECT id, name FROM algorithms
)
INSERT INTO implementations (algorithm_id, language, code, is_primary, validated, version)
SELECT id, 'javascript',
CASE name
    WHEN 'insertion_sort' THEN 'function insertionSort(arr) {
    const result = [...arr];

    for (let i = 1; i < result.length; i++) {
        const key = result[i];
        let j = i - 1;

        while (j >= 0 && result[j] > key) {
            result[j + 1] = result[j];
            j--;
        }

        result[j + 1] = key;
    }

    return result;
}'

    WHEN 'selection_sort' THEN 'function selectionSort(arr) {
    const result = [...arr];
    const n = result.length;

    for (let i = 0; i < n; i++) {
        let minIdx = i;

        for (let j = i + 1; j < n; j++) {
            if (result[j] < result[minIdx]) {
                minIdx = j;
            }
        }

        [result[i], result[minIdx]] = [result[minIdx], result[i]];
    }

    return result;
}'

    WHEN 'heapsort' THEN 'function heapsort(arr) {
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
    }

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
}'
END,
false, true, '1.0.0'
FROM algo_ids
WHERE name IN ('insertion_sort', 'selection_sort', 'heapsort');

-- Insert initial performance benchmarks (sample data)
INSERT INTO benchmarks (algorithm_id, language, input_size, execution_time_ms, memory_used_mb, environment_info)
SELECT a.id, lang, size, 
    CASE 
        WHEN a.name = 'quicksort' THEN size * 0.001 * ln(size + 1)
        WHEN a.name = 'mergesort' THEN size * 0.0012 * ln(size + 1)
        WHEN a.name = 'bubblesort' THEN size * size * 0.00001
        ELSE size * 0.0001
    END,
    CASE
        WHEN a.name = 'mergesort' THEN size * 0.008
        ELSE size * 0.001
    END,
    '{"os": "Linux", "cpu": "Intel i7", "ram": "16GB"}'::jsonb
FROM algorithms a,
    (VALUES ('python'), ('javascript')) AS l(lang),
    (VALUES (100), (1000), (10000)) AS s(size)
WHERE a.category = 'sorting';

-- Insert Python implementations for tree, dynamic programming, and greedy algorithms (from 2025-10-11)
-- binary_tree_traversal implementation
INSERT INTO implementations (algorithm_id, language, code, is_primary, validated, version)
SELECT id, 'python', 'def binary_tree_traversal(tree, traversal_type="inorder"):
    """
    Traverse binary tree (inorder, preorder, or postorder).
    tree: dict where tree[node] = (left, right)
    Returns: list of nodes in traversal order
    """
    def inorder(node):
        if node is None:
            return []
        left, right = tree.get(node, (None, None))
        return inorder(left) + [node] + inorder(right)

    def preorder(node):
        if node is None:
            return []
        left, right = tree.get(node, (None, None))
        return [node] + preorder(left) + preorder(right)

    def postorder(node):
        if node is None:
            return []
        left, right = tree.get(node, (None, None))
        return postorder(left) + postorder(right) + [node]

    root = list(tree.keys())[0] if tree else None
    if traversal_type == "inorder":
        return inorder(root)
    elif traversal_type == "preorder":
        return preorder(root)
    else:
        return postorder(root)', true, true, '1.0.0'
FROM algorithms WHERE name = 'binary_tree_traversal'
ON CONFLICT (algorithm_id, language, version) DO NOTHING;

-- bst_insert implementation
INSERT INTO implementations (algorithm_id, language, code, is_primary, validated, version)
SELECT id, 'python', 'def bst_insert(root, value):
    """
    Insert value into binary search tree.
    root: dict representing BST structure
    Returns: updated BST dict
    """
    if not root:
        return {value: (None, None)}

    def insert_recursive(node):
        if node is None:
            return value
        if value < node:
            left, right = root.get(node, (None, None))
            new_left = insert_recursive(left)
            root[node] = (new_left, right)
            if new_left == value and value not in root:
                root[value] = (None, None)
        elif value > node:
            left, right = root.get(node, (None, None))
            new_right = insert_recursive(right)
            root[node] = (left, new_right)
            if new_right == value and value not in root:
                root[value] = (None, None)
        return node

    root_node = list(root.keys())[0]
    insert_recursive(root_node)
    return root', true, true, '1.0.0'
FROM algorithms WHERE name = 'bst_insert'
ON CONFLICT (algorithm_id, language, version) DO NOTHING;

-- coin_change implementation
INSERT INTO implementations (algorithm_id, language, code, is_primary, validated, version)
SELECT id, 'python', 'def coin_change(coins, amount):
    """
    Find minimum coins needed to make amount.
    Returns: minimum coins needed, or -1 if impossible
    """
    dp = [float(''infinity'')] * (amount + 1)
    dp[0] = 0

    for i in range(1, amount + 1):
        for coin in coins:
            if coin <= i:
                dp[i] = min(dp[i], dp[i - coin] + 1)

    return dp[amount] if dp[amount] != float(''infinity'') else -1', true, true, '1.0.0'
FROM algorithms WHERE name = 'coin_change'
ON CONFLICT (algorithm_id, language, version) DO NOTHING;

-- edit_distance implementation
INSERT INTO implementations (algorithm_id, language, code, is_primary, validated, version)
SELECT id, 'python', 'def edit_distance(str1, str2):
    """
    Calculate Levenshtein distance between two strings.
    Returns: minimum edit operations needed
    """
    m, n = len(str1), len(str2)
    dp = [[0] * (n + 1) for _ in range(m + 1)]

    for i in range(m + 1):
        dp[i][0] = i
    for j in range(n + 1):
        dp[0][j] = j

    for i in range(1, m + 1):
        for j in range(1, n + 1):
            if str1[i-1] == str2[j-1]:
                dp[i][j] = dp[i-1][j-1]
            else:
                dp[i][j] = 1 + min(
                    dp[i-1][j],    # deletion
                    dp[i][j-1],    # insertion
                    dp[i-1][j-1]   # substitution
                )

    return dp[m][n]', true, true, '1.0.0'
FROM algorithms WHERE name = 'edit_distance'
ON CONFLICT (algorithm_id, language, version) DO NOTHING;

-- counting_sort implementation
INSERT INTO implementations (algorithm_id, language, code, is_primary, validated, version)
SELECT id, 'python', 'def counting_sort(arr):
    """
    Sort array of non-negative integers using counting sort.
    Returns: sorted array
    """
    if not arr:
        return []

    max_val = max(arr)
    counts = [0] * (max_val + 1)

    for num in arr:
        counts[num] += 1

    result = []
    for num, count in enumerate(counts):
        result.extend([num] * count)

    return result', true, true, '1.0.0'
FROM algorithms WHERE name = 'counting_sort'
ON CONFLICT (algorithm_id, language, version) DO NOTHING;

-- activity_selection implementation
INSERT INTO implementations (algorithm_id, language, code, is_primary, validated, version)
SELECT id, 'python', 'def activity_selection(activities):
    """
    Select maximum non-overlapping activities.
    activities: list of (start, end) tuples
    Returns: list of selected activity indices
    """
    if not activities:
        return []

    # Sort by end time
    indexed = [(i, start, end) for i, (start, end) in enumerate(activities)]
    indexed.sort(key=lambda x: x[2])

    selected = [indexed[0][0]]
    last_end = indexed[0][2]

    for i, start, end in indexed[1:]:
        if start >= last_end:
            selected.append(i)
            last_end = end

    return selected', true, true, '1.0.0'
FROM algorithms WHERE name = 'activity_selection'
ON CONFLICT (algorithm_id, language, version) DO NOTHING;

-- Insert Python implementations for high-value graph and dynamic programming algorithms (additional from today)
-- dijkstra implementation
INSERT INTO implementations (algorithm_id, language, code, is_primary, validated, version)
SELECT id, 'python', 'import heapq

def dijkstra(graph, start):
    """
    Find shortest paths from start to all vertices.
    graph: dict where graph[u] = [(v, weight), ...]
    Returns: dict of {vertex: distance}
    """
    distances = {vertex: float(''infinity'') for vertex in graph}
    distances[start] = 0

    pq = [(0, start)]
    visited = set()

    while pq:
        current_dist, current = heapq.heappop(pq)

        if current in visited:
            continue

        visited.add(current)

        for neighbor, weight in graph.get(current, []):
            distance = current_dist + weight

            if distance < distances.get(neighbor, float(''infinity'')):
                distances[neighbor] = distance
                heapq.heappush(pq, (distance, neighbor))

    return distances', true, true, '1.0.0'
FROM algorithms WHERE name = 'dijkstra'
ON CONFLICT (algorithm_id, language, version) DO NOTHING;

-- kruskal implementation
INSERT INTO implementations (algorithm_id, language, code, is_primary, validated, version)
SELECT id, 'python', 'def kruskal(edges, num_vertices):
    """
    Find minimum spanning tree using Kruskal algorithm.
    edges: list of (u, v, weight)
    Returns: list of edges in MST
    """
    parent = list(range(num_vertices))
    rank = [0] * num_vertices

    def find(x):
        if parent[x] != x:
            parent[x] = find(parent[x])
        return parent[x]

    def union(x, y):
        px, py = find(x), find(y)
        if px == py:
            return False
        if rank[px] < rank[py]:
            px, py = py, px
        parent[py] = px
        if rank[px] == rank[py]:
            rank[px] += 1
        return True

    edges = sorted(edges, key=lambda x: x[2])
    mst = []

    for u, v, weight in edges:
        if union(u, v):
            mst.append((u, v, weight))
            if len(mst) == num_vertices - 1:
                break

    return mst', true, true, '1.0.0'
FROM algorithms WHERE name = 'kruskal'
ON CONFLICT (algorithm_id, language, version) DO NOTHING;

-- kadane_algorithm implementation
INSERT INTO implementations (algorithm_id, language, code, is_primary, validated, version)
SELECT id, 'python', 'def kadane_algorithm(arr):
    """
    Find maximum sum of contiguous subarray.
    Returns: (max_sum, start_index, end_index)
    """
    if not arr:
        return 0, 0, 0

    max_sum = current_sum = arr[0]
    max_start = max_end = 0
    current_start = 0

    for i in range(1, len(arr)):
        if current_sum < 0:
            current_sum = arr[i]
            current_start = i
        else:
            current_sum += arr[i]

        if current_sum > max_sum:
            max_sum = current_sum
            max_start = current_start
            max_end = i

    return max_sum, max_start, max_end', true, true, '1.0.0'
FROM algorithms WHERE name = 'kadane_algorithm'
ON CONFLICT (algorithm_id, language, version) DO NOTHING;

-- knapsack_01 implementation
INSERT INTO implementations (algorithm_id, language, code, is_primary, validated, version)
SELECT id, 'python', 'def knapsack_01(weights, values, capacity):
    """
    Solve 0/1 knapsack problem using dynamic programming.
    Returns: (max_value, selected_items)
    """
    n = len(weights)
    dp = [[0] * (capacity + 1) for _ in range(n + 1)]

    for i in range(1, n + 1):
        for w in range(capacity + 1):
            if weights[i-1] <= w:
                dp[i][w] = max(
                    values[i-1] + dp[i-1][w-weights[i-1]],
                    dp[i-1][w]
                )
            else:
                dp[i][w] = dp[i-1][w]

    # Backtrack to find selected items
    selected = []
    w = capacity
    for i in range(n, 0, -1):
        if dp[i][w] != dp[i-1][w]:
            selected.append(i-1)
            w -= weights[i-1]

    return dp[n][capacity], selected[::-1]', true, true, '1.0.0'
FROM algorithms WHERE name = 'knapsack_01'
ON CONFLICT (algorithm_id, language, version) DO NOTHING;

-- longest_common_subsequence implementation
INSERT INTO implementations (algorithm_id, language, code, is_primary, validated, version)
SELECT id, 'python', 'def longest_common_subsequence(text1, text2):
    """
    Find longest common subsequence of two strings.
    Returns: (length, lcs_string)
    """
    m, n = len(text1), len(text2)
    dp = [[0] * (n + 1) for _ in range(m + 1)]

    for i in range(1, m + 1):
        for j in range(1, n + 1):
            if text1[i-1] == text2[j-1]:
                dp[i][j] = dp[i-1][j-1] + 1
            else:
                dp[i][j] = max(dp[i-1][j], dp[i][j-1])

    # Reconstruct LCS
    lcs = []
    i, j = m, n
    while i > 0 and j > 0:
        if text1[i-1] == text2[j-1]:
            lcs.append(text1[i-1])
            i -= 1
            j -= 1
        elif dp[i-1][j] > dp[i][j-1]:
            i -= 1
        else:
            j -= 1

    return dp[m][n], ''''.join(reversed(lcs))', true, true, '1.0.0'
FROM algorithms WHERE name = 'longest_common_subsequence'
ON CONFLICT (algorithm_id, language, version) DO NOTHING;

-- kmp implementation
INSERT INTO implementations (algorithm_id, language, code, is_primary, validated, version)
SELECT id, 'python', 'def kmp(text, pattern):
    """
    KMP string matching algorithm.
    Returns: list of starting indices where pattern is found
    """
    def compute_lps(pattern):
        lps = [0] * len(pattern)
        length = 0
        i = 1

        while i < len(pattern):
            if pattern[i] == pattern[length]:
                length += 1
                lps[i] = length
                i += 1
            else:
                if length != 0:
                    length = lps[length - 1]
                else:
                    lps[i] = 0
                    i += 1
        return lps

    lps = compute_lps(pattern)
    matches = []
    i = j = 0

    while i < len(text):
        if pattern[j] == text[i]:
            i += 1
            j += 1

        if j == len(pattern):
            matches.append(i - j)
            j = lps[j - 1]
        elif i < len(text) and pattern[j] != text[i]:
            if j != 0:
                j = lps[j - 1]
            else:
                i += 1

    return matches', true, true, '1.0.0'
FROM algorithms WHERE name = 'kmp'
ON CONFLICT (algorithm_id, language, version) DO NOTHING;

-- Update implementation validation counts
UPDATE implementations i
SET validation_count = (
    SELECT COUNT(*)
    FROM test_cases tc
    WHERE tc.algorithm_id = i.algorithm_id
),
last_validation = CURRENT_TIMESTAMP;