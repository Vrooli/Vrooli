import { useState, useEffect, useRef, useCallback } from 'react';
import { Shield, Clock, RefreshCw, AlertCircle, Cpu, HardDrive, Network, Database, Zap, X, Save, Settings } from 'lucide-react';
import { apiFetch, protoFetch } from '../../../shared/api/apiFetch';
import {
  parseGetTriggersResponse,
  parseGetCooldownStatusResponse,
  parseMetricsResponse,
  parseDetailedMetrics,
  parseProcessMonitorData,
} from '../../../shared/api/proto-contracts';
import { TriggerCondition } from '../../../types/api';
import { timestampDate } from '@bufbuild/protobuf/wkt';
import { usePolling } from '../../../shared/hooks/usePolling';
import { useToast } from '../../../shared/components/ToastProvider';
import { ToggleSwitch } from '../../../shared/components/ToggleSwitch';
import { formatDurationSeconds } from '../../../shared/utils/formatters';
import {
  buildMetricValues,
  computeTriggerProgress,
  getProgressColor,
  formatTriggerReadout,
  type SystemMetricSources,
} from '../utils/triggerMetrics';

interface TriggerCardConfig {
  id: string;
  name: string;
  description: string;
  icon: React.ElementType;
  enabled: boolean;
  autoFix: boolean;
  threshold: number;
  unit: string;
  condition: 'above' | 'below';
  currentValue?: number;
}

interface AutomaticTriggersSectionProps {
  onUpdateTrigger: (triggerId: string, config: Partial<TriggerCardConfig>) => void;
}

export const AutomaticTriggersSection = ({ onUpdateTrigger }: AutomaticTriggersSectionProps) => {
  const { showApiError } = useToast();
  const [triggers, setTriggers] = useState<TriggerCardConfig[]>([]);
  const cooldownUpdateTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [cooldownStatus, setCooldownStatus] = useState({
    cooldownPeriodSeconds: 300,
    remainingSeconds: 0,
    lastTriggerTime: new Date(),
    isReady: true
  });
  const [localCooldownValue, setLocalCooldownValue] = useState(300);
  const [loading, setLoading] = useState(true);
  const [editingTrigger, setEditingTrigger] = useState<string | null>(null);
  const [editValues, setEditValues] = useState<{ [key: string]: number }>({});

  // Update cooldown timer
  const tickCooldown = useCallback(() => {
    setCooldownStatus(prev => ({
      ...prev,
      remainingSeconds: Math.max(0, prev.remainingSeconds - 1),
      isReady: prev.remainingSeconds <= 1
    }));
  }, []);
  usePolling(tickCooldown, 1000, cooldownStatus.remainingSeconds > 0);

  // Cleanup timeout on unmount
  useEffect(() => {
    return () => {
      if (cooldownUpdateTimeoutRef.current) {
        clearTimeout(cooldownUpdateTimeoutRef.current);
      }
    };
  }, []);

  const getIconComponent = useCallback((iconName: string): React.ElementType => {
    switch (iconName) {
      case 'cpu': return Cpu;
      case 'database': return Database;
      case 'hard-drive': return HardDrive;
      case 'network': return Network;
      case 'zap': return Zap;
      default: return AlertCircle;
    }
  }, []);

  const conditionToString = (cond: TriggerCondition): 'above' | 'below' => {
    return cond === TriggerCondition.BELOW ? 'below' : 'above';
  };

  const loadData = useCallback(async (options: { suppressLoading?: boolean } = {}) => {
    const { suppressLoading = false } = options;
    try {
      if (!suppressLoading) {
        setLoading(true);
      }

      const [triggersResult, cooldownResult, metricsResult, detailedResult, processResult] = await Promise.allSettled([
        protoFetch('/investigations/triggers', parseGetTriggersResponse),
        protoFetch('/investigations/cooldown', parseGetCooldownStatusResponse),
        protoFetch('/metrics/current', parseMetricsResponse),
        protoFetch('/metrics/detailed', parseDetailedMetrics),
        protoFetch('/metrics/process-monitor', parseProcessMonitorData),
      ]);

      // Collect raw metric sources from all endpoints
      const sources: SystemMetricSources = {};
      if (metricsResult.status === 'fulfilled') {
        sources.cpuUsage = metricsResult.value.cpuUsage;
        sources.memoryUsage = metricsResult.value.memoryUsage;
        sources.tcpConnections = metricsResult.value.tcpConnections;
      }
      if (detailedResult.status === 'fulfilled') {
        const diskInfo = detailedResult.value.memoryDetails?.diskUsage;
        if (diskInfo) {
          sources.diskUsagePercent = diskInfo.percent;
        }
      }
      if (processResult.status === 'fulfilled') {
        const health = processResult.value.processHealth;
        if (health) {
          sources.anomalousProcessCount =
            (health.zombieProcesses?.length ?? 0) +
            (health.highThreadCount?.length ?? 0);
        }
      }

      const metricValues = buildMetricValues(sources);

      if (triggersResult.status === 'fulfilled') {
        const triggersMap = triggersResult.value.triggers;
        const uiTriggers: TriggerCardConfig[] = Object.values(triggersMap).map((trigger) => ({
          id: trigger.id,
          name: trigger.name,
          description: trigger.description,
          icon: getIconComponent(trigger.icon),
          enabled: trigger.enabled,
          autoFix: trigger.autoFix,
          threshold: trigger.threshold,
          unit: trigger.unit,
          condition: conditionToString(trigger.condition),
          currentValue: metricValues[trigger.id],
        }));
        setTriggers(uiTriggers);
      }

      if (cooldownResult.status === 'fulfilled') {
        const cooldown = cooldownResult.value.cooldown;
        if (cooldown) {
          setCooldownStatus({
            cooldownPeriodSeconds: cooldown.cooldownPeriodSeconds,
            remainingSeconds: cooldown.remainingSeconds,
            lastTriggerTime: cooldown.lastTriggerTime ? timestampDate(cooldown.lastTriggerTime) : new Date(),
            isReady: cooldown.isReady,
          });
          setLocalCooldownValue(cooldown.cooldownPeriodSeconds);
        }
      }
    } catch (error) {
      if (!suppressLoading) showApiError(error);
    } finally {
      if (!suppressLoading) {
        setLoading(false);
      }
    }
  }, [getIconComponent, showApiError]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const refreshTriggerData = useCallback(() => {
    void loadData({ suppressLoading: true });
  }, [loadData]);
  usePolling(refreshTriggerData, 5000);

  const handleToggleTrigger = async (triggerId: string) => {
    try {
      const trigger = triggers.find(t => t.id === triggerId);
      if (!trigger) return;

      await apiFetch(`/investigations/triggers/${triggerId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ enabled: !trigger.enabled }),
      });

      setTriggers(prev => prev.map(t =>
        t.id === triggerId ? { ...t, enabled: !t.enabled } : t
      ));
      onUpdateTrigger(triggerId, { enabled: !trigger.enabled });
    } catch (error) {
      showApiError(error);
    }
  };

  const handleToggleAutoFix = async (triggerId: string) => {
    try {
      const trigger = triggers.find(t => t.id === triggerId);
      if (!trigger) return;

      await apiFetch(`/investigations/triggers/${triggerId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ auto_fix: !trigger.autoFix }),
      });

      setTriggers(prev => prev.map(t =>
        t.id === triggerId ? { ...t, autoFix: !t.autoFix } : t
      ));
      onUpdateTrigger(triggerId, { autoFix: !trigger.autoFix });
    } catch (error) {
      showApiError(error);
    }
  };

  const handleUpdateCooldownPeriod = async (newPeriodSeconds: number) => {
    try {
      await apiFetch('/investigations/cooldown/period', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ cooldown_period_seconds: newPeriodSeconds }),
      });

      setCooldownStatus(prev => ({
        ...prev,
        cooldownPeriodSeconds: newPeriodSeconds,
      }));
    } catch (error) {
      showApiError(error);
      setLocalCooldownValue(cooldownStatus.cooldownPeriodSeconds);
    }
  };

  const handleUpdateTriggerThreshold = async (triggerId: string, newThreshold: number) => {
    try {
      await apiFetch(`/investigations/triggers/${triggerId}/threshold`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ threshold: newThreshold }),
      });

      setTriggers(prev => prev.map(t =>
        t.id === triggerId ? { ...t, threshold: newThreshold } : t
      ));
      setEditingTrigger(null);
      setEditValues({});
    } catch (error) {
      showApiError(error);
    }
  };

  const handleResetCooldown = async () => {
    try {
      await apiFetch('/investigations/cooldown/reset', {
        method: 'POST',
      });

      setCooldownStatus(prev => ({
        ...prev,
        remainingSeconds: 0,
        isReady: true,
      }));
    } catch (error) {
      showApiError(error);
    }
  };

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  if (loading) {
    return (
      <div className="trigger-section text-center">
        <span className="text-sm text-muted">Loading triggers...</span>
      </div>
    );
  }

  return (
    <div className="trigger-section">
      <h3 className="section-heading" style={{ margin: '0 0 var(--spacing-md) 0' }}>Automatic Triggers</h3>

      {/* Cooldown Controls */}
      <div className="cooldown-inline">
        <Clock size={14} className="text-muted" />
        <span className="text-sm font-bold">Cooldown:</span>
        <input
          type="range"
          min="60"
          max="3600"
          step="60"
          value={localCooldownValue}
          onChange={(e) => {
            const newValue = parseInt(e.target.value);
            setLocalCooldownValue(newValue);
            if (cooldownUpdateTimeoutRef.current) {
              clearTimeout(cooldownUpdateTimeoutRef.current);
            }
            cooldownUpdateTimeoutRef.current = setTimeout(() => {
              handleUpdateCooldownPeriod(newValue);
            }, 500);
          }}
        />
        <span className="text-sm font-mono text-success">{formatDurationSeconds(localCooldownValue)}</span>

        <div className="cooldown-actions">
          {cooldownStatus.remainingSeconds > 0 ? (
            <>
              <span className="text-sm text-warning">{formatTime(cooldownStatus.remainingSeconds)}</span>
              <button
                className="btn btn-secondary text-xs"
                onClick={handleResetCooldown}
                style={{ padding: 'var(--spacing-xs) var(--spacing-sm)' }}
              >
                <RefreshCw size={12} />
                Reset
              </button>
            </>
          ) : (
            <span className="text-sm text-success">Ready</span>
          )}
        </div>
      </div>

      {/* Triggers Grid */}
      <div className="trigger-grid">
        {triggers.map(trigger => {
          const Icon = trigger.icon;
          const pct = computeTriggerProgress(trigger.currentValue, trigger.threshold, trigger.condition, trigger.unit);
          return (
            <div
              key={trigger.id}
              className={`trigger-card${trigger.enabled ? ' trigger-card-enabled' : ''}`}
              style={{ flexDirection: 'column', alignItems: 'stretch', padding: 0 }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-md)', padding: 'var(--spacing-sm) var(--spacing-md)' }}>
                <Icon size={16} className={trigger.enabled ? 'text-success' : 'text-muted'} style={{ flexShrink: 0 }} />

                <div className="trigger-card-content">
                  <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-sm)' }}>
                    <span className="text-sm font-bold">{trigger.name}</span>
                    {editingTrigger === trigger.id ? (
                      <div className="trigger-threshold-edit">
                        <span className="text-xs">{trigger.condition === 'above' ? '>' : '<'}</span>
                        <input
                          type="number"
                          value={editValues[trigger.id] ?? trigger.threshold}
                          onChange={(e) => setEditValues({ ...editValues, [trigger.id]: parseFloat(e.target.value) })}
                          className="trigger-threshold-input"
                          onKeyDown={(e) => {
                            if (e.key === 'Enter') {
                              handleUpdateTriggerThreshold(trigger.id, editValues[trigger.id] ?? trigger.threshold);
                            } else if (e.key === 'Escape') {
                              setEditingTrigger(null);
                              setEditValues({});
                            }
                          }}
                        />
                        <span className="text-xs">{trigger.unit}</span>
                        <button
                          className="btn-icon text-success"
                          onClick={() => handleUpdateTriggerThreshold(trigger.id, editValues[trigger.id] ?? trigger.threshold)}
                          title="Save"
                        >
                          <Save size={14} />
                        </button>
                        <button
                          className="btn-icon text-error"
                          onClick={() => { setEditingTrigger(null); setEditValues({}); }}
                          title="Cancel"
                        >
                          <X size={14} />
                        </button>
                      </div>
                    ) : (
                      <>
                        <span className="text-xs font-mono text-warning">
                          {trigger.condition === 'above' ? '>' : '<'} {trigger.threshold}{trigger.unit}
                        </span>
                        <button
                          className="btn-icon"
                          onClick={() => {
                            setEditingTrigger(trigger.id);
                            setEditValues({ [trigger.id]: trigger.threshold });
                          }}
                          title="Configure threshold"
                        >
                          <Settings size={14} />
                        </button>
                      </>
                    )}
                  </div>
                  {trigger.description && (
                    <span className="text-xs text-muted" style={{ lineHeight: 1.3 }}>{trigger.description}</span>
                  )}
                </div>

                <div className="trigger-card-actions">
                  <button
                    className={`btn-icon${trigger.autoFix && trigger.enabled ? ' text-success' : ''}`}
                    onClick={() => trigger.enabled && handleToggleAutoFix(trigger.id)}
                    disabled={!trigger.enabled}
                    title={trigger.autoFix ? 'Disable auto-fix' : 'Enable auto-fix'}
                    style={{ opacity: trigger.enabled ? 1 : 0.4 }}
                  >
                    <Shield size={14} />
                  </button>

                  <ToggleSwitch
                    checked={trigger.enabled}
                    onChange={() => handleToggleTrigger(trigger.id)}
                    title={trigger.enabled ? 'Disable trigger' : 'Enable trigger'}
                  />
                </div>
              </div>

              {/* Progress bar with current/threshold readout */}
              <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-sm)', padding: '0 var(--spacing-md) var(--spacing-xs)' }}>
                <div className="progress-bar" style={{ flex: 1, borderRadius: 'var(--radius-full)', height: 4 }}>
                  <div
                    className="progress-fill"
                    style={{
                      width: `${pct * 100}%`,
                      background: getProgressColor(pct),
                    }}
                  />
                </div>
                <span className="text-xs font-mono" style={{ color: getProgressColor(pct), flexShrink: 0 }}>
                  {formatTriggerReadout(trigger.currentValue, trigger.threshold, trigger.unit)}
                </span>
              </div>
            </div>
          );
        })}
      </div>

      {/* Warning Note */}
      <div className="warning-box text-xs" style={{ marginTop: 'var(--spacing-md)' }}>
        Triggers respect the cooldown period. Auto-fix favors safe operations first.
      </div>
    </div>
  );
};
