-- Math Tools Database Schema
-- Comprehensive mathematical computation and analysis platform

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Mathematical Models table
CREATE TABLE IF NOT EXISTS mathematical_models (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    model_type VARCHAR(50) NOT NULL CHECK (model_type IN ('linear_regression', 'polynomial', 'optimization', 'differential_equation', 'statistical', 'forecasting')),
    formula TEXT,
    parameters JSONB,
    variables JSONB,
    constraints JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_used TIMESTAMP WITH TIME ZONE,
    accuracy_metrics JSONB,
    description TEXT,
    tags TEXT[],
    version INTEGER DEFAULT 1
);

-- Calculations history table
CREATE TABLE IF NOT EXISTS calculations (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    model_id UUID REFERENCES mathematical_models(id) ON DELETE SET NULL,
    operation_type VARCHAR(50) NOT NULL,
    input_data JSONB NOT NULL,
    result JSONB NOT NULL,
    execution_time_ms INTEGER,
    precision_level DECIMAL(10,8),
    algorithm_used VARCHAR(100),
    convergence_status VARCHAR(20) CHECK (convergence_status IN ('converged', 'max_iterations', 'failed')),
    error_estimate DECIMAL(20,15),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB
);

-- Datasets table
CREATE TABLE IF NOT EXISTS datasets (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    data_type VARCHAR(50) CHECK (data_type IN ('time_series', 'cross_sectional', 'panel', 'matrix')),
    size_rows BIGINT,
    size_columns INTEGER,
    data_path TEXT,
    schema_definition JSONB,
    statistical_summary JSONB,
    quality_metrics JSONB,
    source VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_analyzed TIMESTAMP WITH TIME ZONE,
    tags TEXT[]
);

-- Statistical Analyses table
CREATE TABLE IF NOT EXISTS statistical_analyses (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    dataset_id UUID REFERENCES datasets(id) ON DELETE CASCADE,
    analysis_type VARCHAR(50) NOT NULL,
    parameters JSONB,
    results JSONB NOT NULL,
    confidence_level DECIMAL(3,2),
    p_value DECIMAL(10,8),
    test_statistic DECIMAL(20,10),
    degrees_of_freedom INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB
);

-- Visualizations table
CREATE TABLE IF NOT EXISTS visualizations (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    dataset_id UUID REFERENCES datasets(id) ON DELETE CASCADE,
    calculation_id UUID REFERENCES calculations(id) ON DELETE CASCADE,
    plot_type VARCHAR(50) CHECK (plot_type IN ('scatter', 'line', 'histogram', 'heatmap', 'surface', 'box', 'violin', 'contour')),
    configuration JSONB,
    image_path TEXT,
    interactive_config JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    view_count INTEGER DEFAULT 0,
    export_formats TEXT[],
    dimensions JSONB
);

-- Optimization Problems table
CREATE TABLE IF NOT EXISTS optimization_problems (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    objective_function TEXT NOT NULL,
    variables JSONB NOT NULL,
    constraints JSONB,
    optimization_type VARCHAR(10) CHECK (optimization_type IN ('minimize', 'maximize')),
    algorithm VARCHAR(50),
    initial_guess JSONB,
    bounds JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Optimization Results table
CREATE TABLE IF NOT EXISTS optimization_results (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    problem_id UUID REFERENCES optimization_problems(id) ON DELETE CASCADE,
    optimal_solution JSONB NOT NULL,
    optimal_value DECIMAL(20,10),
    status VARCHAR(20) CHECK (status IN ('optimal', 'feasible', 'infeasible', 'unbounded')),
    iterations INTEGER,
    algorithm_used VARCHAR(50),
    execution_time_ms INTEGER,
    sensitivity_analysis JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Time Series Forecasts table
CREATE TABLE IF NOT EXISTS time_series_forecasts (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    dataset_id UUID REFERENCES datasets(id) ON DELETE CASCADE,
    forecast_horizon INTEGER NOT NULL,
    method VARCHAR(50) CHECK (method IN ('arima', 'exponential_smoothing', 'linear_trend', 'polynomial', 'prophet', 'lstm')),
    forecast_values JSONB NOT NULL,
    confidence_intervals JSONB,
    model_parameters JSONB,
    model_metrics JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Matrix Operations Cache table (for expensive computations)
CREATE TABLE IF NOT EXISTS matrix_cache (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    operation VARCHAR(50) NOT NULL,
    input_hash VARCHAR(64) NOT NULL UNIQUE,
    matrix_a JSONB,
    matrix_b JSONB,
    result JSONB NOT NULL,
    computation_time_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    accessed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    access_count INTEGER DEFAULT 1
);

-- Create indexes for performance
CREATE INDEX idx_models_type ON mathematical_models(model_type);
CREATE INDEX idx_models_tags ON mathematical_models USING GIN(tags);
CREATE INDEX idx_calculations_operation ON calculations(operation_type);
CREATE INDEX idx_calculations_created ON calculations(created_at DESC);
CREATE INDEX idx_datasets_type ON datasets(data_type);
CREATE INDEX idx_datasets_tags ON datasets USING GIN(tags);
CREATE INDEX idx_analyses_type ON statistical_analyses(analysis_type);
CREATE INDEX idx_visualizations_type ON visualizations(plot_type);
CREATE INDEX idx_matrix_cache_hash ON matrix_cache(input_hash);
CREATE INDEX idx_matrix_cache_accessed ON matrix_cache(accessed_at DESC);

-- Create search indexes
CREATE INDEX idx_models_search ON mathematical_models USING GIN(to_tsvector('english', name || ' ' || COALESCE(description, '')));
CREATE INDEX idx_datasets_search ON datasets USING GIN(to_tsvector('english', name || ' ' || COALESCE(source, '')));

-- Create function to update last_used timestamp
CREATE OR REPLACE FUNCTION update_model_last_used()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.model_id IS NOT NULL THEN
        UPDATE mathematical_models 
        SET last_used = CURRENT_TIMESTAMP 
        WHERE id = NEW.model_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for updating last_used
CREATE TRIGGER update_model_usage
    AFTER INSERT ON calculations
    FOR EACH ROW
    EXECUTE FUNCTION update_model_last_used();

-- Create function to update matrix cache access
CREATE OR REPLACE FUNCTION update_cache_access()
RETURNS TRIGGER AS $$
BEGIN
    NEW.accessed_at = CURRENT_TIMESTAMP;
    NEW.access_count = OLD.access_count + 1;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for cache access tracking
CREATE TRIGGER track_cache_access
    BEFORE UPDATE ON matrix_cache
    FOR EACH ROW
    WHEN (OLD.* IS DISTINCT FROM NEW.*)
    EXECUTE FUNCTION update_cache_access();

-- Add comments for documentation
COMMENT ON TABLE mathematical_models IS 'Stores reusable mathematical models and formulas';
COMMENT ON TABLE calculations IS 'History of all mathematical calculations performed';
COMMENT ON TABLE datasets IS 'Metadata for stored datasets used in calculations';
COMMENT ON TABLE statistical_analyses IS 'Results of statistical analyses performed on datasets';
COMMENT ON TABLE visualizations IS 'Configuration and metadata for generated plots';
COMMENT ON TABLE optimization_problems IS 'Definition of optimization problems';
COMMENT ON TABLE optimization_results IS 'Solutions to optimization problems';
COMMENT ON TABLE time_series_forecasts IS 'Time series forecasting models and results';
COMMENT ON TABLE matrix_cache IS 'Cache for expensive matrix computations';