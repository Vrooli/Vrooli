import { ScriptEditorModal } from './ScriptEditorModal';
import { ScriptResultsModal } from './ScriptResultsModal';
import type { ModalState, InvestigationScript } from '../../types';

interface ModalsContainerProps {
  modalState: ModalState;
  onCloseScriptEditor: () => void;
  onCloseScriptResults: () => void;
  onExecuteScript: (scriptId: string, content: string) => Promise<void>;
  onSaveScript?: (script: InvestigationScript, content: string) => Promise<void>;
}

export const ModalsContainer = ({ 
  modalState, 
  onCloseScriptEditor, 
  onCloseScriptResults, 
  onExecuteScript,
  onSaveScript 
}: ModalsContainerProps) => {
  return (
    <>
      <ScriptEditorModal
        isOpen={modalState.scriptEditor.isOpen}
        script={modalState.scriptEditor.script}
        scriptContent={modalState.scriptEditor.scriptContent}
        mode={modalState.scriptEditor.mode}
        onClose={onCloseScriptEditor}
        onExecute={onExecuteScript}
        onSave={onSaveScript}
      />
      
      <ScriptResultsModal
        isOpen={modalState.scriptResults.isOpen}
        execution={modalState.scriptResults.execution}
        onClose={onCloseScriptResults}
      />
    </>
  );
};
