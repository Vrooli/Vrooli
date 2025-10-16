import { useState, useEffect } from 'react';
import { X, Save, RotateCcw, AlertTriangle, CheckCircle, Settings, Activity } from 'lucide-react';
import { buildApiUrl } from '../../utils/apiBase';

interface SystemSettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
}

interface SystemSettings {
  active: boolean;
  metric_collection_interval: number;
  anomaly_detection_interval: number;
  threshold_check_interval: number;
  cooldown_period_seconds: number;
  cpu_threshold: number;
  memory_threshold: number;
  disk_threshold: number;
}

interface SettingsResponse {
  success: boolean;
  settings?: SystemSettings;
  error?: string;
}

const defaultSettings: SystemSettings = {
  active: false,
  metric_collection_interval: 10,
  anomaly_detection_interval: 30,
  threshold_check_interval: 20,
  cooldown_period_seconds: 300,
  cpu_threshold: 85.0,
  memory_threshold: 90.0,
  disk_threshold: 85.0,
};

export const SystemSettingsModal = ({ isOpen, onClose }: SystemSettingsModalProps) => {
  const [settings, setSettings] = useState<SystemSettings>(defaultSettings);
  const [originalSettings, setOriginalSettings] = useState<SystemSettings>(defaultSettings);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      loadSettings();
    }
  }, [isOpen]);

  const loadSettings = async () => {
    setLoading(true);
    setError(null);
    
    try {
      const response = await fetch(buildApiUrl('/api/settings'));
      const data: SettingsResponse = await response.json();
      
      if (data.success && data.settings) {
        setSettings(data.settings);
        setOriginalSettings(data.settings);
      } else {
        throw new Error(data.error || 'Failed to load settings');
      }
    } catch (err) {
      console.error('Failed to load settings:', err);
      setError(err instanceof Error ? err.message : 'Failed to load settings');
      // Use default settings on error
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
      const response = await fetch(buildApiUrl('/api/settings'), {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(settings)
      });
      
      const data: SettingsResponse = await response.json();
      
      if (data.success && data.settings) {
        setSettings(data.settings);
        setOriginalSettings(data.settings);
        setSuccessMessage('Settings saved successfully!');
        
        // Auto-hide success message after 3 seconds
        setTimeout(() => setSuccessMessage(null), 3000);
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
      const response = await fetch(buildApiUrl('/api/settings/reset'), {
        method: 'POST'
      });
      
      const data: SettingsResponse = await response.json();
      
      if (data.success && data.settings) {
        setSettings(data.settings);
        setOriginalSettings(data.settings);
        setSuccessMessage('Settings reset to defaults!');
        
        setTimeout(() => setSuccessMessage(null), 3000);
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

  const hasChanges = JSON.stringify(settings) !== JSON.stringify(originalSettings);

  const handleClose = () => {
    if (hasChanges && !confirm('You have unsaved changes. Are you sure you want to close?')) {
      return;
    }
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="modal-overlay" style={{
      position: 'fixed',
      top: 0,
      left: 0,
      right: 0,
      bottom: 0,
      background: 'rgba(0, 0, 0, 0.8)',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      zIndex: 1000,
      padding: 'var(--spacing-lg)'
    }}>
      <div className="modal-content" style={{
        background: 'var(--color-bg)',
        border: '1px solid var(--color-accent)',
        borderRadius: 'var(--border-radius-lg)',
        boxShadow: '0 20px 40px rgba(0, 255, 0, 0.2)',
        maxWidth: '600px',
        width: '100%',
        maxHeight: '90vh',
        overflow: 'auto'
      }}>
        {/* Header */}
        <div style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: 'var(--spacing-lg)',
          borderBottom: '1px solid var(--color-accent)',
          background: 'rgba(0, 255, 0, 0.02)'
        }}>
          <div style={{
            display: 'flex',
            alignItems: 'center',
            gap: 'var(--spacing-sm)'
          }}>
            <Settings size={24} style={{ color: 'var(--color-accent)' }} />
            <h2 style={{
              margin: 0,
              color: 'var(--color-text-bright)',
              fontSize: 'var(--font-size-xl)'
            }}>
              System Monitor Settings
            </h2>
          </div>
          
          <button
            onClick={handleClose}
            style={{
              background: 'none',
              border: 'none',
              color: 'var(--color-text-dim)',
              cursor: 'pointer',
              padding: 'var(--spacing-xs)',
              borderRadius: 'var(--border-radius-sm)',
              transition: 'color 0.2s'
            }}
            onMouseEnter={(e) => e.currentTarget.style.color = 'var(--color-text-bright)'}
            onMouseLeave={(e) => e.currentTarget.style.color = 'var(--color-text-dim)'}
          >
            <X size={24} />
          </button>
        </div>

        {/* Body */}
        <div style={{ padding: 'var(--spacing-lg)' }}>
          {loading && (
            <div style={{
              textAlign: 'center',
              padding: 'var(--spacing-xl)',
              color: 'var(--color-text-dim)'
            }}>
              Loading settings...
            </div>
          )}

          {error && (
            <div style={{
              display: 'flex',
              alignItems: 'center',
              gap: 'var(--spacing-sm)',
              padding: 'var(--spacing-md)',
              background: 'rgba(255, 0, 0, 0.1)',
              border: '1px solid var(--color-error)',
              borderRadius: 'var(--border-radius-md)',
              color: 'var(--color-error)',
              marginBottom: 'var(--spacing-lg)'
            }}>
              <AlertTriangle size={16} />
              {error}
            </div>
          )}

          {successMessage && (
            <div style={{
              display: 'flex',
              alignItems: 'center',
              gap: 'var(--spacing-sm)',
              padding: 'var(--spacing-md)',
              background: 'rgba(0, 255, 0, 0.1)',
              border: '1px solid var(--color-success)',
              borderRadius: 'var(--border-radius-md)',
              color: 'var(--color-success)',
              marginBottom: 'var(--spacing-lg)'
            }}>
              <CheckCircle size={16} />
              {successMessage}
            </div>
          )}

          {!loading && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-lg)' }}>
              {/* System Status Section */}
              <div>
                <h3 style={{
                  margin: '0 0 var(--spacing-md) 0',
                  color: 'var(--color-text-bright)',
                  fontSize: 'var(--font-size-lg)',
                  display: 'flex',
                  alignItems: 'center',
                  gap: 'var(--spacing-sm)'
                }}>
                  <Activity size={18} />
                  System Status
                </h3>
                
                <label style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 'var(--spacing-sm)',
                  cursor: 'pointer',
                  padding: 'var(--spacing-md)',
                  background: 'rgba(0, 0, 0, 0.3)',
                  border: '1px solid var(--color-accent)',
                  borderRadius: 'var(--border-radius-md)'
                }}>
                  <input
                    type="checkbox"
                    checked={settings.active}
                    onChange={(e) => setSettings(prev => ({ ...prev, active: e.target.checked }))}
                    style={{
                      width: '18px',
                      height: '18px',
                      accentColor: 'var(--color-success)',
                      cursor: 'pointer'
                    }}
                  />
                  <div>
                    <div style={{
                      color: 'var(--color-text-bright)',
                      fontWeight: 'bold'
                    }}>
                      System Monitor Active
                    </div>
                    <div style={{
                      color: 'var(--color-text-dim)',
                      fontSize: 'var(--font-size-sm)',
                      marginTop: 'var(--spacing-xs)'
                    }}>
                      Enable automatic monitoring, threshold checking, and anomaly detection
                    </div>
                  </div>
                </label>
              </div>

              {/* Monitoring Intervals Section */}
              <div>
                <h3 style={{
                  margin: '0 0 var(--spacing-md) 0',
                  color: 'var(--color-text-bright)',
                  fontSize: 'var(--font-size-lg)'
                }}>
                  Monitoring Intervals (seconds)
                </h3>
                
                <div style={{
                  display: 'grid',
                  gap: 'var(--spacing-md)',
                  gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))'
                }}>
                  <div>
                    <label style={{
                      display: 'block',
                      marginBottom: 'var(--spacing-xs)',
                      color: 'var(--color-text)',
                      fontSize: 'var(--font-size-sm)'
                    }}>
                      Metric Collection
                    </label>
                    <input
                      type="number"
                      min="5"
                      max="3600"
                      value={settings.metric_collection_interval}
                      onChange={(e) => setSettings(prev => ({
                        ...prev,
                        metric_collection_interval: parseInt(e.target.value) || 10
                      }))}
                      style={{
                        width: '100%',
                        padding: 'var(--spacing-sm)',
                        background: 'rgba(0, 0, 0, 0.3)',
                        border: '1px solid var(--color-text-dim)',
                        borderRadius: 'var(--border-radius-sm)',
                        color: 'var(--color-text)',
                        fontSize: 'var(--font-size-sm)'
                      }}
                    />
                  </div>
                  
                  <div>
                    <label style={{
                      display: 'block',
                      marginBottom: 'var(--spacing-xs)',
                      color: 'var(--color-text)',
                      fontSize: 'var(--font-size-sm)'
                    }}>
                      Threshold Checking
                    </label>
                    <input
                      type="number"
                      min="10"
                      max="1800"
                      value={settings.threshold_check_interval}
                      onChange={(e) => setSettings(prev => ({
                        ...prev,
                        threshold_check_interval: parseInt(e.target.value) || 20
                      }))}
                      style={{
                        width: '100%',
                        padding: 'var(--spacing-sm)',
                        background: 'rgba(0, 0, 0, 0.3)',
                        border: '1px solid var(--color-text-dim)',
                        borderRadius: 'var(--border-radius-sm)',
                        color: 'var(--color-text)',
                        fontSize: 'var(--font-size-sm)'
                      }}
                    />
                  </div>
                  
                  <div>
                    <label style={{
                      display: 'block',
                      marginBottom: 'var(--spacing-xs)',
                      color: 'var(--color-text)',
                      fontSize: 'var(--font-size-sm)'
                    }}>
                      Anomaly Detection
                    </label>
                    <input
                      type="number"
                      min="30"
                      max="7200"
                      value={settings.anomaly_detection_interval}
                      onChange={(e) => setSettings(prev => ({
                        ...prev,
                        anomaly_detection_interval: parseInt(e.target.value) || 30
                      }))}
                      style={{
                        width: '100%',
                        padding: 'var(--spacing-sm)',
                        background: 'rgba(0, 0, 0, 0.3)',
                        border: '1px solid var(--color-text-dim)',
                        borderRadius: 'var(--border-radius-sm)',
                        color: 'var(--color-text)',
                        fontSize: 'var(--font-size-sm)'
                      }}
                    />
                  </div>
                </div>
              </div>

              {/* System Thresholds Section */}
              <div>
                <h3 style={{
                  margin: '0 0 var(--spacing-md) 0',
                  color: 'var(--color-text-bright)',
                  fontSize: 'var(--font-size-lg)'
                }}>
                  Alert Thresholds (%)
                </h3>
                
                <div style={{
                  display: 'grid',
                  gap: 'var(--spacing-md)',
                  gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))'
                }}>
                  <div>
                    <label style={{
                      display: 'block',
                      marginBottom: 'var(--spacing-xs)',
                      color: 'var(--color-text)',
                      fontSize: 'var(--font-size-sm)'
                    }}>
                      CPU Usage
                    </label>
                    <input
                      type="number"
                      min="1"
                      max="100"
                      step="0.1"
                      value={settings.cpu_threshold}
                      onChange={(e) => setSettings(prev => ({
                        ...prev,
                        cpu_threshold: parseFloat(e.target.value) || 85
                      }))}
                      style={{
                        width: '100%',
                        padding: 'var(--spacing-sm)',
                        background: 'rgba(0, 0, 0, 0.3)',
                        border: '1px solid var(--color-text-dim)',
                        borderRadius: 'var(--border-radius-sm)',
                        color: 'var(--color-text)',
                        fontSize: 'var(--font-size-sm)'
                      }}
                    />
                  </div>
                  
                  <div>
                    <label style={{
                      display: 'block',
                      marginBottom: 'var(--spacing-xs)',
                      color: 'var(--color-text)',
                      fontSize: 'var(--font-size-sm)'
                    }}>
                      Memory Usage
                    </label>
                    <input
                      type="number"
                      min="1"
                      max="100"
                      step="0.1"
                      value={settings.memory_threshold}
                      onChange={(e) => setSettings(prev => ({
                        ...prev,
                        memory_threshold: parseFloat(e.target.value) || 90
                      }))}
                      style={{
                        width: '100%',
                        padding: 'var(--spacing-sm)',
                        background: 'rgba(0, 0, 0, 0.3)',
                        border: '1px solid var(--color-text-dim)',
                        borderRadius: 'var(--border-radius-sm)',
                        color: 'var(--color-text)',
                        fontSize: 'var(--font-size-sm)'
                      }}
                    />
                  </div>
                  
                  <div>
                    <label style={{
                      display: 'block',
                      marginBottom: 'var(--spacing-xs)',
                      color: 'var(--color-text)',
                      fontSize: 'var(--font-size-sm)'
                    }}>
                      Disk Usage
                    </label>
                    <input
                      type="number"
                      min="1"
                      max="100"
                      step="0.1"
                      value={settings.disk_threshold}
                      onChange={(e) => setSettings(prev => ({
                        ...prev,
                        disk_threshold: parseFloat(e.target.value) || 85
                      }))}
                      style={{
                        width: '100%',
                        padding: 'var(--spacing-sm)',
                        background: 'rgba(0, 0, 0, 0.3)',
                        border: '1px solid var(--color-text-dim)',
                        borderRadius: 'var(--border-radius-sm)',
                        color: 'var(--color-text)',
                        fontSize: 'var(--font-size-sm)'
                      }}
                    />
                  </div>
                </div>
              </div>

              {/* Investigation Settings Section */}
              <div>
                <h3 style={{
                  margin: '0 0 var(--spacing-md) 0',
                  color: 'var(--color-text-bright)',
                  fontSize: 'var(--font-size-lg)'
                }}>
                  Investigation Settings
                </h3>
                
                <div>
                  <label style={{
                    display: 'block',
                    marginBottom: 'var(--spacing-xs)',
                    color: 'var(--color-text)',
                    fontSize: 'var(--font-size-sm)'
                  }}>
                    Cooldown Period (seconds)
                  </label>
                  <input
                    type="number"
                    min="0"
                    max="86400"
                    value={settings.cooldown_period_seconds}
                    onChange={(e) => setSettings(prev => ({
                      ...prev,
                      cooldown_period_seconds: parseInt(e.target.value) || 300
                    }))}
                    style={{
                      width: '200px',
                      padding: 'var(--spacing-sm)',
                      background: 'rgba(0, 0, 0, 0.3)',
                      border: '1px solid var(--color-text-dim)',
                      borderRadius: 'var(--border-radius-sm)',
                      color: 'var(--color-text)',
                      fontSize: 'var(--font-size-sm)'
                    }}
                  />
                  <div style={{
                    color: 'var(--color-text-dim)',
                    fontSize: 'var(--font-size-xs)',
                    marginTop: 'var(--spacing-xs)'
                  }}>
                    Minimum time between automatic investigations to prevent spam
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          padding: 'var(--spacing-lg)',
          borderTop: '1px solid var(--color-accent)',
          background: 'rgba(0, 0, 0, 0.3)'
        }}>
          <button
            onClick={resetSettings}
            disabled={saving || loading}
            className="btn btn-secondary"
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 'var(--spacing-xs)',
              opacity: saving || loading ? 0.5 : 1
            }}
          >
            <RotateCcw size={16} />
            Reset to Defaults
          </button>
          
          <div style={{ display: 'flex', gap: 'var(--spacing-sm)' }}>
            <button
              onClick={handleClose}
              disabled={saving}
              className="btn btn-secondary"
              style={{ opacity: saving ? 0.5 : 1 }}
            >
              Cancel
            </button>
            
            <button
              onClick={saveSettings}
              disabled={saving || loading || !hasChanges}
              className="btn btn-primary"
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 'var(--spacing-xs)',
                opacity: saving || loading || !hasChanges ? 0.5 : 1
              }}
            >
              <Save size={16} />
              {saving ? 'Saving...' : 'Save Settings'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
