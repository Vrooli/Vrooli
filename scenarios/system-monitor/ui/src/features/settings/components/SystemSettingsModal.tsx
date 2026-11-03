import { useState, useEffect, useRef, useCallback } from 'react';
import { Save, RotateCcw, AlertTriangle, CheckCircle, Settings, Activity } from 'lucide-react';
import { Modal, ModalHeader } from '../../../shared/components/Modal';
import { protoFetch } from '../../../shared/api/apiFetch';
import {
  parseGetSettingsResponse,
  parseUpdateSettingsResponse,
  parseResetSettingsResponse,
  SystemSettingsSchema,
  toJsonString,
  create,
} from '../../../shared/api/proto-contracts';
import type { MessageShape } from '@bufbuild/protobuf';

interface SystemSettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
}

type ProtoSettings = MessageShape<typeof SystemSettingsSchema>;
type NumericSettingKey = { [K in keyof ProtoSettings]: ProtoSettings[K] extends number | bigint ? K : never }[keyof ProtoSettings];

const defaultSettings: ProtoSettings = create(SystemSettingsSchema, {
  active: false,
  metricCollectionInterval: 20,
  anomalyDetectionInterval: 30,
  thresholdCheckInterval: 20,
  cooldownPeriodSeconds: 300,
  alertSustainSeconds: 300,
  cpuThreshold: 85.0,
  cpuHighPercent: 92.0,
  cpuCriticalPercent: 97.0,
  cpuEscalationCooldownSeconds: 1800,
  cpuEscalationDebounceTicks: 2,
  cpuSustainedWindowTicks: 3,
  cpuPressureThreshold: 10.0,
  memoryThreshold: 90.0,
  memoryCriticalPercent: 95.0,
  tcpConnectionsWarning: 5000,
  tcpConnectionsCritical: 20000,
  tcpGrowthWindowSeconds: 600,
  tcpGrowthFloor: 500,
  tcpGrowthMinSamples: 5,
  tcpGrowthFactor: 2,
  gpuUsageWarningPercent: 90,
  metricsSnapshotStaleIntervals: 10,
  orphanedAgentAgeSeconds: 3600,
  orphanedAgentMemoryBytes: 2147483648n,
  autoInvestigateCriticalRules: {
    metrics_snapshot: true,
    cpu_usage: true,
    memory_usage: true,
    tcp_connections: true,
    gpu_usage: true,
    memory_pressure: true,
    orphaned_agents: true,
  },
  diskThreshold: 80.0,
  diskHighPercent: 90,
  diskCriticalPercent: 95,
  diskEscalationCooldownSeconds: 1800,
  diskEscalationDebounceTicks: 2,
  diskFastFillJumpPercent: 5,
  memoryPressureSomeAvg60ElevatedPercent: 10,
  memoryPressureFullAvg60CriticalPercent: 5,
  memoryPressureSwapInElevatedBytesPerSecond: 10485760,
  memoryPressureSwapInCriticalBytesPerSecond: 52428800,
});

type NumericSettingProps = {
  label: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  suffix?: string;
  onChange: (value: number) => void;
};

function NumericSetting({ label, value, min, max, step = 1, suffix, onChange }: NumericSettingProps) {
  return (
    <div>
      <label className="input-label">
        {label}{suffix ? ` (${suffix})` : ''}
        <input
          type="number"
          className="input-field"
          min={min}
          max={max}
          step={step}
          value={value}
          onChange={(event) => { onChange(Number(event.target.value)); }}
        />
      </label>
    </div>
  );
}

export const SystemSettingsModal = ({ isOpen, onClose }: SystemSettingsModalProps) => {
  const [settings, setSettings] = useState<ProtoSettings>(defaultSettings);
  const [originalSettings, setOriginalSettings] = useState<ProtoSettings>(defaultSettings);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const successTimeoutRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const setNumber = (key: NumericSettingKey, value: number) => {
    setSettings((previous) => {
      if (key === 'orphanedAgentMemoryBytes') {
        return { ...previous, orphanedAgentMemoryBytes: BigInt(Math.round(value)) };
      }
      return { ...previous, [String(key)]: value };
    });
  };

  const showTemporarySuccess = useCallback((msg: string) => {
    if (successTimeoutRef.current) {
      clearTimeout(successTimeoutRef.current);
    }
    setSuccessMessage(msg);
    successTimeoutRef.current = setTimeout(() => { setSuccessMessage(null); }, 3000);
  }, []);

  useEffect(() => {
    return () => {
      if (successTimeoutRef.current) {
        clearTimeout(successTimeoutRef.current);
      }
    };
  }, []);

  useEffect(() => {
    if (isOpen) {
      void loadSettings();
    }
  }, [isOpen]);

  const loadSettings = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await protoFetch('/settings', parseGetSettingsResponse);
      if (data.success && data.settings) {
        const loadedSettings = {
          ...defaultSettings,
          ...data.settings,
          autoInvestigateCriticalRules: {
            ...defaultSettings.autoInvestigateCriticalRules,
            ...data.settings.autoInvestigateCriticalRules,
          },
        };
        setSettings(loadedSettings);
        setOriginalSettings(loadedSettings);
      } else {
        throw new Error(data.error || 'Failed to load settings');
      }
    } catch (err) {
      console.error('Failed to load settings:', err);
      setError(err instanceof Error ? err.message : 'Failed to load settings');
      setSettings(defaultSettings);
      setOriginalSettings(defaultSettings);
    } finally {
      setLoading(false);
    }
  };

  const saveSettings = async () => {
    setSaving(true);
    setError(null);
    setSuccessMessage(null);
    try {
      const data = await protoFetch('/settings', parseUpdateSettingsResponse, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: toJsonString(SystemSettingsSchema, settings),
      });
      if (data.success && data.settings) {
        const updatedSettings = {
          ...defaultSettings,
          ...data.settings,
          autoInvestigateCriticalRules: {
            ...defaultSettings.autoInvestigateCriticalRules,
            ...data.settings.autoInvestigateCriticalRules,
          },
        };
        setSettings(updatedSettings);
        setOriginalSettings(updatedSettings);
        showTemporarySuccess('Settings saved successfully!');
      } else {
        throw new Error(data.error || 'Failed to save settings');
      }
    } catch (err) {
      console.error('Failed to save settings:', err);
      setError(err instanceof Error ? err.message : 'Failed to save settings');
    } finally {
      setSaving(false);
    }
  };

  const resetSettings = async () => {
    if (!confirm('Are you sure you want to reset all settings to defaults? This cannot be undone.')) {
      return;
    }

    setLoading(true);
    setError(null);
    setSuccessMessage(null);

    try {
      const data = await protoFetch('/settings/reset', parseResetSettingsResponse, {
        method: 'POST'
      });

      if (data.success && data.settings) {
        const reset = {
          ...defaultSettings,
          ...data.settings,
          autoInvestigateCriticalRules: {
            ...defaultSettings.autoInvestigateCriticalRules,
            ...data.settings.autoInvestigateCriticalRules,
          },
        };
        setSettings(reset);
        setOriginalSettings(reset);
        showTemporarySuccess('Settings reset to defaults!');
      } else {
        throw new Error(data.error || 'Failed to reset settings');
      }
    } catch (err) {
      console.error('Failed to reset settings:', err);
      setError(err instanceof Error ? err.message : 'Failed to reset settings');
    } finally {
      setLoading(false);
    }
  };

  const hasChanges = toJsonString(SystemSettingsSchema, settings) !== toJsonString(SystemSettingsSchema, originalSettings);

  const handleClose = () => {
    if (hasChanges && !confirm('You have unsaved changes. Are you sure you want to close?')) {
      return;
    }
    onClose();
  };

  return (
    <Modal isOpen={isOpen} onClose={handleClose} ariaLabel="System settings" className="modal-sm">
      <ModalHeader onClose={handleClose}>
        <div className="icon-text">
          <Settings size={24} data-sm-style="sm-style-392c7463c7" />
          <h2 data-sm-style="sm-style-28fbfae5b9">
            System Monitor Settings
          </h2>
        </div>
      </ModalHeader>

      {/* Body */}
      <div data-sm-style="sm-style-fe9d9d2f6f">
        {loading && (
          <div data-sm-style="sm-style-586c42c087">
            Loading settings...
          </div>
        )}

        {error && (
          <div className="error-banner" data-sm-style="sm-style-b439d4fdb0">
            <AlertTriangle size={16} />
            {error}
          </div>
        )}

        {successMessage && (
          <div className="success-banner" data-sm-style="sm-style-b439d4fdb0">
            <CheckCircle size={16} />
            {successMessage}
          </div>
        )}

        {!loading && (
          <div className="flex-col-gap-lg">
            {/* System Status Section */}
            <div>
              <h3 className="icon-text section-heading" data-sm-style="sm-style-aa254bd4d7">
                <Activity size={18} />
                System Status
              </h3>

              <label data-sm-style="sm-style-53626c1c9a">
                <input
                  type="checkbox"
                  checked={settings.active}
                  onChange={(e) => { setSettings(prev => ({ ...prev, active: e.target.checked })); }}
                  data-sm-style="sm-style-61e095ca60"
                />
                <div>
                  <div data-sm-style="sm-style-65dd98adc6">
                    System Monitor Active
                  </div>
                  <div data-sm-style="sm-style-aba19f690c">
                    Enable automatic monitoring, threshold checking, and anomaly detection
                  </div>
                </div>
              </label>
            </div>

            {/* Monitoring Intervals Section */}
            <div>
              <h3 className="section-heading" data-sm-style="sm-style-aa254bd4d7">
                Monitoring Intervals (seconds)
              </h3>

              <div data-sm-style="sm-style-4ac5de75de">
                <div>
                  <label className="input-label">Metric Collection</label>
                  <input
                    type="number"
                    className="input-field"
                    min="5"
                    max="3600"
                    value={settings.metricCollectionInterval}
                    onChange={(e) => { setSettings(prev => ({
                      ...prev,
                      metricCollectionInterval: parseInt(e.target.value) || 10
                    })); }}
                  />
                </div>

                <div>
                  <label className="input-label">Threshold Checking</label>
                  <input
                    type="number"
                    className="input-field"
                    min="10"
                    max="1800"
                    value={settings.thresholdCheckInterval}
                    onChange={(e) => { setSettings(prev => ({
                      ...prev,
                      thresholdCheckInterval: parseInt(e.target.value) || 20
                    })); }}
                  />
                </div>

                <div>
                  <label className="input-label">Anomaly Detection</label>
                  <input
                    type="number"
                    className="input-field"
                    min="30"
                    max="7200"
                    value={settings.anomalyDetectionInterval}
                    onChange={(e) => { setSettings(prev => ({
                      ...prev,
                      anomalyDetectionInterval: parseInt(e.target.value) || 30
                    })); }}
                  />
                </div>
              </div>
            </div>

            {/* System Thresholds Section */}
            <div>
              <h3 className="section-heading" data-sm-style="sm-style-aa254bd4d7">
                Alert Thresholds (%)
              </h3>

              <div data-sm-style="sm-style-52943ea1ec">
                <div>
                  <label className="input-label">CPU Usage</label>
                  <input
                    type="number"
                    className="input-field"
                    min="1"
                    max="100"
                    step="0.1"
                    value={settings.cpuThreshold}
                    onChange={(e) => { setSettings(prev => ({
                      ...prev,
                      cpuThreshold: parseFloat(e.target.value) || 85
                    })); }}
                  />
                </div>

                <div>
                  <label className="input-label">Memory Usage</label>
                  <input
                    type="number"
                    className="input-field"
                    min="1"
                    max="100"
                    step="0.1"
                    value={settings.memoryThreshold}
                    onChange={(e) => { setSettings(prev => ({
                      ...prev,
                      memoryThreshold: parseFloat(e.target.value) || 90
                    })); }}
                  />
                </div>

                <div>
                  <label className="input-label">CPU High Band</label>
                  <input type="number" className="input-field" min="1" max="100" step="0.1" value={settings.cpuHighPercent} onChange={(e) => { setSettings(prev => ({ ...prev, cpuHighPercent: parseFloat(e.target.value) || 92 })); }} />
                </div>

                <div>
                  <label className="input-label">CPU Critical Band</label>
                  <input type="number" className="input-field" min="1" max="100" step="0.1" value={settings.cpuCriticalPercent} onChange={(e) => { setSettings(prev => ({ ...prev, cpuCriticalPercent: parseFloat(e.target.value) || 97 })); }} />
                </div>

                <div>
                  <label className="input-label">CPU Sustained Samples</label>
                  <input type="number" className="input-field" min="1" max="100" value={settings.cpuSustainedWindowTicks} onChange={(e) => { setSettings(prev => ({ ...prev, cpuSustainedWindowTicks: parseInt(e.target.value) || 3 })); }} />
                </div>

                <div>
                  <label className="input-label">CPU Debounce Samples</label>
                  <input type="number" className="input-field" min="1" max="100" value={settings.cpuEscalationDebounceTicks} onChange={(e) => { setSettings(prev => ({ ...prev, cpuEscalationDebounceTicks: parseInt(e.target.value) || 2 })); }} />
                </div>

                <div>
                  <label className="input-label">CPU Cooldown (seconds)</label>
                  <input type="number" className="input-field" min="0" max="86400" value={settings.cpuEscalationCooldownSeconds} onChange={(e) => { setSettings(prev => ({ ...prev, cpuEscalationCooldownSeconds: parseInt(e.target.value) || 1800 })); }} />
                </div>

                <div>
                  <label className="input-label">Disk Usage</label>
                  <input
                    type="number"
                    className="input-field"
                    min="1"
                    max="100"
                    step="0.1"
                    value={settings.diskThreshold}
                    onChange={(e) => { setSettings(prev => ({
                      ...prev,
                      diskThreshold: parseFloat(e.target.value) || 85
                    })); }}
                  />
                </div>

                <NumericSetting label="Memory Critical" value={settings.memoryCriticalPercent} min={1} max={100} step={0.1} suffix="%" onChange={(value) => { setNumber('memoryCriticalPercent', value); }} />
                <NumericSetting label="TCP Connections Warning" value={settings.tcpConnectionsWarning} min={1} max={10000000} onChange={(value) => { setNumber('tcpConnectionsWarning', value); }} />
                <NumericSetting label="TCP Connections Critical" value={settings.tcpConnectionsCritical} min={1} max={10000000} onChange={(value) => { setNumber('tcpConnectionsCritical', value); }} />
                <NumericSetting label="TCP Growth Window" value={settings.tcpGrowthWindowSeconds} min={60} max={86400} suffix="seconds" onChange={(value) => { setNumber('tcpGrowthWindowSeconds', value); }} />
                <NumericSetting label="TCP Growth Floor" value={settings.tcpGrowthFloor} min={1} max={10000000} suffix="connections" onChange={(value) => { setNumber('tcpGrowthFloor', value); }} />
                <NumericSetting label="TCP Growth Samples" value={settings.tcpGrowthMinSamples} min={2} max={100} onChange={(value) => { setNumber('tcpGrowthMinSamples', value); }} />
                <NumericSetting label="TCP Growth Factor" value={settings.tcpGrowthFactor} min={1.1} max={100} step={0.1} onChange={(value) => { setNumber('tcpGrowthFactor', value); }} />
                <NumericSetting label="GPU Usage Warning" value={settings.gpuUsageWarningPercent} min={1} max={100} step={0.1} suffix="%" onChange={(value) => { setNumber('gpuUsageWarningPercent', value); }} />
                <NumericSetting label="Stale Snapshot Limit" value={settings.metricsSnapshotStaleIntervals} min={2} max={1000} suffix="collection intervals" onChange={(value) => { setNumber('metricsSnapshotStaleIntervals', value); }} />
                <NumericSetting label="Orphaned Agent Age" value={settings.orphanedAgentAgeSeconds} min={60} max={31536000} suffix="seconds" onChange={(value) => { setNumber('orphanedAgentAgeSeconds', value); }} />
                <NumericSetting label="Orphaned Agent Memory" value={Number(settings.orphanedAgentMemoryBytes) / 1073741824} min={0.001} max={1048576} step={0.1} suffix="GiB" onChange={(value) => { setNumber('orphanedAgentMemoryBytes', Math.round(value * 1073741824)); }} />
                <NumericSetting label="Disk High Band" value={settings.diskHighPercent} min={1} max={100} step={0.1} suffix="%" onChange={(value) => { setNumber('diskHighPercent', value); }} />
                <NumericSetting label="Disk Critical Band" value={settings.diskCriticalPercent} min={1} max={100} step={0.1} suffix="%" onChange={(value) => { setNumber('diskCriticalPercent', value); }} />
                <NumericSetting label="Disk Escalation Cooldown" value={settings.diskEscalationCooldownSeconds} min={0} max={86400} suffix="seconds" onChange={(value) => { setNumber('diskEscalationCooldownSeconds', value); }} />
                <NumericSetting label="Disk Debounce Samples" value={settings.diskEscalationDebounceTicks} min={1} max={100} onChange={(value) => { setNumber('diskEscalationDebounceTicks', value); }} />
                <NumericSetting label="Disk Fast Fill Jump" value={settings.diskFastFillJumpPercent} min={0} max={100} step={0.1} suffix="percentage points" onChange={(value) => { setNumber('diskFastFillJumpPercent', value); }} />
                <NumericSetting label="Memory Pressure Elevated PSI" value={settings.memoryPressureSomeAvg60ElevatedPercent} min={0} max={100} step={0.1} suffix="%" onChange={(value) => { setNumber('memoryPressureSomeAvg60ElevatedPercent', value); }} />
                <NumericSetting label="Memory Pressure Critical PSI" value={settings.memoryPressureFullAvg60CriticalPercent} min={0} max={100} step={0.1} suffix="%" onChange={(value) => { setNumber('memoryPressureFullAvg60CriticalPercent', value); }} />
                <NumericSetting label="Memory Swap In Elevated" value={settings.memoryPressureSwapInElevatedBytesPerSecond / 1048576} min={0} max={100000} step={0.1} suffix="MiB/s" onChange={(value) => { setNumber('memoryPressureSwapInElevatedBytesPerSecond', Math.round(value * 1048576)); }} />
                <NumericSetting label="Memory Swap In Critical" value={settings.memoryPressureSwapInCriticalBytesPerSecond / 1048576} min={0} max={100000} step={0.1} suffix="MiB/s" onChange={(value) => { setNumber('memoryPressureSwapInCriticalBytesPerSecond', Math.round(value * 1048576)); }} />
              </div>
            </div>

            {/* Investigation Settings Section */}
            <div>
              <h3 className="section-heading" data-sm-style="sm-style-aa254bd4d7">
                Investigation Settings
              </h3>

              <p className="text-xs text-muted-foreground">Alert notification categories are enabled and muted in Notification Hub. These settings control detection thresholds and investigation timing.</p>

              <div className="flex-col-gap-sm" aria-label="Automatic investigations">
                <h4 className="text-sm font-medium">Investigate sustained critical alerts</h4>
                <p className="text-xs text-muted-foreground">Selected rules run their native catalog investigation after the critical condition remains sustained. This does not start an Agent Manager run or perform remediation.</p>
                {([
                  ['metrics_snapshot', 'Metrics collection unavailable'],
                  ['cpu_usage', 'CPU usage'],
                  ['memory_usage', 'Memory usage'],
                  ['tcp_connections', 'TCP connections'],
                  ['gpu_usage', 'GPU usage'],
                  ['memory_pressure', 'Memory pressure'],
                  ['orphaned_agents', 'Orphaned agents'],
                ] as const).map(([ruleId, label]) => (
                  <label key={ruleId} className="text-sm flex items-center gap-2">
                    <input
                      type="checkbox"
                      checked={settings.autoInvestigateCriticalRules[ruleId] ?? true}
                      onChange={(event) => {
                        setSettings((previous) => ({
                          ...previous,
                          autoInvestigateCriticalRules: {
                            ...previous.autoInvestigateCriticalRules,
                            [ruleId]: event.target.checked,
                          },
                        }));
                      }}
                    />
                    {label}
                  </label>
                ))}
              </div>

              <NumericSetting label="Alert Sustain and Clear Duration" value={settings.alertSustainSeconds} min={1} max={86400} suffix="seconds" onChange={(value) => { setNumber('alertSustainSeconds', value); }} />

              <div>
                <label className="input-label">Cooldown Period (seconds)</label>
                <input
                  type="number"
                  className="input-field"
                  min="1"
                  max="86400"
                  value={settings.cooldownPeriodSeconds}
                  onChange={(e) => { setSettings(prev => ({
                    ...prev,
                    cooldownPeriodSeconds: parseInt(e.target.value) || 300
                  })); }}
                  data-sm-style="sm-style-28702649fb"
                />
                <div data-sm-style="sm-style-583863c6ac">
                  Minimum time between automatic investigations to prevent spam
                </div>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Footer */}
      <div data-sm-style="sm-style-e3e852fc99">
        <button
          onClick={() => { void resetSettings(); }}
          disabled={saving || loading}
          className={`btn btn-secondary icon-text icon-text-xs ${saving || loading ? 'is-disabled' : ''}`}
        >
          <RotateCcw size={16} />
          Reset to Defaults
        </button>

        <div data-sm-style="sm-style-2f94a4de27">
          <button
            onClick={handleClose}
            disabled={saving}
            className={`btn btn-secondary ${saving ? 'is-disabled' : ''}`}
          >
            Cancel
          </button>

          <button
            onClick={() => { void saveSettings(); }}
            disabled={saving || loading || !hasChanges}
            className={`btn btn-primary icon-text icon-text-xs ${saving || loading || !hasChanges ? 'is-disabled' : ''}`}
          >
            <Save size={16} />
            {saving ? 'Saving...' : 'Save Settings'}
          </button>
        </div>
      </div>
    </Modal>
  );
};
