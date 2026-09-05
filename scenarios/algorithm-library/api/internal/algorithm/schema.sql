
-- Migration: Add performance history tracking
-- This migration adds tables to track performance trends over time

-- Performance history table - tracks algorithm performance over time
CREATE TABLE IF NOT EXISTS performance_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    algorithm_id UUID NOT NULL REFERENCES algorithms(id) ON DELETE CASCADE,
    implementation_id UUID REFERENCES implementations(id) ON DELETE CASCADE,
    language programming_language NOT NULL,
    
    -- Performance metrics
    avg_execution_time_ms DECIMAL(10,2) NOT NULL,
    min_execution_time_ms DECIMAL(10,2) NOT NULL,
    max_execution_time_ms DECIMAL(10,2) NOT NULL,
    std_dev_time_ms DECIMAL(10,2),
    
    avg_memory_mb DECIMAL(10,2) NOT NULL,
    min_memory_mb DECIMAL(10,2) NOT NULL,
    max_memory_mb DECIMAL(10,2) NOT NULL,
    
    -- Test conditions
    input_size INTEGER NOT NULL,
    sample_count INTEGER NOT NULL DEFAULT 1,
    test_category VARCHAR(50), -- 'small', 'medium', 'large', 'edge_case'
    
    -- Comparison metrics
    performance_score DECIMAL(5,2), -- 0-100 score relative to other implementations
    rank_in_category INTEGER, -- Rank among same algorithm implementations
    
    -- Metadata
    environment_info JSONB,
    notes TEXT,
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure we don't duplicate entries
    UNIQUE(algorithm_id, implementation_id, language, input_size, recorded_at)
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_perf_history_algorithm ON performance_history(algorithm_id);
CREATE INDEX IF NOT EXISTS idx_perf_history_implementation ON performance_history(implementation_id);
CREATE INDEX IF NOT EXISTS idx_perf_history_language ON performance_history(language);
CREATE INDEX IF NOT EXISTS idx_perf_history_recorded ON performance_history(recorded_at);
CREATE INDEX IF NOT EXISTS idx_perf_history_input_size ON performance_history(input_size);

-- Performance trends view - aggregated weekly performance data
CREATE OR REPLACE VIEW performance_trends AS
SELECT 
    ph.algorithm_id,
    a.name as algorithm_name,
    a.display_name,
    ph.language,
    DATE_TRUNC('week', ph.recorded_at) as week,
    AVG(ph.avg_execution_time_ms) as avg_weekly_time_ms,
    AVG(ph.avg_memory_mb) as avg_weekly_memory_mb,
    AVG(ph.performance_score) as avg_weekly_score,
    COUNT(*) as sample_count,
    MIN(ph.recorded_at) as first_recorded,
    MAX(ph.recorded_at) as last_recorded
FROM performance_history ph
JOIN algorithms a ON a.id = ph.algorithm_id
GROUP BY 
    ph.algorithm_id,
    a.name,
    a.display_name,
    ph.language,
    DATE_TRUNC('week', ph.recorded_at)
ORDER BY week DESC;

-- Performance comparison table - track head-to-head comparisons
CREATE TABLE IF NOT EXISTS performance_comparisons (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    comparison_name VARCHAR(255) NOT NULL,
    
    -- Algorithms being compared
    algorithm_ids UUID[] NOT NULL,
    implementation_ids UUID[],
    languages programming_language[],
    
    -- Test parameters
    input_sizes INTEGER[] NOT NULL,
    test_data_description TEXT,
    
    -- Results
    results JSONB NOT NULL, -- Detailed comparison results
    winner_id UUID REFERENCES algorithms(id),
    summary TEXT,
    
    -- Metadata
    compared_by VARCHAR(100),
    compared_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_perf_comparisons_date ON performance_comparisons(compared_at);
CREATE INDEX IF NOT EXISTS idx_perf_comparisons_algorithms ON performance_comparisons USING GIN(algorithm_ids);

-- Function to calculate performance score relative to other implementations
CREATE OR REPLACE FUNCTION calculate_performance_score(
    p_algorithm_id UUID,
    p_execution_time_ms DECIMAL,
    p_memory_mb DECIMAL
) RETURNS DECIMAL AS $$
DECLARE
    v_score DECIMAL;
    v_time_rank INTEGER;
    v_memory_rank INTEGER;
    v_total_implementations INTEGER;
BEGIN
    -- Get ranking for execution time (lower is better)
    SELECT COUNT(*) + 1 INTO v_time_rank
    FROM performance_history
    WHERE algorithm_id = p_algorithm_id
    AND avg_execution_time_ms < p_execution_time_ms;
    
    -- Get ranking for memory usage (lower is better)
    SELECT COUNT(*) + 1 INTO v_memory_rank
    FROM performance_history
    WHERE algorithm_id = p_algorithm_id
    AND avg_memory_mb < p_memory_mb;
    
    -- Get total number of implementations
    SELECT COUNT(DISTINCT implementation_id) INTO v_total_implementations
    FROM performance_history
    WHERE algorithm_id = p_algorithm_id;
    
    -- Calculate score (weighted: 60% time, 40% memory)
    IF v_total_implementations > 0 THEN
        v_score := 100.0 * (
            0.6 * (1.0 - (v_time_rank::DECIMAL / v_total_implementations)) +
            0.4 * (1.0 - (v_memory_rank::DECIMAL / v_total_implementations))
        );
    ELSE
        v_score := 50.0; -- Default score if no comparisons available
    END IF;
    
    RETURN ROUND(v_score, 2);
END;
$$ LANGUAGE plpgsql;

-- Trigger to update performance scores when new history is added
CREATE OR REPLACE FUNCTION update_performance_scores() RETURNS TRIGGER AS $$
BEGIN
    -- Update the performance score for the new entry
    NEW.performance_score := calculate_performance_score(
        NEW.algorithm_id,
        NEW.avg_execution_time_ms,
        NEW.avg_memory_mb
    );
    
    -- Update implementation's latest performance score
    UPDATE implementations
    SET performance_score = NEW.performance_score,
        execution_time_ms = NEW.avg_execution_time_ms,
        memory_usage_bytes = NEW.avg_memory_mb * 1024 * 1024
    WHERE id = NEW.implementation_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_performance_scores ON performance_history;
CREATE TRIGGER trigger_update_performance_scores
BEFORE INSERT OR UPDATE ON performance_history
FOR EACH ROW
EXECUTE FUNCTION update_performance_scores();

-- Add migration record (commented out since migrations table may not exist)
-- INSERT INTO migrations (name, applied_at) 
-- VALUES ('002_performance_history', NOW());
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
-- Algorithm Library Database Schema
-- Stores algorithms, implementations, test cases, and validation results

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Algorithm categories enum
DO $$
BEGIN
    CREATE TYPE algorithm_category AS ENUM (
    'sorting',
    'searching', 
    'graph',
    'dynamic_programming',
    'greedy',
    'divide_conquer',
    'backtracking',
    'string',
    'tree',
    'heap',
    'hash_table',
    'linked_list',
    'stack',
    'queue',
    'math',
    'bit_manipulation',
    'other'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END
$$;

-- Programming languages enum
DO $$
BEGIN
    CREATE TYPE programming_language AS ENUM (
    'python',
    'javascript',
    'go',
    'java',
    'cpp',
    'c',
    'rust',
    'typescript',
    'csharp',
    'ruby'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END
$$;

-- Algorithms table - core algorithm definitions
CREATE TABLE IF NOT EXISTS algorithms (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    category algorithm_category NOT NULL,
    subcategory VARCHAR(100),
    description TEXT NOT NULL,
    complexity_time VARCHAR(50) NOT NULL,  -- e.g., "O(n log n)"
    complexity_space VARCHAR(50) NOT NULL,  -- e.g., "O(1)"
    complexity_explanation TEXT,
    tags TEXT[] DEFAULT '{}',
    difficulty VARCHAR(20) CHECK (difficulty IN ('easy', 'medium', 'hard', 'expert')),
    common_applications TEXT[],
    prerequisites TEXT[],  -- Other algorithms that should be understood first
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for searching
CREATE INDEX IF NOT EXISTS idx_algorithms_category ON algorithms(category);
CREATE INDEX IF NOT EXISTS idx_algorithms_name ON algorithms(name);
CREATE INDEX IF NOT EXISTS idx_algorithms_tags ON algorithms USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_algorithms_difficulty ON algorithms(difficulty);

-- Implementations table - actual code implementations
CREATE TABLE IF NOT EXISTS implementations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    algorithm_id UUID NOT NULL REFERENCES algorithms(id) ON DELETE CASCADE,
    language programming_language NOT NULL,
    code TEXT NOT NULL,
    version VARCHAR(20) DEFAULT '1.0.0',
    is_primary BOOLEAN DEFAULT false,  -- Primary/recommended implementation
    validated BOOLEAN DEFAULT false,
    validation_count INTEGER DEFAULT 0,
    last_validation TIMESTAMP,
    performance_score DECIMAL(5,2),  -- Relative performance score 0-100
    memory_usage_bytes INTEGER,
    execution_time_ms INTEGER,
    notes TEXT,
    author VARCHAR(100) DEFAULT 'system',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(algorithm_id, language, version)
);

CREATE INDEX IF NOT EXISTS idx_implementations_algorithm ON implementations(algorithm_id);
CREATE INDEX IF NOT EXISTS idx_implementations_language ON implementations(language);

CREATE TABLE IF NOT EXISTS contributions (
    id VARCHAR(36) PRIMARY KEY,
    type VARCHAR(20) NOT NULL,
    algorithm_id VARCHAR(36),
    contributor_name VARCHAR(100) NOT NULL,
    contributor_email VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    submitted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reviewed_at TIMESTAMP,
    review_notes TEXT,
    content JSONB NOT NULL,
    CONSTRAINT fk_algorithm FOREIGN KEY (algorithm_id)
        REFERENCES algorithms(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_implementations_validated ON implementations(validated);

-- Test cases for algorithms
CREATE TABLE IF NOT EXISTS test_cases (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    algorithm_id UUID NOT NULL REFERENCES algorithms(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    input JSONB NOT NULL,
    expected_output JSONB NOT NULL,
    is_edge_case BOOLEAN DEFAULT false,
    is_performance_test BOOLEAN DEFAULT false,
    timeout_ms INTEGER DEFAULT 5000,
    memory_limit_mb INTEGER DEFAULT 128,
    sequence_order INTEGER DEFAULT 0,  -- Order to run tests
    tags TEXT[] DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_test_cases_algorithm ON test_cases(algorithm_id);
CREATE INDEX IF NOT EXISTS idx_test_cases_edge ON test_cases(is_edge_case);

-- Validation results - tracks test execution results
CREATE TABLE IF NOT EXISTS validation_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    implementation_id UUID NOT NULL REFERENCES implementations(id) ON DELETE CASCADE,
    test_case_id UUID NOT NULL REFERENCES test_cases(id) ON DELETE CASCADE,
    passed BOOLEAN NOT NULL,
    execution_time_ms INTEGER,
    memory_used_bytes INTEGER,
    actual_output JSONB,
    error_message TEXT,
    execution_id VARCHAR(100),  -- Reference to the local execution result
    validated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_validation_results_implementation ON validation_results(implementation_id);
CREATE INDEX IF NOT EXISTS idx_validation_results_test_case ON validation_results(test_case_id);
CREATE INDEX IF NOT EXISTS idx_validation_results_passed ON validation_results(passed);
CREATE INDEX IF NOT EXISTS idx_validation_results_date ON validation_results(validated_at);

-- Benchmarks table - performance comparisons
CREATE TABLE IF NOT EXISTS benchmarks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    algorithm_id UUID NOT NULL REFERENCES algorithms(id) ON DELETE CASCADE,
    language programming_language NOT NULL,
    input_size INTEGER NOT NULL,
    execution_time_ms DECIMAL(10,2) NOT NULL,
    memory_used_mb DECIMAL(10,2) NOT NULL,
    cpu_usage_percent DECIMAL(5,2),
    environment_info JSONB,  -- OS, CPU, memory specs
    notes TEXT,
    benchmarked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_benchmarks_algorithm ON benchmarks(algorithm_id);
CREATE INDEX IF NOT EXISTS idx_benchmarks_language ON benchmarks(language);
CREATE INDEX IF NOT EXISTS idx_benchmarks_input_size ON benchmarks(input_size);

-- User submissions - for tracking external validation requests
CREATE TABLE IF NOT EXISTS user_submissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    algorithm_id UUID NOT NULL REFERENCES algorithms(id) ON DELETE CASCADE,
    language programming_language NOT NULL,
    submitted_code TEXT NOT NULL,
    validation_status VARCHAR(20) CHECK (validation_status IN ('pending', 'running', 'passed', 'failed', 'error')),
    test_results JSONB,
    performance_metrics JSONB,
    feedback TEXT,
    submitted_by VARCHAR(100),  -- Could be agent name or user identifier
    submitted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_submissions_algorithm ON user_submissions(algorithm_id);
CREATE INDEX IF NOT EXISTS idx_user_submissions_status ON user_submissions(validation_status);
CREATE INDEX IF NOT EXISTS idx_user_submissions_date ON user_submissions(submitted_at);

-- Algorithm relationships - for tracking related algorithms
CREATE TABLE IF NOT EXISTS algorithm_relationships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    from_algorithm_id UUID NOT NULL REFERENCES algorithms(id) ON DELETE CASCADE,
    to_algorithm_id UUID NOT NULL REFERENCES algorithms(id) ON DELETE CASCADE,
    relationship_type VARCHAR(50) CHECK (relationship_type IN (
        'variant_of',       -- e.g., QuickSort is variant of Sorting
        'optimizes',        -- e.g., HeapSort optimizes SelectionSort
        'uses',            -- e.g., MergeSort uses Divide&Conquer
        'prerequisite',    -- Should understand this first
        'alternative'      -- Can be used instead of
    )),
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(from_algorithm_id, to_algorithm_id, relationship_type)
);

CREATE INDEX IF NOT EXISTS idx_algorithm_relationships_from ON algorithm_relationships(from_algorithm_id);
CREATE INDEX IF NOT EXISTS idx_algorithm_relationships_to ON algorithm_relationships(to_algorithm_id);
CREATE INDEX IF NOT EXISTS idx_algorithm_relationships_type ON algorithm_relationships(relationship_type);

-- Usage statistics - track which algorithms are most accessed
CREATE TABLE IF NOT EXISTS usage_stats (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    algorithm_id UUID REFERENCES algorithms(id) ON DELETE CASCADE,
    implementation_id UUID REFERENCES implementations(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL CHECK (action IN ('view', 'copy', 'validate', 'benchmark', 'api_call')),
    caller VARCHAR(100),  -- Agent or scenario name
    metadata JSONB,
    accessed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_usage_stats_algorithm ON usage_stats(algorithm_id);
CREATE INDEX IF NOT EXISTS idx_usage_stats_action ON usage_stats(action);
CREATE INDEX IF NOT EXISTS idx_usage_stats_date ON usage_stats(accessed_at);

-- Create update trigger for updated_at columns
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

DROP TRIGGER IF EXISTS update_algorithms_updated_at ON algorithms;
CREATE TRIGGER update_algorithms_updated_at BEFORE UPDATE ON algorithms
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_implementations_updated_at ON implementations;
CREATE TRIGGER update_implementations_updated_at BEFORE UPDATE ON implementations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Helper views for common queries
CREATE OR REPLACE VIEW v_algorithm_summary AS
SELECT 
    a.id,
    a.name,
    a.display_name,
    a.category,
    a.difficulty,
    a.complexity_time,
    a.complexity_space,
    COUNT(DISTINCT i.language) as language_count,
    COUNT(DISTINCT tc.id) as test_case_count,
    BOOL_OR(i.validated) as has_validated_impl
FROM algorithms a
LEFT JOIN implementations i ON a.id = i.algorithm_id
LEFT JOIN test_cases tc ON a.id = tc.algorithm_id
GROUP BY a.id;

CREATE OR REPLACE VIEW v_implementation_status AS
SELECT 
    i.id,
    a.name as algorithm_name,
    i.language,
    i.validated,
    i.validation_count,
    i.performance_score,
    COUNT(DISTINCT vr.test_case_id) as tests_passed,
    COUNT(DISTINCT tc.id) as total_tests
FROM implementations i
JOIN algorithms a ON i.algorithm_id = a.id
LEFT JOIN validation_results vr ON i.id = vr.implementation_id AND vr.passed = true
LEFT JOIN test_cases tc ON a.id = tc.algorithm_id
GROUP BY i.id, a.name;

-- Grant permissions (adjust as needed)
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO postgres;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO postgres;
