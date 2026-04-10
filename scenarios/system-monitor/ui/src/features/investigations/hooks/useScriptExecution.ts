import { useState, useCallback } from 'react';
import { protoFetch } from '../../../shared/api/apiFetch';
import { parseExecuteScriptResponse } from '../../../shared/api/proto-contracts';
import type { ModalState, InvestigationScript, ScriptExecution } from '../../../types';
import { ScriptExecutionStatus } from '../../../types';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';

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
      const pendingExecution = {
        scriptId,
        executionId: `exec-${Date.now()}`,
        status: ScriptExecutionStatus.RUNNING,
        startedAt: timestampFromDate(new Date()),
      } as ScriptExecution;

      setModalState(prev => ({
        ...prev,
        scriptResults: {
          isOpen: true,
          scriptId,
          executionId: pendingExecution.executionId,
          execution: pendingExecution
        },
        scriptEditor: {
          ...prev.scriptEditor,
          isOpen: false
        }
      }));

      const requestBody = scriptContent ? JSON.stringify({ content: scriptContent }) : '{}';

      try {
        const data = await protoFetch(
          `/investigations/scripts/${encodeURIComponent(scriptId)}/execute`,
          parseExecuteScriptResponse,
          {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: requestBody,
          },
        );

        const completedExecution = data.execution;
        if (completedExecution) {
          setModalState(prev => ({
            ...prev,
            scriptResults: {
              ...prev.scriptResults,
              execution: completedExecution as ScriptExecution
            }
          }));
        }
      } catch (fetchErr) {
        const now = timestampFromDate(new Date());
        const errMsg = fetchErr instanceof Error ? fetchErr.message : 'Script execution failed';
        setModalState(prev => ({
          ...prev,
          scriptResults: {
            ...prev.scriptResults,
            execution: {
              scriptId,
              executionId: `exec-${Date.now()}`,
              status: ScriptExecutionStatus.FAILED,
              startedAt: now,
              completedAt: now,
              exitCode: 1,
              error: errMsg,
            } as ScriptExecution,
          }
        }));
      }
    } catch (err) {
      console.error('Failed to execute script:', err);

      const now = timestampFromDate(new Date());
      setModalState(prev => ({
        ...prev,
        scriptResults: {
          ...prev.scriptResults,
          execution: {
            scriptId,
            executionId: `exec-${Date.now()}`,
            status: ScriptExecutionStatus.FAILED,
            startedAt: now,
            completedAt: now,
            exitCode: 1,
            error: err instanceof Error ? err.message : 'Unknown error occurred',
          } as ScriptExecution,
        }
      }));
    }
  }, []);

  const saveScript = useCallback(async (_script: InvestigationScript, _content: string) => {
    // TODO: Implement actual API call to save script
    setModalState(prev => ({
      ...prev,
      scriptEditor: {
        ...prev.scriptEditor,
        isOpen: false
      }
    }));
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
