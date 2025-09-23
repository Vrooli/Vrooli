import { useState, useEffect, useRef, useCallback } from 'react';
import { Settings, Shield, Clock, RefreshCw, AlertCircle, Cpu, HardDrive, Network, Database, Zap, X, Save } from 'lucide-react';

interface TriggerConfig {
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
  progress?: number;
}

interface TriggerApiResponse {
  id: string;
  name: string;
  description: string;
  icon: string;
  enabled: boolean;
  auto_fix: boolean;
  threshold: number;
  unit: string;
  condition: 'above' | 'below';
  current_value?: number;
  progress?: number;
}

interface AutomaticTriggersSectionProps {
  onUpdateTrigger: (triggerId: string, config: Partial<TriggerConfig>) => void;
}

export const AutomaticTriggersSection = ({ onUpdateTrigger }: AutomaticTriggersSectionProps) => {
  const [triggers, setTriggers] = useState<TriggerConfig[]>([]);
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
  useEffect(() => {
    if (cooldownStatus.remainingSeconds <= 0) return;
    
    const timer = setInterval(() => {
      setCooldownStatus(prev => ({
        ...prev,
        remainingSeconds: Math.max(0, prev.remainingSeconds - 1),
        isReady: prev.remainingSeconds <= 1
      }));
    }, 1000);

    return () => clearInterval(timer);
  }, [cooldownStatus.remainingSeconds]);

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

  const loadData = useCallback(async (options: { suppressLoading?: boolean } = {}) => {
    const { suppressLoading = false } = options;
    try {
      if (!suppressLoading) {
        setLoading(true);
      }

      // Load triggers and cooldown status
      const [triggersResponse, cooldownResponse] = await Promise.all([
        fetch('/api/investigations/triggers'),
        fetch('/api/investigations/cooldown')
      ]);
      
      if (triggersResponse.ok) {
        const triggersData: Record<string, TriggerApiResponse> = await triggersResponse.json();
        // Convert API data to UI format
        const uiTriggers = Object.values(triggersData).map((trigger) => ({
          id: trigger.id,
          name: trigger.name,
          description: trigger.description,
          icon: getIconComponent(trigger.icon),
          enabled: trigger.enabled,
          autoFix: trigger.auto_fix,
          threshold: trigger.threshold,
          unit: trigger.unit,
          condition: trigger.condition,
          currentValue: typeof trigger.current_value === 'number' ? trigger.current_value : undefined,
          progress: typeof trigger.progress === 'number' ? trigger.progress : undefined
        }));
        setTriggers(uiTriggers as TriggerConfig[]);
      }
      
      if (cooldownResponse.ok) {
        const cooldownData = await cooldownResponse.json();
        setCooldownStatus({
          cooldownPeriodSeconds: cooldownData.cooldown_period_seconds,
          remainingSeconds: cooldownData.remaining_seconds,
          lastTriggerTime: new Date(cooldownData.last_trigger_time),
          isReady: cooldownData.is_ready
        });
        setLocalCooldownValue(cooldownData.cooldown_period_seconds);
      }
    } catch (error) {
      console.error('Failed to load trigger data:', error);
    } finally {
      if (!suppressLoading) {
        setLoading(false);
      }
    }
  }, [getIconComponent]);

  // Load data from API on mount and refresh periodically for live progress updates
  useEffect(() => {
    loadData();

    const refreshInterval = setInterval(() => {
      loadData({ suppressLoading: true });
    }, 5000);

    return () => clearInterval(refreshInterval);
  }, [loadData]);

  const handleToggleTrigger = async (triggerId: string) => {
    try {
      const trigger = triggers.find(t => t.id === triggerId);
      if (!trigger) return;
      
      const response = await fetch(`/api/investigations/triggers/${triggerId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ enabled: !trigger.enabled })
      });
      
      if (response.ok) {
        setTriggers(prev => prev.map(t => 
          t.id === triggerId ? { ...t, enabled: !t.enabled } : t
        ));
        onUpdateTrigger(triggerId, { enabled: !trigger.enabled });
      } else {
        console.error('Failed to update trigger');
      }
    } catch (error) {
      console.error('Failed to update trigger:', error);
    }
  };

  const handleToggleAutoFix = async (triggerId: string) => {
    try {
      const trigger = triggers.find(t => t.id === triggerId);
      if (!trigger) return;
      
      const response = await fetch(`/api/investigations/triggers/${triggerId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ auto_fix: !trigger.autoFix })
      });
      
      if (response.ok) {
        setTriggers(prev => prev.map(t => 
          t.id === triggerId ? { ...t, autoFix: !t.autoFix } : t
        ));
        onUpdateTrigger(triggerId, { autoFix: !trigger.autoFix });
      } else {
        console.error('Failed to update trigger auto-fix');
      }
    } catch (error) {
      console.error('Failed to update trigger auto-fix:', error);
    }
  };

  const handleUpdateCooldownPeriod = async (newPeriodSeconds: number) => {
    try {
      const response = await fetch('/api/investigations/cooldown/period', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          cooldown_period_seconds: newPeriodSeconds
        })
      });
      
      if (response.ok) {
        setCooldownStatus(prev => ({
          ...prev,
          cooldownPeriodSeconds: newPeriodSeconds
        }));
      } else {
        console.error('Failed to update cooldown period:', response.status);
        // Revert local value on failure
        setLocalCooldownValue(cooldownStatus.cooldownPeriodSeconds);
      }
    } catch (error) {
      console.error('Failed to update cooldown period:', error);
      // Revert local value on error
      setLocalCooldownValue(cooldownStatus.cooldownPeriodSeconds);
    }
  };

  const handleUpdateTriggerThreshold = async (triggerId: string, newThreshold: number) => {
    try {
      const response = await fetch(`/api/investigations/triggers/${triggerId}/threshold`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ threshold: newThreshold })
      });
      
      if (response.ok) {
        setTriggers(prev => prev.map(t => 
          t.id === triggerId ? { ...t, threshold: newThreshold } : t
        ));
        setEditingTrigger(null);
        setEditValues({});
      } else {
        console.error('Failed to update trigger threshold');
      }
    } catch (error) {
      console.error('Failed to update trigger threshold:', error);
    }
  };

  const handleResetCooldown = async () => {
    try {
      const response = await fetch('/api/investigations/cooldown/reset', {
        method: 'POST'
      });
      
      if (response.ok) {
        setCooldownStatus(prev => ({
          ...prev,
          remainingSeconds: 0,
          isReady: true
        }));
      } else {
        console.error('Failed to reset cooldown');
      }
    } catch (error) {
      console.error('Failed to reset cooldown:', error);
    }
  };

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  const formatCooldownPeriod = (seconds: number) => {
    if (seconds < 60) return `${seconds}s`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
    return `${Math.floor(seconds / 3600)}h`;
  };

  if (loading) {
    return (
      <div className="automatic-triggers-section" style={{
        background: 'rgba(0, 255, 0, 0.03)',
        border: '1px solid var(--color-accent)',
        borderRadius: 'var(--border-radius-md)',
        padding: 'var(--spacing-lg)',
        marginBottom: 'var(--spacing-xl)',
        textAlign: 'center',
        color: 'var(--color-text-dim)'
      }}>
        Loading trigger configuration...
      </div>
    );
  }

  return (
    <div className="automatic-triggers-section" style={{
      background: 'rgba(0, 255, 0, 0.03)',
      border: '1px solid var(--color-accent)',
      borderRadius: 'var(--border-radius-md)',
      padding: 'var(--spacing-lg)',
      marginBottom: 'var(--spacing-xl)'
    }}>
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: 'var(--spacing-lg)' }}>
        <Settings size={48} style={{ 
          color: 'var(--color-accent)', 
          flexShrink: 0,
          filter: 'drop-shadow(0 0 10px rgba(0, 255, 0, 0.3))'
        }} />
        
        <div style={{ flex: 1 }}>
          <h3 style={{ 
            margin: '0 0 var(--spacing-sm) 0',
            color: 'var(--color-text-bright)',
            fontSize: 'var(--font-size-lg)'
          }}>
            Automatic Investigation Triggers
          </h3>
          
          <p style={{ 
            margin: '0 0 var(--spacing-lg) 0',
            color: 'var(--color-text-dim)',
            fontSize: 'var(--font-size-sm)',
            lineHeight: '1.5'
          }}>
            Configure conditions that automatically spawn investigation agents. 
            Each trigger can be individually enabled and configured for auto-fix mode.
          </p>

          {/* Cooldown Controls */}
          <div style={{
            display: 'flex',
            alignItems: 'center',
            gap: 'var(--spacing-lg)',
            marginBottom: 'var(--spacing-lg)',
            padding: 'var(--spacing-md)',
            background: 'rgba(0, 0, 0, 0.3)',
            borderRadius: 'var(--border-radius-sm)',
            border: '1px solid rgba(0, 255, 0, 0.1)'
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-sm)' }}>
              <Clock size={16} style={{ color: 'var(--color-accent)' }} />
              <span style={{ 
                color: 'var(--color-text)',
                fontSize: 'var(--font-size-sm)',
                fontWeight: 'bold'
              }}>
                Cooldown Period:
              </span>
            </div>

            <div style={{ flex: 1, display: 'flex', alignItems: 'center', gap: 'var(--spacing-md)' }}>
              <input
                type="range"
                min="60"
                max="3600"
                step="60"
                value={localCooldownValue}
                onChange={(e) => {
                  const newValue = parseInt(e.target.value);
                  // Update local state immediately for responsive UI
                  setLocalCooldownValue(newValue);
                  // Clear any existing timeout
                  if (cooldownUpdateTimeoutRef.current) {
                    clearTimeout(cooldownUpdateTimeoutRef.current);
                  }
                  // Set new timeout for API call
                  cooldownUpdateTimeoutRef.current = setTimeout(() => {
                    handleUpdateCooldownPeriod(newValue);
                  }, 500);
                }}
                style={{
                  flex: 1,
                  height: '6px',
                  background: 'rgba(0, 255, 0, 0.2)',
                  borderRadius: '3px',
                  outline: 'none',
                  WebkitAppearance: 'none',
                  cursor: 'pointer'
                }}
              />
              <span style={{
                color: 'var(--color-success)',
                fontSize: 'var(--font-size-sm)',
                fontFamily: 'var(--font-family-mono)',
                minWidth: '50px'
              }}>
                {formatCooldownPeriod(localCooldownValue)}
              </span>
            </div>

            <div style={{
              display: 'flex',
              alignItems: 'center',
              gap: 'var(--spacing-md)',
              paddingLeft: 'var(--spacing-md)',
              borderLeft: '1px solid rgba(0, 255, 0, 0.2)'
            }}>
              {cooldownStatus.remainingSeconds > 0 ? (
                <>
                  <span style={{
                    color: 'var(--color-warning)',
                    fontSize: 'var(--font-size-sm)'
                  }}>
                    Cooldown: {formatTime(cooldownStatus.remainingSeconds)}
                  </span>
                  <button
                    className="btn btn-secondary"
                    onClick={handleResetCooldown}
                    style={{
                      padding: 'var(--spacing-xs) var(--spacing-sm)',
                      fontSize: 'var(--font-size-xs)',
                      display: 'flex',
                      alignItems: 'center',
                      gap: 'var(--spacing-xs)'
                    }}
                  >
                    <RefreshCw size={12} />
                    RESET
                  </button>
                </>
              ) : (
                <span style={{
                  color: 'var(--color-success)',
                  fontSize: 'var(--font-size-sm)'
                }}>
                  Ready
                </span>
              )}
            </div>
          </div>

          {/* Triggers List */}
          <div style={{
            display: 'grid',
            gap: 'var(--spacing-md)',
            marginTop: 'var(--spacing-lg)'
          }}>
            {triggers.map(trigger => {
              const Icon = trigger.icon;
              const progressValue = Math.min(Math.max(trigger.progress ?? 0, 0), 1);
              const showProgress = trigger.enabled && typeof trigger.progress === 'number';
              return (
                <div
                  key={trigger.id}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 'var(--spacing-md)',
                    padding: 'var(--spacing-md)',
                    background: trigger.enabled ? 'rgba(0, 255, 0, 0.05)' : 'rgba(0, 0, 0, 0.3)',
                    border: `1px solid ${trigger.enabled ? 'var(--color-accent)' : 'rgba(0, 255, 0, 0.2)'}`,
                    borderRadius: 'var(--border-radius-sm)',
                    transition: 'all 0.2s'
                  }}
                >
                  <Icon size={20} style={{ 
                    color: trigger.enabled ? 'var(--color-success)' : 'var(--color-text-dim)',
                    flexShrink: 0
                  }} />

                  <div style={{ flex: 1 }}>
                    <div style={{ 
                      display: 'flex', 
                      alignItems: 'center', 
                      gap: 'var(--spacing-sm)',
                      marginBottom: 'var(--spacing-xs)'
                    }}>
                      <span style={{
                        color: trigger.enabled ? 'var(--color-text-bright)' : 'var(--color-text)',
                        fontSize: 'var(--font-size-sm)',
                        fontWeight: 'bold'
                      }}>
                        {trigger.name}
                      </span>
                      {editingTrigger === trigger.id ? (
                        <div style={{
                          display: 'flex',
                          alignItems: 'center',
                          gap: 'var(--spacing-xs)'
                        }}>
                          <span style={{
                            color: 'var(--color-text)',
                            fontSize: 'var(--font-size-xs)'
                          }}>
                            {trigger.condition === 'above' ? '>' : '<'}
                          </span>
                          <input
                            type="number"
                            value={editValues[trigger.id] ?? trigger.threshold}
                            onChange={(e) => setEditValues({ ...editValues, [trigger.id]: parseFloat(e.target.value) })}
                            style={{
                              width: '60px',
                              padding: '2px 4px',
                              background: 'rgba(0, 0, 0, 0.5)',
                              border: '1px solid var(--color-accent)',
                              borderRadius: 'var(--border-radius-sm)',
                              color: 'var(--color-success)',
                              fontSize: 'var(--font-size-xs)',
                              fontFamily: 'var(--font-family-mono)'
                            }}
                            onKeyDown={(e) => {
                              if (e.key === 'Enter') {
                                handleUpdateTriggerThreshold(trigger.id, editValues[trigger.id] ?? trigger.threshold);
                              } else if (e.key === 'Escape') {
                                setEditingTrigger(null);
                                setEditValues({});
                              }
                            }}
                          />
                          <span style={{
                            color: 'var(--color-text)',
                            fontSize: 'var(--font-size-xs)'
                          }}>
                            {trigger.unit}
                          </span>
                          <button
                            onClick={() => handleUpdateTriggerThreshold(trigger.id, editValues[trigger.id] ?? trigger.threshold)}
                            style={{
                              background: 'transparent',
                              border: 'none',
                              color: 'var(--color-success)',
                              cursor: 'pointer',
                              padding: '2px'
                            }}
                            title="Save"
                          >
                            <Save size={14} />
                          </button>
                          <button
                            onClick={() => {
                              setEditingTrigger(null);
                              setEditValues({});
                            }}
                            style={{
                              background: 'transparent',
                              border: 'none',
                              color: 'var(--color-error)',
                              cursor: 'pointer',
                              padding: '2px'
                            }}
                            title="Cancel"
                          >
                            <X size={14} />
                          </button>
                        </div>
                      ) : (
                        <span style={{
                          color: 'var(--color-warning)',
                          fontSize: 'var(--font-size-xs)',
                          fontFamily: 'var(--font-family-mono)'
                        }}>
                          {trigger.condition === 'above' ? '>' : '<'} {trigger.threshold}{trigger.unit}
                        </span>
                      )}
                      <button
                        onClick={() => {
                          setEditingTrigger(trigger.id);
                          setEditValues({ [trigger.id]: trigger.threshold });
                        }}
                        style={{
                          background: 'transparent',
                          border: 'none',
                          color: 'var(--color-accent)',
                          cursor: 'pointer',
                          padding: '2px',
                          display: editingTrigger === trigger.id ? 'none' : 'block'
                        }}
                        title="Configure threshold"
                      >
                        <Settings size={14} />
                      </button>
                    </div>
                    <span style={{
                      color: 'var(--color-text-dim)',
                      fontSize: 'var(--font-size-xs)'
                    }}>
                      {trigger.description}
                    </span>
                    {showProgress && (
                      <div
                        style={{
                          marginTop: 'var(--spacing-sm)',
                          background: 'rgba(0, 255, 0, 0.08)',
                          borderRadius: '999px',
                          overflow: 'hidden',
                          height: '6px',
                          boxShadow: '0 0 8px rgba(0, 255, 0, 0.2)'
                        }}
                      >
                        <div
                          style={{
                            width: `${progressValue * 100}%`,
                            height: '100%',
                            background: 'linear-gradient(90deg, rgba(0, 255, 0, 0.3) 0%, rgba(0, 255, 0, 0.8) 100%)',
                            transition: 'width 0.4s ease',
                            boxShadow: progressValue > 0.95 ? '0 0 12px rgba(0, 255, 0, 0.6)' : 'none'
                          }}
                        />
                      </div>
                    )}
                  </div>

                  <div style={{ 
                    display: 'flex', 
                    alignItems: 'center', 
                    gap: 'var(--spacing-md)'
                  }}>
                    {/* Auto-fix Toggle */}
                    <label style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 'var(--spacing-xs)',
                      cursor: trigger.enabled ? 'pointer' : 'not-allowed',
                      opacity: trigger.enabled ? 1 : 0.5
                    }}>
                      <input
                        type="checkbox"
                        checked={trigger.autoFix}
                        onChange={() => trigger.enabled && handleToggleAutoFix(trigger.id)}
                        disabled={!trigger.enabled}
                        style={{
                          width: '16px',
                          height: '16px',
                          accentColor: 'var(--color-success)',
                          cursor: trigger.enabled ? 'pointer' : 'not-allowed'
                        }}
                      />
                      <Shield size={14} style={{ 
                        color: trigger.autoFix && trigger.enabled ? 'var(--color-success)' : 'var(--color-text-dim)' 
                      }} />
                      <span style={{
                        color: trigger.enabled ? 'var(--color-text)' : 'var(--color-text-dim)',
                        fontSize: 'var(--font-size-xs)',
                        userSelect: 'none'
                      }}>
                        Auto-fix
                      </span>
                    </label>

                    {/* Enable/Disable Toggle */}
                    <button
                      className={trigger.enabled ? 'btn btn-success' : 'btn btn-secondary'}
                      onClick={() => handleToggleTrigger(trigger.id)}
                      style={{
                        padding: 'var(--spacing-xs) var(--spacing-sm)',
                        fontSize: 'var(--font-size-xs)',
                        minWidth: '80px'
                      }}
                    >
                      {trigger.enabled ? 'ENABLED' : 'DISABLED'}
                    </button>
                  </div>
                </div>
              );
            })}
          </div>

          {/* Warning Note */}
          <div style={{
            display: 'flex',
            alignItems: 'flex-start',
            gap: 'var(--spacing-sm)',
            marginTop: 'var(--spacing-lg)',
            padding: 'var(--spacing-sm)',
            background: 'rgba(255, 166, 0, 0.1)',
            border: '1px solid var(--color-warning)',
            borderRadius: 'var(--border-radius-sm)'
          }}>
            <AlertCircle size={16} style={{ 
              color: 'var(--color-warning)',
              flexShrink: 0,
              marginTop: '2px'
            }} />
            <span style={{
              color: 'var(--color-text-dim)',
              fontSize: 'var(--font-size-xs)',
              lineHeight: '1.4'
            }}>
              <strong>Note:</strong> Triggers respect the cooldown period to prevent investigation spam. 
              Auto-fix mode only applies to safe operations like cache clearing and service restarts.
            </span>
          </div>
        </div>
      </div>
    </div>
  );
};
