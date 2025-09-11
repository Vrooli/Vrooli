import { useState, useEffect } from 'react';
import { FileText, RefreshCw, Plus } from 'lucide-react';
import type { InvestigationScript } from '../../types';
import { LoadingSkeleton } from '../common/LoadingSkeleton';

interface InvestigationScriptsPanelProps {
  onOpenScriptEditor: (script?: InvestigationScript, content?: string, mode?: 'create' | 'edit' | 'view') => void;
  embedded?: boolean;
  searchFilter?: string;
}

export const InvestigationScriptsPanel = ({ onOpenScriptEditor, embedded = false, searchFilter = '' }: InvestigationScriptsPanelProps) => {
  const [scripts, setScripts] = useState<InvestigationScript[]>([]);
  const [loading, setLoading] = useState(true);

  // Filter scripts based on search
  const filteredScripts = scripts.filter(script => {
    if (!searchFilter) return true;
    const searchLower = searchFilter.toLowerCase();
    return script.name.toLowerCase().includes(searchLower) ||
           script.description.toLowerCase().includes(searchLower) ||
           script.category.toLowerCase().includes(searchLower) ||
           script.id.toLowerCase().includes(searchLower);
  });

  const loadScripts = async () => {
    setLoading(true);
    try {
      // TODO: Implement API call to load scripts
      // For now, show placeholder data
      setTimeout(() => {
        setScripts([
          {
            id: 'high-cpu-analysis',
            name: 'High CPU Analysis',
            description: 'Analyzes processes consuming high CPU resources',
            category: 'performance',
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            author: 'system',
            enabled: true
          },
          {
            id: 'process-genealogy',
            name: 'Process Genealogy',
            description: 'Traces process parent-child relationships',
            category: 'process-analysis',
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            author: 'system',
            enabled: true
          }
        ]);
        setLoading(false);
      }, 1000);
    } catch (error) {
      console.error('Failed to load scripts:', error);
      setLoading(false);
    }
  };

  useEffect(() => {
    loadScripts();
  }, []);

  const showNewScriptDialog = () => {
    onOpenScriptEditor(undefined, '', 'create');
  };

  const getScriptContent = (scriptId: string): string => {
    // Mock script content - in real implementation, this would come from API
    const scriptContents: Record<string, string> = {
      'high-cpu-analysis': `#!/bin/bash
# High CPU Analysis Script
# Analyzes processes consuming high CPU resources

echo "=== High CPU Analysis Report ==="
echo "Generated at: $(date)"
echo ""

echo "Top 10 CPU-consuming processes:"
ps aux --sort=-%cpu | head -11

echo ""
echo "CPU usage by user:"
ps -eo user,%cpu --no-headers | awk '{cpu[$1]+=$2} END {for (u in cpu) printf("%-15s %.1f%%\\n", u, cpu[u])}' | sort -k2 -nr

echo ""
echo "Load average:"
uptime

echo ""
echo "CPU cores and current frequency:"
lscpu | grep -E "(CPU\\(s\\)|Core\\(s\\)|Thread\\(s\\)|CPU MHz)"

echo ""
echo "Context switches per second:"
grep ctxt /proc/stat

echo ""
echo "Interrupts per second:"
grep intr /proc/stat

echo ""
echo "Analysis complete."`,

      'process-genealogy': `#!/bin/bash
# Process Genealogy Script
# Traces process parent-child relationships

echo "=== Process Genealogy Report ==="
echo "Generated at: $(date)"
echo ""

echo "Process tree (showing all processes):"
pstree -p

echo ""
echo "Process hierarchy with CPU and memory usage:"
ps -eo pid,ppid,cmd,%cpu,%mem --sort=-%cpu | head -20

echo ""
echo "Processes by session ID:"
ps -eo pid,sid,cmd | sort -k2 -n

echo ""
echo "Process groups:"
ps -eo pid,pgid,cmd | sort -k2 -n

echo ""
echo "Orphaned processes (PPID = 1):"
ps -eo pid,ppid,cmd | awk '$2 == 1 && $1 != 1 {print}'

echo ""
echo "Zombie processes:"
ps aux | awk '$8 ~ /^Z/ {print}'

echo ""
echo "Analysis complete."`
    };

    return scriptContents[scriptId] || `#!/bin/bash
# Investigation Script: ${scriptId}
# Add your investigation logic here

echo "Starting investigation..."
echo "Script ID: ${scriptId}"
echo "Timestamp: $(date)"

# Add your commands here

echo "Investigation complete."`;
  };

  const openScript = async (script: InvestigationScript) => {
    const content = getScriptContent(script.id);
    onOpenScriptEditor(script, content, 'view');
  };

  // If embedded, render without header
  if (embedded) {
    return (
      <div className="investigation-scripts-list">
        {loading ? (
          <LoadingSkeleton variant="list" count={3} />
        ) : scripts.length === 0 ? (
          <div style={{
            textAlign: 'center',
            color: 'var(--color-text-dim)',
            padding: 'var(--spacing-lg)',
            fontSize: 'var(--font-size-lg)'
          }}>
            NO SCRIPTS AVAILABLE
          </div>
        ) : (
          <div className="scripts-list">
            {filteredScripts.map(script => (
              <div 
                key={script.id} 
                className="script-item" 
                onClick={() => openScript(script)}
                style={{
                  padding: 'var(--spacing-md)',
                  borderBottom: '1px solid var(--color-accent)',
                  background: 'rgba(0, 0, 0, 0.2)',
                  cursor: 'pointer',
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
                  <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-sm)' }}>
                    <FileText size={16} style={{ color: 'var(--color-accent)' }} />
                    <span style={{ 
                      margin: 0, 
                      color: 'var(--color-text-bright)',
                      fontSize: 'var(--font-size-md)',
                      fontWeight: 'bold'
                    }}>
                      {script.name}
                    </span>
                  </div>
                  
                  <span style={{
                    color: script.enabled ? 'var(--color-success)' : 'var(--color-text-dim)',
                    fontSize: 'var(--font-size-sm)',
                    textTransform: 'uppercase',
                    fontWeight: 'bold'
                  }}>
                    {script.enabled ? 'ENABLED' : 'DISABLED'}
                  </span>
                </div>
                
                <div style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  marginBottom: 'var(--spacing-sm)'
                }}>
                  <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
                    Category: {script.category}
                  </span>
                  <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
                    By: {script.author}
                  </span>
                </div>
                
                <div style={{
                  color: 'var(--color-text)',
                  fontSize: 'var(--font-size-sm)',
                  lineHeight: 1.4
                }}>
                  {script.description}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    );
  }

  return (
    <section className="investigation-scripts-panel card">
      <div className="panel-header" style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 'var(--spacing-md)'
      }}>
        <h2 style={{ margin: 0, color: 'var(--color-text-bright)' }}>
          INVESTIGATION SCRIPTS
        </h2>
        
        <div className="investigation-script-controls" style={{
          display: 'flex',
          gap: 'var(--spacing-sm)'
        }}>
          <button 
            className="btn btn-action"
            onClick={showNewScriptDialog}
          >
            <Plus size={16} />
            NEW SCRIPT
          </button>
          <button 
            className="btn btn-action"
            onClick={loadScripts}
          >
            <RefreshCw size={16} />
            REFRESH
          </button>
        </div>
      </div>
      
      <div className="investigation-scripts-list">
        {loading ? (
          <LoadingSkeleton variant="list" count={3} />
        ) : scripts.length === 0 ? (
          <div style={{
            textAlign: 'center',
            color: 'var(--color-text-dim)',
            padding: 'var(--spacing-lg)',
            fontSize: 'var(--font-size-lg)'
          }}>
            NO SCRIPTS AVAILABLE
          </div>
        ) : (
          <div className="scripts-list">
            {filteredScripts.map(script => (
              <div 
                key={script.id} 
                className="script-item" 
                onClick={() => openScript(script)}
                style={{
                  padding: 'var(--spacing-md)',
                  borderBottom: '1px solid var(--color-accent)',
                  background: 'rgba(0, 0, 0, 0.2)',
                  cursor: 'pointer',
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
                  <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-sm)' }}>
                    <FileText size={16} style={{ color: 'var(--color-accent)' }} />
                    <span style={{ 
                      margin: 0, 
                      color: 'var(--color-text-bright)',
                      fontSize: 'var(--font-size-md)',
                      fontWeight: 'bold'
                    }}>
                      {script.name}
                    </span>
                  </div>
                  
                  <span style={{
                    color: script.enabled ? 'var(--color-success)' : 'var(--color-text-dim)',
                    fontSize: 'var(--font-size-sm)',
                    textTransform: 'uppercase',
                    fontWeight: 'bold'
                  }}>
                    {script.enabled ? 'ENABLED' : 'DISABLED'}
                  </span>
                </div>
                
                <div style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  marginBottom: 'var(--spacing-sm)'
                }}>
                  <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
                    Category: {script.category}
                  </span>
                  <span style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-sm)' }}>
                    By: {script.author}
                  </span>
                </div>
                
                <div style={{
                  color: 'var(--color-text)',
                  fontSize: 'var(--font-size-sm)',
                  lineHeight: 1.4
                }}>
                  {script.description}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </section>
  );
};