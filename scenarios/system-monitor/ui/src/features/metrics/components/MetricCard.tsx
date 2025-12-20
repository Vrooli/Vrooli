import { ChevronDown, ChevronUp } from 'lucide-react';
import type {
  CardType,
  CPUMetrics,
  MemoryMetrics,
  NetworkMetrics,
  ChartDataPoint,
  DiskCardDetails,
  GPUCardDetails
} from '../../../types';
/* eslint-disable no-case-declarations */
import { MetricSparkline } from './MetricSparkline';

const formatBytes = (value?: number) => {
  if (!Number.isFinite(value ?? NaN) || !value) {
    return '—';
  }
  const absolute = Math.max(value, 0);
  if (absolute === 0) {
    return '0 B';
  }
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
  const exponent = Math.min(Math.floor(Math.log(absolute) / Math.log(1024)), units.length - 1);
  const scaled = absolute / Math.pow(1024, exponent);
  const precision = scaled >= 100 ? 0 : scaled >= 10 ? 1 : 2;
  return `${scaled.toFixed(precision)} ${units[exponent]}`;
};

interface MetricCardProps {
  type: CardType;
  label: string;
  unit: string;
  value?: number | null;
  isExpanded: boolean;
  onToggle: () => void;
  details?: CPUMetrics | MemoryMetrics | NetworkMetrics | DiskCardDetails | GPUCardDetails;
  alertCount: number;
  history?: ChartDataPoint[];
  historyWindowSeconds?: number;
  valueDomain?: [number, number];
  threshold?: number;
  historyUnit?: string;
  onOpenDetails?: () => void;
  detailButtonLabel?: string;
}

export const MetricCard = ({ 
  type, 
  label, 
  unit, 
  value, 
  isExpanded, 
  onToggle, 
  details, 
  alertCount,
  history,
  historyWindowSeconds,
  valueDomain,
  threshold,
  historyUnit,
  onOpenDetails,
  detailButtonLabel = 'VIEW DETAIL'
}: MetricCardProps) => {
  const hasNumericValue = typeof value === 'number' && Number.isFinite(value);
  const resolvedValue = hasNumericValue ? (value as number) : null;
  
  const getProgressValue = (): number => {
    if (!hasNumericValue || resolvedValue === null) {
      return 0;
    }
    return Math.min(resolvedValue, 100);
  };

  const getValueColor = (): string => {
    if (!hasNumericValue || resolvedValue === null) {
      return 'var(--color-text-dim)';
    }
    if (resolvedValue >= 90) return 'var(--color-error)';
    if (resolvedValue >= 70) return 'var(--color-warning)';
    return 'var(--color-text-bright)';
  };

  const sparklineColor = (() => {
    switch (type) {
      case 'cpu':
        return 'var(--color-accent)';
      case 'memory':
        return 'var(--color-warning)';
      case 'network':
        return 'var(--color-accent)';
      case 'disk':
        return 'var(--color-info)';
      case 'gpu':
        return 'var(--color-info)';
      default:
        return 'var(--color-accent)';
    }
  })();

  const defaultThresholds: Partial<Record<CardType, number>> = {
    cpu: 75,
    memory: 80,
    disk: 80,
    gpu: 85
  };

  const sparklineThreshold = typeof threshold === 'number' ? threshold : defaultThresholds[type];

  const formatWindowLabel = (seconds?: number): string | undefined => {
    if (!seconds) return undefined;
    if (seconds < 60) {
      return `Past ${seconds}s`;
    }
    const minutes = seconds / 60;
    if (Number.isInteger(minutes)) {
      return `Past ${minutes.toFixed(0)}m`;
    }
    return `Past ${minutes.toFixed(1)}m`;
  };

  const sparklineUnit = (() => {
    if (historyUnit) {
      return historyUnit;
    }
    if (type === 'network') {
      return ' connections';
    }
    if (type === 'gpu') {
      return '%';
    }
    if (unit === '%') {
      return '%';
    }
    return unit ? ` ${unit}` : undefined;
  })();

  const formatPercentage = (val?: number | null) => {
    if (!Number.isFinite(val ?? NaN)) {
      return '—';
    }
    return `${Number(val).toFixed(1)}%`;
  };

  const formatMegabytes = (val?: number | null) => {
    if (!Number.isFinite(val ?? NaN)) {
      return '—';
    }
    return `${Number(val).toFixed(0)} MB`;
  };

  const renderExpandedContent = () => {
    if (!isExpanded || !details) return null;

    switch (type) {
      case 'cpu':
        const cpuDetails = details as CPUMetrics;
        return (
          <div className="metric-details" style={{ marginTop: 'var(--spacing-md)' }}>
            <div className="detail-section" style={{ marginBottom: 'var(--spacing-md)' }}>
              <h4 style={{ margin: '0 0 var(--spacing-sm) 0', color: 'var(--color-text-bright)' }}>
                Top Processes by CPU:
              </h4>
              <div className="process-list">
                {cpuDetails.top_processes.slice(0, 5).map((process) => (
                  <div key={process.pid} style={{ 
                    display: 'flex', 
                    justifyContent: 'space-between',
                    margin: 'var(--spacing-xs) 0',
                    fontSize: 'var(--font-size-sm)'
                  }}>
                    <span>{process.name} ({process.pid})</span>
                    <span style={{ color: 'var(--color-accent)' }}>
                      {process.cpu_percent.toFixed(1)}%
                    </span>
                  </div>
                ))}
              </div>
            </div>
            
            <div className="detail-row" style={{ 
              display: 'grid', 
              gridTemplateColumns: '1fr 1fr', 
              gap: 'var(--spacing-md)',
              marginBottom: 'var(--spacing-sm)'
            }}>
              <div className="detail-item">
                <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
                  Load Average:
                </span>
                <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
                  {cpuDetails.load_average.slice(0, 3).map(load => load.toFixed(2)).join(', ')}
                </span>
              </div>
              <div className="detail-item">
                <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
                  Context Switches:
                </span>
                <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
                  {cpuDetails.context_switches.toLocaleString()}
                </span>
              </div>
            </div>
            
            <div className="detail-item">
              <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
                Total Goroutines:
              </span>
              <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
                {cpuDetails.total_goroutines}
              </span>
            </div>
          </div>
        );

      case 'memory':
        const memoryDetails = details as MemoryMetrics;
        return (
          <div className="metric-details" style={{ marginTop: 'var(--spacing-md)' }}>
            <div className="detail-section" style={{ marginBottom: 'var(--spacing-md)' }}>
              <h4 style={{ margin: '0 0 var(--spacing-sm) 0', color: 'var(--color-text-bright)' }}>
                Top Processes by Memory:
              </h4>
              <div className="process-list">
                {memoryDetails.top_processes.slice(0, 5).map((process) => (
                  <div key={process.pid} style={{ 
                    display: 'flex', 
                    justifyContent: 'space-between',
                    margin: 'var(--spacing-xs) 0',
                    fontSize: 'var(--font-size-sm)'
                  }}>
                    <span>{process.name} ({process.pid})</span>
                    <span style={{ color: 'var(--color-accent)' }}>
                      {process.memory_mb.toFixed(1)} MB
                    </span>
                  </div>
                ))}
              </div>
            </div>
            
            <div className="detail-row" style={{ 
              display: 'grid', 
              gridTemplateColumns: '1fr 1fr', 
              gap: 'var(--spacing-md)',
              marginBottom: 'var(--spacing-sm)'
            }}>
              <div className="detail-item">
                <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
                  Swap Usage:
                </span>
                <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
                  {memoryDetails.swap_usage.percent.toFixed(1)}%
                </span>
              </div>
              <div className="detail-item">
                <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
                  Disk Usage:
                </span>
                <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
                  {memoryDetails.disk_usage.percent.toFixed(1)}%
                </span>
              </div>
            </div>
            
            {memoryDetails.growth_patterns.length > 0 && (
              <div className="detail-section">
                <h4 style={{ margin: '0 0 var(--spacing-sm) 0', color: 'var(--color-text-bright)' }}>
                  Memory Growth Patterns:
                </h4>
                <div className="growth-patterns">
                  {memoryDetails.growth_patterns.slice(0, 3).map((pattern, index) => (
                    <div key={index} style={{ 
                      margin: 'var(--spacing-xs) 0',
                      fontSize: 'var(--font-size-sm)'
                    }}>
                      <span>{pattern.process}: </span>
                      <span style={{ 
                        color: pattern.risk_level === 'high' ? 'var(--color-error)' : 
                               pattern.risk_level === 'medium' ? 'var(--color-warning)' : 
                               'var(--color-accent)'
                      }}>
                        {pattern.growth_mb_per_hour.toFixed(1)} MB/hr ({pattern.risk_level})
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        );

      case 'disk':
        const diskDetails = details as DiskCardDetails;
        const freeBytes = Number.isFinite(diskDetails.diskUsage.total - diskDetails.diskUsage.used)
          ? diskDetails.diskUsage.total - diskDetails.diskUsage.used
          : undefined;
        return (
          <div className="metric-details" style={{ marginTop: 'var(--spacing-md)' }}>
            <div className="detail-row" style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))',
              gap: 'var(--spacing-md)',
              marginBottom: 'var(--spacing-md)'
            }}>
              <div className="detail-item">
                <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
                  Total Capacity:
                </span>
                <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
                  {formatBytes(diskDetails.diskUsage.total)}
                </span>
              </div>
              <div className="detail-item">
                <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
                  Used:
                </span>
                <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
                  {formatBytes(diskDetails.diskUsage.used)}
                </span>
              </div>
              <div className="detail-item">
                <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
                  Free:
                </span>
                <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
                  {formatBytes(freeBytes)}
                </span>
              </div>
              <div className="detail-item">
                <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
                  Utilization:
                </span>
                <span className="detail-value" style={{ color: 'var(--color-warning)' }}>
                  {diskDetails.diskUsage.percent.toFixed(1)}%
                </span>
              </div>
            </div>

            {diskDetails.storageIO && (
              <div className="detail-row" style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))',
                gap: 'var(--spacing-md)',
                marginBottom: 'var(--spacing-sm)'
              }}>
                <div className="detail-item">
                  <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
                    Disk Queue Depth:
                  </span>
                  <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
                    {diskDetails.storageIO.disk_queue_depth.toFixed(2)}
                  </span>
                </div>
                <div className="detail-item">
                  <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
                    I/O Wait:
                  </span>
                  <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
                    {diskDetails.storageIO.io_wait_percent.toFixed(1)}%
                  </span>
                </div>
                <div className="detail-item">
                  <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
                    Read Throughput:
                  </span>
                  <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
                    {diskDetails.storageIO.read_mb_per_sec.toFixed(2)} MB/s
                  </span>
                </div>
                <div className="detail-item">
                  <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
                    Write Throughput:
                  </span>
                  <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
                    {diskDetails.storageIO.write_mb_per_sec.toFixed(2)} MB/s
                  </span>
                </div>
              </div>
            )}

            {diskDetails.lastUpdated && (
              <div style={{
                marginTop: 'var(--spacing-sm)',
                color: 'var(--color-text-dim)',
                fontSize: 'var(--font-size-xs)'
              }}>
                Updated {new Date(diskDetails.lastUpdated).toLocaleTimeString()}
              </div>
            )}
          </div>
        );

      case 'gpu':
        const gpuDetails = details as GPUCardDetails;
        const metrics = gpuDetails?.metrics;
        if (!metrics) {
          return (
            <div style={{ marginTop: 'var(--spacing-md)', color: 'var(--color-text-dim)' }}>
              GPU metrics unavailable. Ensure compatible NVIDIA drivers are installed.
            </div>
          );
        }

        const { summary, devices, errors } = metrics;

        return (
          <div className="metric-details" style={{ marginTop: 'var(--spacing-md)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-md)' }}>
            <div className="detail-row" style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))',
              gap: 'var(--spacing-md)'
            }}>
              <div className="detail-item">
                <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>Devices:</span>
                <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>{summary.device_count}</span>
              </div>
              <div className="detail-item">
                <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>Average Utilization:</span>
                <span className="detail-value" style={{ color: 'var(--color-accent)' }}>{summary.average_utilization_percent.toFixed(1)}%</span>
              </div>
              <div className="detail-item">
                <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>Memory:</span>
                <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
                  {summary.used_memory_mb.toFixed(0)} / {summary.total_memory_mb.toFixed(0)} MB
                </span>
              </div>
              <div className="detail-item">
                <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>Avg Temp:</span>
                <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
                  {summary.device_count > 0 && summary.average_temperature_c > 0 ? `${summary.average_temperature_c.toFixed(1)}°C` : '—'}
                </span>
              </div>
            </div>

            {metrics.driver_version && (
              <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
                Driver Version: <span style={{ color: 'var(--color-text-bright)' }}>{metrics.driver_version}</span>
              </div>
            )}

            {metrics.primary_model && (
              <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
                Primary Model: <span style={{ color: 'var(--color-text-bright)' }}>{metrics.primary_model}</span>
              </div>
            )}

            {errors && errors.length > 0 && (
              <div style={{
                border: '1px solid var(--color-warning)',
                padding: 'var(--spacing-sm) var(--spacing-md)',
                borderRadius: 'var(--border-radius-md)',
                color: 'var(--color-warning)',
                fontSize: 'var(--font-size-sm)'
              }}>
                {errors.join(' • ')}
              </div>
            )}

            {devices.length > 0 ? (
              <div className="device-list" style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-sm)' }}>
                {devices.map(device => (
                  <div key={device.uuid || device.index} style={{
                    border: '1px solid var(--color-surface)',
                    borderRadius: 'var(--border-radius-md)',
                    padding: 'var(--spacing-md)',
                    background: 'rgba(0, 40, 0, 0.25)'
                  }}>
                    <div style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                      marginBottom: 'var(--spacing-sm)'
                    }}>
                      <span style={{ color: 'var(--color-text-bright)', fontWeight: 600 }}>
                        {device.name} (GPU {device.index})
                      </span>
                      <span style={{ color: 'var(--color-accent)', fontSize: 'var(--font-size-sm)' }}>
                        {device.utilization_percent.toFixed(1)}%
                      </span>
                    </div>

                    <div style={{
                      display: 'grid',
                      gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))',
                      gap: 'var(--spacing-sm)',
                      fontSize: 'var(--font-size-xs)',
                      color: 'var(--color-text-dim)'
                    }}>
                      <div>Memory: <span style={{ color: 'var(--color-text-bright)' }}>{formatMegabytes(device.memory_used_mb)} / {formatMegabytes(device.memory_total_mb)}</span></div>
                      <div>Memory Util: <span style={{ color: 'var(--color-text-bright)' }}>{formatPercentage(device.memory_utilization_percent)}</span></div>
                      <div>Temp: <span style={{ color: 'var(--color-text-bright)' }}>{device.temperature_c != null ? `${device.temperature_c.toFixed(1)}°C` : '—'}</span></div>
                      <div>Fan: <span style={{ color: 'var(--color-text-bright)' }}>{device.fan_speed_percent != null ? `${device.fan_speed_percent.toFixed(0)}%` : '—'}</span></div>
                      <div>Power: <span style={{ color: 'var(--color-text-bright)' }}>{device.power_draw_w != null ? `${device.power_draw_w.toFixed(1)} W` : '—'}</span></div>
                      <div>SM Clock: <span style={{ color: 'var(--color-text-bright)' }}>{device.sm_clock_mhz != null ? `${device.sm_clock_mhz.toFixed(0)} MHz` : '—'}</span></div>
                      <div>Mem Clock: <span style={{ color: 'var(--color-text-bright)' }}>{device.memory_clock_mhz != null ? `${device.memory_clock_mhz.toFixed(0)} MHz` : '—'}</span></div>
                    </div>

                    {device.processes && device.processes.length > 0 && (
                      <div style={{ marginTop: 'var(--spacing-sm)' }}>
                        <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)', marginBottom: 'var(--spacing-xs)' }}>
                          Active Processes
                        </div>
                        <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-xxs)' }}>
                          {device.processes.slice(0, 5).map(process => (
                            <div key={`${device.uuid}-${process.pid}`} style={{
                              display: 'flex',
                              justifyContent: 'space-between',
                              fontSize: 'var(--font-size-xs)',
                              color: 'var(--color-text-bright)'
                            }}>
                              <span>{process.process_name} ({process.pid})</span>
                              <span>{formatMegabytes(process.memory_used_mb)}</span>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            ) : (
              <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
                No GPU devices detected.
              </div>
            )}

            {gpuDetails.lastUpdated && (
              <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)' }}>
                Updated {new Date(gpuDetails.lastUpdated).toLocaleTimeString()}
              </div>
            )}
          </div>
        );

      case 'network':
        const networkDetails = details as NetworkMetrics;
        return (
          <div className="metric-details" style={{ marginTop: 'var(--spacing-md)' }}>
            <div className="detail-section" style={{ marginBottom: 'var(--spacing-md)' }}>
              <h4 style={{ margin: '0 0 var(--spacing-sm) 0', color: 'var(--color-text-bright)' }}>
                Connection States:
              </h4>
              <div className="connection-states" style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(2, 1fr)',
                gap: 'var(--spacing-sm)',
                fontSize: 'var(--font-size-sm)'
              }}>
                <div>Established: <span style={{ color: 'var(--color-accent)' }}>
                  {networkDetails.tcp_states.established}
                </span></div>
                <div>Time Wait: <span style={{ color: 'var(--color-accent)' }}>
                  {networkDetails.tcp_states.time_wait}
                </span></div>
                <div>Listen: <span style={{ color: 'var(--color-accent)' }}>
                  {networkDetails.tcp_states.listen}
                </span></div>
                <div>Total: <span style={{ color: 'var(--color-accent)' }}>
                  {networkDetails.tcp_states.total}
                </span></div>
              </div>
            </div>
            
            <div className="detail-row" style={{ 
              display: 'grid', 
              gridTemplateColumns: '1fr 1fr', 
              gap: 'var(--spacing-md)',
              marginBottom: 'var(--spacing-sm)'
            }}>
              <div className="detail-item">
                <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
                  Port Usage:
                </span>
                <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
                  {networkDetails.port_usage.used} / {networkDetails.port_usage.total}
                </span>
              </div>
              <div className="detail-item">
                <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
                  DNS Health:
                </span>
                <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
                  {networkDetails.network_stats.dns_success_rate.toFixed(1)}%
                </span>
              </div>
            </div>
          </div>
        );

      default:
        return null;
    }
  };

  return (
    <div 
      className={`metric-card expandable ${isExpanded ? 'expanded' : ''}`}
      onClick={onToggle}
      style={{
        background: 'rgba(0, 0, 0, 0.9)',
        border: '1px solid var(--color-accent)',
        borderRadius: 'var(--border-radius-lg)',
        padding: 'var(--spacing-lg)',
        cursor: 'pointer',
        transition: 'all var(--transition-normal)',
        ...(isExpanded && {
          borderColor: 'var(--color-text-bright)',
          boxShadow: '0 0 20px var(--color-matrix-glow)'
        })
      }}
    >
      <div className="metric-header" style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 'var(--spacing-md)'
      }}>
        <span className="metric-label" style={{
          fontSize: 'var(--font-size-sm)',
          color: 'var(--color-text-bright)',
          letterSpacing: '0.1em'
        }}>
          {label}
        </span>
        
        <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-sm)' }}>
          {alertCount > 0 && (
            <span className="alert-badge" style={{
              background: 'var(--color-error)',
              color: 'var(--color-background)',
              padding: 'var(--spacing-xs) var(--spacing-sm)',
              borderRadius: '50px',
              fontSize: 'var(--font-size-xs)',
              fontWeight: 'bold',
              minWidth: '20px',
              textAlign: 'center'
            }}>
              {alertCount}
            </span>
          )}
          
          <span className="metric-unit" style={{
            fontSize: 'var(--font-size-sm)',
            color: 'var(--color-text-dim)'
          }}>
            {unit}
          </span>
          
          <span className="expand-arrow" style={{ color: 'var(--color-accent)' }}>
            {isExpanded ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
          </span>
        </div>
      </div>

      <div className="metric-value" style={{
        fontSize: 'var(--font-size-xxl)',
        fontWeight: 'bold',
        color: getValueColor(),
        textShadow: 'var(--text-shadow-glow)',
        marginBottom: 'var(--spacing-sm)'
      }}>
        {resolvedValue !== null ? resolvedValue.toFixed(1) : '—'}
      </div>

      {history && history.length > 0 ? (
        <MetricSparkline
          data={history}
          color={sparklineColor}
          valueDomain={valueDomain}
          threshold={sparklineThreshold}
          unit={sparklineUnit}
          windowLabel={formatWindowLabel(historyWindowSeconds)}
        />
      ) : (
        <div className="metric-bar" style={{
          width: '100%',
          height: '4px',
          background: 'var(--alpha-accent-20)',
          borderRadius: '2px',
          overflow: 'hidden'
        }}>
          <div 
            className="metric-fill" 
            style={{
              height: '100%',
              width: `${getProgressValue()}%`,
              background: 'linear-gradient(90deg, var(--color-accent), var(--color-text-bright))',
              transition: 'width var(--transition-normal)',
              boxShadow: '0 0 10px var(--color-matrix-green)'
            }}
          />
        </div>
      )}

      {renderExpandedContent()}

      {onOpenDetails && (
        <div
          style={{
            marginTop: 'var(--spacing-md)',
            display: 'flex',
            justifyContent: 'flex-end'
          }}
        >
          <button
            type="button"
            className="btn btn-action"
            style={{
              letterSpacing: '0.08em',
              fontSize: 'var(--font-size-xs)'
            }}
            onClick={event => {
              event.stopPropagation();
              onOpenDetails?.();
            }}
          >
            {detailButtonLabel}
          </button>
        </div>
      )}
    </div>
  );
};
