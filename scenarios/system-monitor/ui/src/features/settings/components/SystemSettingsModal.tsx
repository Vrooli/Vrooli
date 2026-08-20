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

const defaultSettings: ProtoSettings = create(SystemSettingsSchema, {
  active: false,
  metricCollectionInterval: 10,
  anomalyDetectionInterval: 30,
  thresholdCheckInterval: 20,
  cooldownPeriodSeconds: 300,
  cpuThreshold: 85.0,
  memoryThreshold: 90.0,
  diskThreshold: 85.0,
});

export const SystemSettingsModal = ({ isOpen, onClose }: SystemSettingsModalProps) => {
  const [settings, setSettings] = useState<ProtoSettings>(defaultSettings);
  const [originalSettings, setOriginalSettings] = useState<ProtoSettings>(defaultSettings);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const successTimeoutRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

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
        setSettings(data.settings);
        setOriginalSettings(data.settings);
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
        setSettings(data.settings);
        setOriginalSettings(data.settings);
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
        setSettings(data.settings);
        setOriginalSettings(data.settings);
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
              </div>
            </div>

            {/* Investigation Settings Section */}
            <div>
              <h3 className="section-heading" data-sm-style="sm-style-aa254bd4d7">
                Investigation Settings
              </h3>

              <div>
                <label className="input-label">Cooldown Period (seconds)</label>
                <input
                  type="number"
                  className="input-field"
                  min="0"
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
