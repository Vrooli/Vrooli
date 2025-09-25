-- Extended seed data for Algorithm Library
-- Adds 50+ fundamental algorithms across multiple categories

-- Additional sorting algorithms
INSERT INTO algorithms (name, display_name, category, description, complexity_time, complexity_space, difficulty, tags, common_applications) VALUES
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

('bucket_sort', 'Bucket Sort', 'sorting', 'Distributes elements into buckets and sorts each bucket', 'O(n+k)', 'O(n)', 'medium',
    ARRAY['distribution-sort', 'stable'],
    ARRAY['Uniformly distributed data', 'Floating-point numbers']),

('shell_sort', 'Shell Sort', 'sorting', 'Generalization of insertion sort allowing exchanges of far items', 'O(n log² n)', 'O(1)', 'medium',
    ARRAY['in-place', 'unstable', 'gap-sequence'],
    ARRAY['Medium-sized datasets', 'Embedded systems']),

('tim_sort', 'TimSort', 'sorting', 'Hybrid stable sorting algorithm derived from merge sort and insertion sort', 'O(n log n)', 'O(n)', 'hard',
    ARRAY['hybrid', 'stable', 'adaptive'],
    ARRAY['Python built-in sort', 'Java Collections.sort', 'Real-world data']),

-- Tree algorithms
('binary_tree_traversal', 'Binary Tree Traversal', 'tree', 'Inorder, preorder, and postorder tree traversal', 'O(n)', 'O(h)', 'easy',
    ARRAY['recursion', 'tree-traversal'],
    ARRAY['Expression evaluation', 'Syntax tree processing']),

('bst_insert', 'BST Insert', 'tree', 'Insert node in binary search tree maintaining BST property', 'O(h)', 'O(1)', 'easy',
    ARRAY['binary-search-tree', 'insertion'],
    ARRAY['Database indexing', 'Symbol tables']),

('bst_delete', 'BST Delete', 'tree', 'Delete node from BST maintaining BST property', 'O(h)', 'O(1)', 'medium',
    ARRAY['binary-search-tree', 'deletion'],
    ARRAY['Database operations', 'Memory management']),

('avl_tree', 'AVL Tree Operations', 'tree', 'Self-balancing binary search tree operations', 'O(log n)', 'O(1)', 'hard',
    ARRAY['self-balancing', 'rotation', 'height-balanced'],
    ARRAY['Database systems', 'File systems']),

('red_black_tree', 'Red-Black Tree', 'tree', 'Self-balancing BST with color properties', 'O(log n)', 'O(1)', 'hard',
    ARRAY['self-balancing', 'colored-nodes'],
    ARRAY['Linux kernel', 'C++ STL map/set']),

('trie_operations', 'Trie Operations', 'tree', 'Prefix tree for string operations', 'O(m)', 'O(ALPHABET_SIZE * m)', 'medium',
    ARRAY['prefix-tree', 'string-matching'],
    ARRAY['Autocomplete', 'Spell checkers', 'IP routing']),

('segment_tree', 'Segment Tree', 'tree', 'Tree for storing intervals and range queries', 'O(log n)', 'O(n)', 'hard',
    ARRAY['range-query', 'interval-tree'],
    ARRAY['Range sum queries', 'Computational geometry']),

('heap_operations', 'Heap Operations', 'tree', 'Binary heap insert, delete, heapify operations', 'O(log n)', 'O(1)', 'medium',
    ARRAY['priority-queue', 'complete-tree'],
    ARRAY['Priority queues', 'Heap sort', 'Dijkstra algorithm']),

-- Graph algorithms
('dijkstra', 'Dijkstra Algorithm', 'graph', 'Shortest path in weighted graph with non-negative edges', 'O(E log V)', 'O(V)', 'hard',
    ARRAY['shortest-path', 'greedy', 'weighted-graph'],
    ARRAY['GPS navigation', 'Network routing', 'Flight connections']),

('bellman_ford', 'Bellman-Ford', 'graph', 'Shortest path handling negative edge weights', 'O(VE)', 'O(V)', 'hard',
    ARRAY['shortest-path', 'negative-weights', 'dynamic-programming'],
    ARRAY['Currency arbitrage', 'Network protocols']),

('floyd_warshall', 'Floyd-Warshall', 'graph', 'All-pairs shortest path algorithm', 'O(V³)', 'O(V²)', 'hard',
    ARRAY['all-pairs', 'dynamic-programming'],
    ARRAY['Network analysis', 'Transitive closure']),

('kruskal', 'Kruskal Algorithm', 'graph', 'Minimum spanning tree using edge sorting', 'O(E log E)', 'O(V)', 'medium',
    ARRAY['mst', 'greedy', 'union-find'],
    ARRAY['Network design', 'Circuit design']),

('prim', 'Prim Algorithm', 'graph', 'Minimum spanning tree using priority queue', 'O(E log V)', 'O(V)', 'medium',
    ARRAY['mst', 'greedy', 'priority-queue'],
    ARRAY['Network clustering', 'Maze generation']),

('topological_sort', 'Topological Sort', 'graph', 'Linear ordering of vertices in DAG', 'O(V + E)', 'O(V)', 'medium',
    ARRAY['dag', 'ordering', 'dependency'],
    ARRAY['Build systems', 'Course scheduling', 'Package managers']),

('tarjan_scc', 'Tarjan SCC', 'graph', 'Find strongly connected components', 'O(V + E)', 'O(V)', 'hard',
    ARRAY['scc', 'dfs-based', 'low-link'],
    ARRAY['Social networks', 'Web crawling']),

('graph_coloring', 'Graph Coloring', 'graph', 'Assign colors to vertices with no adjacent same colors', 'O(V^V)', 'O(V)', 'hard',
    ARRAY['np-complete', 'backtracking', 'chromatic-number'],
    ARRAY['Register allocation', 'Scheduling problems']),

('max_flow', 'Maximum Flow', 'graph', 'Ford-Fulkerson method for maximum flow', 'O(E * max_flow)', 'O(V²)', 'hard',
    ARRAY['network-flow', 'augmenting-path'],
    ARRAY['Network capacity', 'Bipartite matching']),

-- Dynamic Programming
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

('matrix_chain', 'Matrix Chain Multiplication', 'dynamic_programming', 'Optimal parenthesization of matrix multiplication', 'O(n³)', 'O(n²)', 'hard',
    ARRAY['optimization', 'parenthesization'],
    ARRAY['Compiler optimization', 'Database query optimization']),

('kadane_algorithm', 'Kadane Algorithm', 'dynamic_programming', 'Maximum sum subarray', 'O(n)', 'O(1)', 'medium',
    ARRAY['maximum-subarray', 'optimization'],
    ARRAY['Financial analysis', 'Image processing']),

('lis', 'Longest Increasing Subsequence', 'dynamic_programming', 'Find length of longest increasing subsequence', 'O(n log n)', 'O(n)', 'medium',
    ARRAY['subsequence', 'patience-sorting'],
    ARRAY['Box stacking', 'Patience card game']),

-- String algorithms
('kmp', 'KMP Pattern Matching', 'string', 'Knuth-Morris-Pratt string matching algorithm', 'O(n+m)', 'O(m)', 'hard',
    ARRAY['pattern-matching', 'failure-function'],
    ARRAY['Text editors', 'Plagiarism detection']),

('rabin_karp', 'Rabin-Karp', 'string', 'String matching using rolling hash', 'O(nm)', 'O(1)', 'medium',
    ARRAY['pattern-matching', 'rolling-hash'],
    ARRAY['Multiple pattern search', 'Plagiarism detection']),

('boyer_moore', 'Boyer-Moore', 'string', 'Efficient string searching algorithm', 'O(nm)', 'O(m)', 'hard',
    ARRAY['pattern-matching', 'bad-character', 'good-suffix'],
    ARRAY['Text search', 'Grep implementation']),

('z_algorithm', 'Z Algorithm', 'string', 'Linear time pattern matching algorithm', 'O(n)', 'O(n)', 'hard',
    ARRAY['pattern-matching', 'z-array'],
    ARRAY['String analysis', 'Pattern finding']),

('manacher', 'Manacher Algorithm', 'string', 'Find longest palindromic substring', 'O(n)', 'O(n)', 'hard',
    ARRAY['palindrome', 'linear-time'],
    ARRAY['Text processing', 'Bioinformatics']),

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

('matrix_multiplication', 'Matrix Multiplication', 'math', 'Standard matrix multiplication algorithm', 'O(n³)', 'O(n²)', 'easy',
    ARRAY['linear-algebra', 'matrix-operations'],
    ARRAY['Graphics', 'Machine learning', 'Scientific computing']),

('strassen_matrix', 'Strassen Matrix Multiplication', 'math', 'Faster matrix multiplication using divide and conquer', 'O(n^2.807)', 'O(n²)', 'hard',
    ARRAY['divide-conquer', 'matrix-operations'],
    ARRAY['Large matrix computations']),

('gaussian_elimination', 'Gaussian Elimination', 'math', 'Solve system of linear equations', 'O(n³)', 'O(n²)', 'medium',
    ARRAY['linear-algebra', 'equation-solving'],
    ARRAY['Circuit analysis', 'Computer graphics']),

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

('combinations', 'Generate Combinations', 'backtracking', 'Generate all combinations of k elements', 'O(C(n,k))', 'O(k)', 'easy',
    ARRAY['combinatorics', 'recursion'],
    ARRAY['Subset generation', 'Lottery systems']),

-- Greedy algorithms
('activity_selection', 'Activity Selection', 'greedy', 'Select maximum non-overlapping activities', 'O(n log n)', 'O(1)', 'easy',
    ARRAY['interval-scheduling', 'optimization'],
    ARRAY['Resource scheduling', 'Meeting rooms']),

('huffman_coding', 'Huffman Coding', 'greedy', 'Optimal prefix-free encoding', 'O(n log n)', 'O(n)', 'medium',
    ARRAY['compression', 'prefix-codes', 'priority-queue'],
    ARRAY['Data compression', 'File encoding']),

('job_scheduling', 'Job Scheduling', 'greedy', 'Minimize completion time or maximize profit', 'O(n log n)', 'O(1)', 'medium',
    ARRAY['scheduling', 'optimization'],
    ARRAY['Task scheduling', 'CPU scheduling'])

ON CONFLICT (name) DO NOTHING;

-- Count total algorithms
SELECT COUNT(*) as total_algorithms FROM algorithms;