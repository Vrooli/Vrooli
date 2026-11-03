import { useState, useCallback } from 'react';
import { protoFetch } from '../../../shared/api/apiFetch';
import type { ModalState, InvestigationScript } from '../../../types';
import { ScriptExecutionStatus } from '../../../types';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { create } from '@bufbuild/protobuf';
import { ScriptExecutionSchema } from '@vrooli/proto-types/system-monitor/v1/scripts/scripts_pb';

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
    const startedAt = timestampFromDate(new Date());
    const executionId = `exec-${Date.now()}`;
    const pendingExecution = create(ScriptExecutionSchema, {
      scriptId,
      executionId,
      status: ScriptExecutionStatus.RUNNING,
      startedAt,
    });

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
      const { parseExecuteScriptResponse } = await import('../../../shared/api/proto-contracts');
      const data = await protoFetch(
        `/investigations/scripts/${encodeURIComponent(scriptId)}/execute`,
        parseExecuteScriptResponse,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: requestBody,
        },
      );

      if (data.execution) {
        setModalState(prev => ({
          ...prev,
          scriptResults: {
            ...prev.scriptResults,
            execution: data.execution,
          }
        }));
      }
    } catch (error) {
      console.error('Failed to execute script:', error);
      const now = timestampFromDate(new Date());
      setModalState(prev => ({
        ...prev,
        scriptResults: {
          ...prev.scriptResults,
          execution: create(ScriptExecutionSchema, {
            scriptId,
            executionId,
            status: ScriptExecutionStatus.FAILED,
            startedAt,
            completedAt: now,
            exitCode: 1,
            error: error instanceof Error ? error.message : 'Script execution failed',
          }),
        }
      }));
    }
  }, []);

  const saveScript = useCallback(async (script: InvestigationScript, content: string): Promise<void> => {
    if (!script.id || !content.trim()) throw new Error('Script content is required');
    const { parseGetScriptResponse } = await import('../../../shared/api/proto-contracts');
    await protoFetch(`/investigations/scripts/${encodeURIComponent(script.id)}`, parseGetScriptResponse, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id: script.id, content }),
    });
    setModalState(prev => ({
      ...prev,
      scriptEditor: { ...prev.scriptEditor, isOpen: false }
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
