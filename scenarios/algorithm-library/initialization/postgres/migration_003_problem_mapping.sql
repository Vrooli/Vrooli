-- Migration 003: Add problem mapping for LeetCode/HackerRank
-- This enables mapping algorithms to common coding challenge problems

-- Create problem mapping table
CREATE TABLE IF NOT EXISTS problem_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    algorithm_id UUID REFERENCES algorithms(id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL, -- 'leetcode', 'hackerrank', 'codeforces', etc.
    problem_id VARCHAR(100) NOT NULL, -- Platform-specific problem ID
    problem_name VARCHAR(255) NOT NULL,
    problem_url TEXT,
    difficulty VARCHAR(20), -- 'easy', 'medium', 'hard'
    topics TEXT[], -- Array of topics/tags from the platform
    notes TEXT, -- Additional notes about how the algorithm applies
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(platform, problem_id)
);

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_problem_mappings_algorithm_id ON problem_mappings(algorithm_id);
CREATE INDEX IF NOT EXISTS idx_problem_mappings_platform ON problem_mappings(platform);
CREATE INDEX IF NOT EXISTS idx_problem_mappings_difficulty ON problem_mappings(difficulty);

-- Insert sample problem mappings
INSERT INTO problem_mappings (algorithm_id, platform, problem_id, problem_name, problem_url, difficulty, topics, notes) VALUES
-- QuickSort problems
((SELECT id FROM algorithms WHERE name = 'quicksort'), 'leetcode', '912', 'Sort an Array', 'https://leetcode.com/problems/sort-an-array/', 'medium', ARRAY['array', 'divide-and-conquer', 'sorting'], 'Direct application of quicksort algorithm'),
((SELECT id FROM algorithms WHERE name = 'quicksort'), 'hackerrank', 'quicksort1', 'Quicksort 1 - Partition', 'https://www.hackerrank.com/challenges/quicksort1/', 'easy', ARRAY['sorting'], 'Focuses on the partition step of quicksort'),

-- MergeSort problems
((SELECT id FROM algorithms WHERE name = 'mergesort'), 'leetcode', '148', 'Sort List', 'https://leetcode.com/problems/sort-list/', 'medium', ARRAY['linked-list', 'sorting', 'divide-and-conquer'], 'Merge sort on linked list'),
((SELECT id FROM algorithms WHERE name = 'mergesort'), 'leetcode', '88', 'Merge Sorted Array', 'https://leetcode.com/problems/merge-sorted-array/', 'easy', ARRAY['array', 'two-pointers', 'sorting'], 'Uses merge step of merge sort'),

-- Binary Search problems
((SELECT id FROM algorithms WHERE name = 'binarysearch'), 'leetcode', '704', 'Binary Search', 'https://leetcode.com/problems/binary-search/', 'easy', ARRAY['array', 'binary-search'], 'Classic binary search implementation'),
((SELECT id FROM algorithms WHERE name = 'binarysearch'), 'leetcode', '35', 'Search Insert Position', 'https://leetcode.com/problems/search-insert-position/', 'easy', ARRAY['array', 'binary-search'], 'Binary search variant'),
((SELECT id FROM algorithms WHERE name = 'binarysearch'), 'hackerrank', 'binary-search', 'Binary Search', 'https://www.hackerrank.com/challenges/binary-search/', 'easy', ARRAY['searching'], 'Standard binary search'),

-- DFS problems
((SELECT id FROM algorithms WHERE name = 'dfs'), 'leetcode', '200', 'Number of Islands', 'https://leetcode.com/problems/number-of-islands/', 'medium', ARRAY['array', 'dfs', 'bfs', 'union-find', 'matrix'], 'DFS to explore connected components'),
((SELECT id FROM algorithms WHERE name = 'dfs'), 'leetcode', '994', 'Rotting Oranges', 'https://leetcode.com/problems/rotting-oranges/', 'medium', ARRAY['array', 'bfs', 'matrix'], 'Can be solved with DFS or BFS'),
((SELECT id FROM algorithms WHERE name = 'dfs'), 'hackerrank', 'connected-cell-in-a-grid', 'Connected Cells in a Grid', 'https://www.hackerrank.com/challenges/connected-cell-in-a-grid/', 'medium', ARRAY['search', 'graph'], 'DFS to find connected regions'),

-- BFS problems
((SELECT id FROM algorithms WHERE name = 'bfs'), 'leetcode', '102', 'Binary Tree Level Order Traversal', 'https://leetcode.com/problems/binary-tree-level-order-traversal/', 'medium', ARRAY['tree', 'bfs', 'binary-tree'], 'Classic BFS on tree'),
((SELECT id FROM algorithms WHERE name = 'bfs'), 'leetcode', '127', 'Word Ladder', 'https://leetcode.com/problems/word-ladder/', 'hard', ARRAY['hash-table', 'string', 'bfs'], 'BFS for shortest transformation sequence'),

-- Dijkstra problems
((SELECT id FROM algorithms WHERE name = 'dijkstra'), 'leetcode', '743', 'Network Delay Time', 'https://leetcode.com/problems/network-delay-time/', 'medium', ARRAY['dfs', 'bfs', 'graph', 'heap', 'shortest-path'], 'Classic Dijkstra application'),
((SELECT id FROM algorithms WHERE name = 'dijkstra'), 'leetcode', '787', 'Cheapest Flights Within K Stops', 'https://leetcode.com/problems/cheapest-flights-within-k-stops/', 'medium', ARRAY['dynamic-programming', 'dfs', 'bfs', 'graph', 'heap', 'shortest-path'], 'Modified Dijkstra with stop limit'),

-- Dynamic Programming problems
((SELECT id FROM algorithms WHERE name = 'knapsack'), 'leetcode', '416', 'Partition Equal Subset Sum', 'https://leetcode.com/problems/partition-equal-subset-sum/', 'medium', ARRAY['array', 'dynamic-programming'], '0/1 Knapsack variant'),
((SELECT id FROM algorithms WHERE name = 'knapsack'), 'hackerrank', 'unbounded-knapsack', 'Knapsack', 'https://www.hackerrank.com/challenges/unbounded-knapsack/', 'medium', ARRAY['dynamic-programming'], 'Unbounded knapsack problem'),

-- Two Pointers problems
((SELECT id FROM algorithms WHERE name = 'two_pointers'), 'leetcode', '15', '3Sum', 'https://leetcode.com/problems/3sum/', 'medium', ARRAY['array', 'two-pointers', 'sorting'], 'Two pointers after sorting'),
((SELECT id FROM algorithms WHERE name = 'two_pointers'), 'leetcode', '11', 'Container With Most Water', 'https://leetcode.com/problems/container-with-most-water/', 'medium', ARRAY['array', 'two-pointers', 'greedy'], 'Classic two pointers optimization')

ON CONFLICT (platform, problem_id) DO NOTHING;

-- Create a view for easy problem lookup
CREATE OR REPLACE VIEW algorithm_problems AS
SELECT 
    a.name as algorithm_name,
    a.display_name as algorithm_display_name,
    a.category as algorithm_category,
    pm.platform,
    pm.problem_id,
    pm.problem_name,
    pm.problem_url,
    pm.difficulty,
    pm.topics,
    pm.notes
FROM algorithms a
JOIN problem_mappings pm ON a.id = pm.algorithm_id
ORDER BY a.name, pm.platform, pm.difficulty;

-- Add function to get problems by algorithm
CREATE OR REPLACE FUNCTION get_algorithm_problems(algo_id UUID)
RETURNS TABLE (
    platform VARCHAR(50),
    problem_id VARCHAR(100),
    problem_name VARCHAR(255),
    problem_url TEXT,
    difficulty VARCHAR(20),
    topics TEXT[],
    notes TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        pm.platform,
        pm.problem_id,
        pm.problem_name,
        pm.problem_url,
        pm.difficulty,
        pm.topics,
        pm.notes
    FROM problem_mappings pm
    WHERE pm.algorithm_id = algo_id
    ORDER BY 
        CASE pm.difficulty 
            WHEN 'easy' THEN 1
            WHEN 'medium' THEN 2
            WHEN 'hard' THEN 3
            ELSE 4
        END,
        pm.platform;
END;
$$ LANGUAGE plpgsql;