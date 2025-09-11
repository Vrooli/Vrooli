import type { Investigation } from '../../types';

interface InvestigationsPanelProps {
  investigations: Investigation[];
  embedded?: boolean;
}

export const InvestigationsPanel = ({ investigations, embedded = false }: InvestigationsPanelProps) => {
  const triggerInvestigation = async () => {
    try {
      const response = await fetch('/api/investigations/trigger', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        }
      });
      
      if (response.ok) {
        // TODO: Show success message or refresh investigations
        console.log('Investigation triggered successfully');
      }
    } catch (error) {
      console.error('Failed to trigger investigation:', error);
    }
  };

  // If embedded, render without header
  if (embedded) {
    return (
      <div className="investigation-list">
        {investigations.length === 0 ? (
          <div className="loading" style={{
            textAlign: 'center',
            color: 'var(--color-text-dim)',
            padding: 'var(--spacing-lg)',
            fontSize: 'var(--font-size-lg)'
          }}>
            SCANNING SYSTEM...
          </div>
        ) : (
          investigations.map(investigation => (
            <div key={investigation.id} className="investigation-item" style={{
              padding: 'var(--spacing-md)',
              borderBottom: '1px solid var(--color-accent)',
              background: 'rgba(0, 0, 0, 0.2)',
              transition: 'background 0.2s'
            }}
            onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(0, 0, 0, 0.4)'}
            onMouseLeave={(e) => e.currentTarget.style.background = 'rgba(0, 0, 0, 0.2)'}
            >
              <div style={{ 
                display: 'flex', 
                justifyContent: 'space-between', 
                alignItems: 'center',
                marginBottom: 'var(--spacing-xs)'
              }}>
                <span style={{ color: 'var(--color-text-bright)', fontWeight: 'bold' }}>
                  Investigation {investigation.id}
                </span>
                <span style={{ 
                  color: investigation.status === 'completed' ? 'var(--color-success)' :
                         investigation.status === 'running' ? 'var(--color-warning)' :
                         'var(--color-text-dim)',
                  fontSize: 'var(--font-size-sm)',
                  textTransform: 'uppercase'
                }}>
                  {investigation.status}
                </span>
              </div>
              
              <div style={{ 
                display: 'flex', 
                justifyContent: 'space-between',
                marginBottom: 'var(--spacing-sm)'
              }}>
                <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
                  Trigger: {investigation.trigger_reason}
                </span>
                <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
                  {new Date(investigation.timestamp).toLocaleString()}
                </span>
              </div>
              
              {investigation.findings && (
                <div style={{ 
                  color: 'var(--color-text)',
                  fontSize: 'var(--font-size-sm)',
                  marginBottom: 'var(--spacing-sm)'
                }}>
                  {investigation.findings}
                </div>
              )}
              
              {investigation.confidence_score && (
                <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-sm)' }}>
                  <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
                    Confidence:
                  </span>
                  <div style={{
                    width: '100px',
                    height: '4px',
                    background: 'rgba(0, 255, 0, 0.2)',
                    borderRadius: '2px',
                    overflow: 'hidden'
                  }}>
                    <div style={{
                      width: `${investigation.confidence_score * 10}%`,
                      height: '100%',
                      background: investigation.confidence_score >= 8 ? 'var(--color-success)' :
                                 investigation.confidence_score >= 6 ? 'var(--color-warning)' :
                                 'var(--color-error)',
                      transition: 'width var(--transition-normal)'
                    }} />
                  </div>
                  <span style={{ color: 'var(--color-accent)', fontSize: 'var(--font-size-sm)' }}>
                    {investigation.confidence_score}/10
                  </span>
                </div>
              )}
            </div>
          ))
        )}
      </div>
    );
  }

  return (
    <section className="investigations-panel card">
      <div className="panel-header" style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 'var(--spacing-md)'
      }}>
        <h2 style={{ margin: 0, color: 'var(--color-text-bright)' }}>
          RECENT INVESTIGATIONS
        </h2>
        <button 
          className="btn btn-action"
          onClick={triggerInvestigation}
        >
          RUN ANOMALY CHECK
        </button>
      </div>
      
      <div className="investigation-list">
        {investigations.length === 0 ? (
          <div className="loading" style={{
            textAlign: 'center',
            color: 'var(--color-text-dim)',
            padding: 'var(--spacing-lg)',
            fontSize: 'var(--font-size-lg)'
          }}>
            SCANNING SYSTEM...
          </div>
        ) : (
          investigations.map(investigation => (
            <div key={investigation.id} className="investigation-item" style={{
              padding: 'var(--spacing-md)',
              borderBottom: '1px solid var(--color-accent)',
              background: 'rgba(0, 0, 0, 0.2)',
              transition: 'background 0.2s'
            }}
            onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(0, 0, 0, 0.4)'}
            onMouseLeave={(e) => e.currentTarget.style.background = 'rgba(0, 0, 0, 0.2)'}
            >
              <div style={{ 
                display: 'flex', 
                justifyContent: 'space-between', 
                alignItems: 'center',
                marginBottom: 'var(--spacing-xs)'
              }}>
                <span style={{ color: 'var(--color-text-bright)', fontWeight: 'bold' }}>
                  Investigation {investigation.id}
                </span>
                <span style={{ 
                  color: investigation.status === 'completed' ? 'var(--color-success)' :
                         investigation.status === 'running' ? 'var(--color-warning)' :
                         'var(--color-text-dim)',
                  fontSize: 'var(--font-size-sm)',
                  textTransform: 'uppercase'
                }}>
                  {investigation.status}
                </span>
              </div>
              
              <div style={{ 
                display: 'flex', 
                justifyContent: 'space-between',
                marginBottom: 'var(--spacing-sm)'
              }}>
                <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
                  Trigger: {investigation.trigger_reason}
                </span>
                <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
                  {new Date(investigation.timestamp).toLocaleString()}
                </span>
              </div>
              
              {investigation.findings && (
                <div style={{ 
                  color: 'var(--color-text)',
                  fontSize: 'var(--font-size-sm)',
                  marginBottom: 'var(--spacing-sm)'
                }}>
                  {investigation.findings}
                </div>
              )}
              
              {investigation.confidence_score && (
                <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-sm)' }}>
                  <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
                    Confidence:
                  </span>
                  <div style={{
                    width: '100px',
                    height: '4px',
                    background: 'rgba(0, 255, 0, 0.2)',
                    borderRadius: '2px',
                    overflow: 'hidden'
                  }}>
                    <div style={{
                      width: `${investigation.confidence_score * 10}%`,
                      height: '100%',
                      background: investigation.confidence_score >= 8 ? 'var(--color-success)' :
                                 investigation.confidence_score >= 6 ? 'var(--color-warning)' :
                                 'var(--color-error)',
                      transition: 'width var(--transition-normal)'
                    }} />
                  </div>
                  <span style={{ color: 'var(--color-accent)', fontSize: 'var(--font-size-sm)' }}>
                    {investigation.confidence_score}/10
                  </span>
                </div>
              )}
            </div>
          ))
        )}
      </div>
    </section>
  );
};