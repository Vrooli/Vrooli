package collectors

import (
	"context"
	"time"
)

// Collector defines the interface for metric collectors
type Collector interface {
	// Collect gathers metrics from the system
	Collect(ctx context.Context) (*MetricData, error)
	
	// GetName returns the collector name
	GetName() string
	
	// GetInterval returns the collection interval
	GetInterval() time.Duration
	
	// IsEnabled returns whether the collector is enabled
	IsEnabled() bool
	
	// SetEnabled enables or disables the collector
	SetEnabled(enabled bool)
}

// MetricData represents collected metrics
type MetricData struct {
	CollectorName string                 `json:"collector_name"`
	Timestamp     time.Time              `json:"timestamp"`
	Type          string                 `json:"type"`
	Values        map[string]interface{} `json:"values"`
	Tags          map[string]string      `json:"tags,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// CollectorRegistry manages all collectors
type CollectorRegistry struct {
	collectors map[string]Collector
}

// NewCollectorRegistry creates a new collector registry
func NewCollectorRegistry() *CollectorRegistry {
	return &CollectorRegistry{
		collectors: make(map[string]Collector),
	}
}

// Register adds a collector to the registry
func (r *CollectorRegistry) Register(collector Collector) {
	r.collectors[collector.GetName()] = collector
}

// Get retrieves a collector by name
func (r *CollectorRegistry) Get(name string) (Collector, bool) {
	collector, exists := r.collectors[name]
	return collector, exists
}

// GetAll returns all registered collectors
func (r *CollectorRegistry) GetAll() []Collector {
	collectors := make([]Collector, 0, len(r.collectors))
	for _, collector := range r.collectors {
		collectors = append(collectors, collector)
	}
	return collectors
}

// GetEnabled returns only enabled collectors
func (r *CollectorRegistry) GetEnabled() []Collector {
	collectors := make([]Collector, 0)
	for _, collector := range r.collectors {
		if collector.IsEnabled() {
			collectors = append(collectors, collector)
		}
	}
	return collectors
}

// CollectAll collects metrics from all enabled collectors
func (r *CollectorRegistry) CollectAll(ctx context.Context) ([]*MetricData, []error) {
	enabled := r.GetEnabled()
	results := make([]*MetricData, 0, len(enabled))
	errors := make([]error, 0)
	
	for _, collector := range enabled {
		data, err := collector.Collect(ctx)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		results = append(results, data)
	}
	
	return results, errors
}

// BaseCollector provides common functionality for collectors
type BaseCollector struct {
	name     string
	interval time.Duration
	enabled  bool
}

// NewBaseCollector creates a new base collector
func NewBaseCollector(name string, interval time.Duration) BaseCollector {
	return BaseCollector{
		name:     name,
		interval: interval,
		enabled:  true,
	}
}

// GetName returns the collector name
func (b *BaseCollector) GetName() string {
	return b.name
}

// GetInterval returns the collection interval
func (b *BaseCollector) GetInterval() time.Duration {
	return b.interval
}

// IsEnabled returns whether the collector is enabled
func (b *BaseCollector) IsEnabled() bool {
	return b.enabled
}

// SetEnabled enables or disables the collector
func (b *BaseCollector) SetEnabled(enabled bool) {
	b.enabled = enabled
}