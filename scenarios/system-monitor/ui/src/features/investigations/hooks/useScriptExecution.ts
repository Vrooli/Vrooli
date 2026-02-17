import { useState, useCallback } from 'react';
import { buildApiUrl } from '../../../shared/api/apiBase';
import { extractNumber, extractString } from '../../../shared/utils/typeGuards';
import type { ModalState, InvestigationScript, ScriptExecution } from '../../../types';

interface UseScriptExecutionReturn {
  modalState: ModalState;
  openScriptEditor: (script?: InvestigationScript, content?: string, mode?: 'create' | 'edit' | 'view') => void;
  closeScriptEditor: () => void;
  closeScriptResults: () => void;
  executeScript: (scriptId: string, scriptContent: string) => Promise<void>;
  saveScript: (script: InvestigationScript, content: string) => Promise<void>;
}

export const useScriptExecution = (): UseScriptExecutionReturn => {
  const [modalState, setModalState] = useState<ModalState>({
    reportModal: {
      isOpen: false,
      loading: false
    },
    scriptEditor: {
      isOpen: false,
      mode: 'view'
    },
    scriptResults: {
      isOpen: false
    }
  });

  const openScriptEditor = useCallback((script?: InvestigationScript, content?: string, mode: 'create' | 'edit' | 'view' = 'view') => {
    setModalState(prev => ({
      ...prev,
      scriptEditor: {
        isOpen: true,
        script,
        scriptContent: content,
        scriptId: script?.id,
        mode
      }
    }));
  }, []);

  const closeScriptEditor = useCallback(() => {
    setModalState(prev => ({
      ...prev,
      scriptEditor: {
        ...prev.scriptEditor,
        isOpen: false
      }
    }));
  }, []);

  const closeScriptResults = useCallback(() => {
    setModalState(prev => ({
      ...prev,
      scriptResults: {
        ...prev.scriptResults,
        isOpen: false
      }
    }));
  }, []);

  const executeScript = useCallback(async (scriptId: string, scriptContent: string) => {
    try {
      const execution: ScriptExecution = {
        script_id: scriptId,
        execution_id: `exec-${Date.now()}`,
        status: 'running',
        started_at: new Date().toISOString()
      };

      setModalState(prev => ({
        ...prev,
        scriptResults: {
          isOpen: true,
          scriptId,
          executionId: execution.execution_id,
          execution
        },
        scriptEditor: {
          ...prev.scriptEditor,
          isOpen: false
        }
      }));

      const requestBody = scriptContent ? { content: scriptContent } : {};

      const response = await fetch(buildApiUrl(`/investigations/scripts/${encodeURIComponent(scriptId)}/execute`), {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(requestBody)
      });

      let data: Record<string, unknown> | null = null;
      try {
        data = await response.json() as Record<string, unknown>;
      } catch {
        data = null;
      }

      const stdout = extractString(data?.['stdout']) ?? extractString(data?.['output']) ?? '';
      const stderr = extractString(data?.['stderr']) ?? '';
      const exitCode = extractNumber(data?.['exit_code']) ?? (response.ok ? 0 : 1);
      const timedOut = data?.['timed_out'] === true;
      const completedAt = extractString(data?.['completed_at']) ?? new Date().toISOString();
      const errorFromResponse = extractString(data?.['error']);
      const durationSeconds = extractNumber(data?.['duration_seconds']);

      const completedExecution: ScriptExecution = {
        ...execution,
        status: response.ok && exitCode === 0 && !timedOut ? 'completed' : 'failed',
        completed_at: completedAt,
        exit_code: exitCode,
        output: stdout,
        stdout,
        stderr,
        error: stderr || errorFromResponse || (!response.ok ? `Request failed with status ${response.status}` : undefined),
        timed_out: timedOut,
        duration_seconds: durationSeconds
      };

      setModalState(prev => ({
        ...prev,
        scriptResults: {
          ...prev.scriptResults,
          execution: completedExecution
        }
      }));
    } catch (err) {
      console.error('Failed to execute script:', err);

      setModalState(prev => ({
        ...prev,
        scriptResults: {
          ...prev.scriptResults,
          execution: {
            script_id: scriptId,
            execution_id: `exec-${Date.now()}`,
            status: 'failed',
            started_at: new Date().toISOString(),
            completed_at: new Date().toISOString(),
            exit_code: 1,
            error: err instanceof Error ? err.message : 'Unknown error occurred'
          }
        }
      }));
    }
  }, []);

  const saveScript = useCallback(async (script: InvestigationScript, content: string) => {
    try {
      // TODO: Implement actual API call to save script
      console.log('Saving script:', script, content);
      // For now, just close the modal
      setModalState(prev => ({
        ...prev,
        scriptEditor: {
          ...prev.scriptEditor,
          isOpen: false
        }
      }));
    } catch (err) {
      console.error('Failed to save script:', err);
    }
  }, []);

  return {
    modalState,
    openScriptEditor,
    closeScriptEditor,
    closeScriptResults,
    executeScript,
    saveScript
  };
};
