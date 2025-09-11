import { useState, useEffect } from 'react';
import { Settings, Shield, Clock, RefreshCw, AlertCircle, Cpu, HardDrive, Network, Database, Zap } from 'lucide-react';

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
}

interface AutomaticTriggersSectionProps {
  onUpdateTrigger: (triggerId: string, config: Partial<TriggerConfig>) => void;
}

export const AutomaticTriggersSection = ({ onUpdateTrigger }: AutomaticTriggersSectionProps) => {
  const [triggers, setTriggers] = useState<TriggerConfig[]>([
    {
      id: 'high-cpu',
      name: 'High CPU Usage',
      description: 'Triggers when CPU usage exceeds threshold',
      icon: Cpu,
      enabled: true,
      autoFix: false,
      threshold: 85,
      unit: '%',
      condition: 'above'
    },
    {
      id: 'high-memory',
      name: 'High Memory Usage',
      description: 'Triggers when memory usage exceeds threshold',
      icon: Database,
      enabled: true,
      autoFix: false,
      threshold: 90,
      unit: '%',
      condition: 'above'
    },
    {
      id: 'disk-space',
      name: 'Low Disk Space',
      description: 'Triggers when available disk space falls below threshold',
      icon: HardDrive,
      enabled: false,
      autoFix: true,
      threshold: 10,
      unit: 'GB',
      condition: 'below'
    },
    {
      id: 'network-connections',
      name: 'Excessive Network Connections',
      description: 'Triggers when TCP connections exceed threshold',
      icon: Network,
      enabled: true,
      autoFix: false,
      threshold: 1000,
      unit: 'connections',
      condition: 'above'
    },
    {
      id: 'process-count',
      name: 'Process Count Anomaly',
      description: 'Triggers when process count exceeds normal range',
      icon: Zap,
      enabled: false,
      autoFix: false,
      threshold: 500,
      unit: 'processes',
      condition: 'above'
    }
  ]);

  const [cooldownPeriod, setCooldownPeriod] = useState(300); // 5 minutes default
  const [cooldownRemaining, setCooldownRemaining] = useState(0);
  const [, setLastTriggerTime] = useState<Date | null>(null);

  // Update cooldown timer
  useEffect(() => {
    if (cooldownRemaining <= 0) return;
    
    const timer = setInterval(() => {
      setCooldownRemaining(prev => Math.max(0, prev - 1));
    }, 1000);

    return () => clearInterval(timer);
  }, [cooldownRemaining]);

  // Simulate checking for last trigger time on mount
  useEffect(() => {
    // TODO: Fetch actual last trigger time from API
    const mockLastTrigger = new Date(Date.now() - 120000); // 2 minutes ago
    setLastTriggerTime(mockLastTrigger);
    
    const elapsed = Math.floor((Date.now() - mockLastTrigger.getTime()) / 1000);
    const remaining = Math.max(0, cooldownPeriod - elapsed);
    setCooldownRemaining(remaining);
  }, [cooldownPeriod]);

  const handleToggleTrigger = (triggerId: string) => {
    setTriggers(prev => prev.map(trigger => {
      if (trigger.id === triggerId) {
        const updated = { ...trigger, enabled: !trigger.enabled };
        onUpdateTrigger(triggerId, { enabled: updated.enabled });
        return updated;
      }
      return trigger;
    }));
  };

  const handleToggleAutoFix = (triggerId: string) => {
    setTriggers(prev => prev.map(trigger => {
      if (trigger.id === triggerId) {
        const updated = { ...trigger, autoFix: !trigger.autoFix };
        onUpdateTrigger(triggerId, { autoFix: updated.autoFix });
        return updated;
      }
      return trigger;
    }));
  };

  const handleResetCooldown = () => {
    setCooldownRemaining(0);
    // TODO: Call API to reset cooldown
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
                value={cooldownPeriod}
                onChange={(e) => setCooldownPeriod(Number(e.target.value))}
                style={{
                  flex: 1,
                  height: '4px',
                  background: 'rgba(0, 255, 0, 0.2)',
                  borderRadius: '2px',
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
                {formatCooldownPeriod(cooldownPeriod)}
              </span>
            </div>

            <div style={{
              display: 'flex',
              alignItems: 'center',
              gap: 'var(--spacing-md)',
              paddingLeft: 'var(--spacing-md)',
              borderLeft: '1px solid rgba(0, 255, 0, 0.2)'
            }}>
              {cooldownRemaining > 0 ? (
                <>
                  <span style={{
                    color: 'var(--color-warning)',
                    fontSize: 'var(--font-size-sm)'
                  }}>
                    Cooldown: {formatTime(cooldownRemaining)}
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
                      <span style={{
                        color: 'var(--color-warning)',
                        fontSize: 'var(--font-size-xs)',
                        fontFamily: 'var(--font-family-mono)'
                      }}>
                        {trigger.condition === 'above' ? '>' : '<'} {trigger.threshold}{trigger.unit}
                      </span>
                    </div>
                    <span style={{
                      color: 'var(--color-text-dim)',
                      fontSize: 'var(--font-size-xs)'
                    }}>
                      {trigger.description}
                    </span>
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