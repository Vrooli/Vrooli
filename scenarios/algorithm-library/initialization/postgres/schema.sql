-- Algorithm Library Database Schema
-- Stores algorithms, implementations, test cases, and validation results

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Algorithm categories enum
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

-- Programming languages enum
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
CREATE INDEX idx_algorithms_category ON algorithms(category);
CREATE INDEX idx_algorithms_name ON algorithms(name);
CREATE INDEX idx_algorithms_tags ON algorithms USING GIN(tags);
CREATE INDEX idx_algorithms_difficulty ON algorithms(difficulty);

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

CREATE INDEX idx_implementations_algorithm ON implementations(algorithm_id);
CREATE INDEX idx_implementations_language ON implementations(language);
CREATE INDEX idx_implementations_validated ON implementations(validated);

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

CREATE INDEX idx_test_cases_algorithm ON test_cases(algorithm_id);
CREATE INDEX idx_test_cases_edge ON test_cases(is_edge_case);

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
    judge0_submission_id VARCHAR(100),  -- Reference to Judge0 submission
    validated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_validation_results_implementation ON validation_results(implementation_id);
CREATE INDEX idx_validation_results_test_case ON validation_results(test_case_id);
CREATE INDEX idx_validation_results_passed ON validation_results(passed);
CREATE INDEX idx_validation_results_date ON validation_results(validated_at);

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

CREATE INDEX idx_benchmarks_algorithm ON benchmarks(algorithm_id);
CREATE INDEX idx_benchmarks_language ON benchmarks(language);
CREATE INDEX idx_benchmarks_input_size ON benchmarks(input_size);

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

CREATE INDEX idx_user_submissions_algorithm ON user_submissions(algorithm_id);
CREATE INDEX idx_user_submissions_status ON user_submissions(validation_status);
CREATE INDEX idx_user_submissions_date ON user_submissions(submitted_at);

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

CREATE INDEX idx_algorithm_relationships_from ON algorithm_relationships(from_algorithm_id);
CREATE INDEX idx_algorithm_relationships_to ON algorithm_relationships(to_algorithm_id);
CREATE INDEX idx_algorithm_relationships_type ON algorithm_relationships(relationship_type);

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

CREATE INDEX idx_usage_stats_algorithm ON usage_stats(algorithm_id);
CREATE INDEX idx_usage_stats_action ON usage_stats(action);
CREATE INDEX idx_usage_stats_date ON usage_stats(accessed_at);

-- Create update trigger for updated_at columns
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_algorithms_updated_at BEFORE UPDATE ON algorithms
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_implementations_updated_at BEFORE UPDATE ON implementations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Helper views for common queries
CREATE VIEW v_algorithm_summary AS
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

CREATE VIEW v_implementation_status AS
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