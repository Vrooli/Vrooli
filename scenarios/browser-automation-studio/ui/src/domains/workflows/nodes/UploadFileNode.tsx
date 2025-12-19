import { FC, memo, useMemo } from 'react';
import type { NodeProps } from 'reactflow';
import { UploadCloud, FileWarning } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import {
  useSyncedString,
  useSyncedNumber,
  textInputHandler,
  numberInputHandler,
} from '@hooks/useSyncedField';
import { selectors } from '@constants/selectors';
import BaseNode from './BaseNode';

// UploadFileParams interface for V2 native action params
interface UploadFileParams {
  selector?: string;
  filePaths?: string[];
  filePath?: string;
  timeoutMs?: number;
  waitForMs?: number;
}

const deriveFileText = (filePaths?: string[], filePath?: string): string => {
  if (filePaths && filePaths.length > 0) {
    return filePaths.join('\n');
  }
  if (filePath) {
    return filePath;
  }
  return '';
};

const parseFilePaths = (value: string): string[] => {
  return value
    .split(/\r?\n|,/)
    .map((entry) => entry.trim())
    .filter(Boolean);
};

const MIN_TIMEOUT = 500;

const UploadFileNode: FC<NodeProps> = ({ id, selected }) => {
  const { params, updateParams } = useActionParams<UploadFileParams>(id);

  // String field
  const selector = useSyncedString(params?.selector ?? '', {
    onCommit: (v) => updateParams({ selector: v || undefined }),
  });

  // File paths as text (special handling for array)
  const fileText = useSyncedString(deriveFileText(params?.filePaths, params?.filePath), {
    trim: false,
    onCommit: (v) => {
      const parsed = parseFilePaths(v);
      if (parsed.length === 0) {
        updateParams({ filePaths: undefined, filePath: undefined });
      } else if (parsed.length === 1) {
        updateParams({ filePaths: parsed, filePath: parsed[0] });
      } else {
        updateParams({ filePaths: parsed, filePath: undefined });
      }
    },
  });

  // Number fields
  const timeoutMs = useSyncedNumber(params?.timeoutMs ?? 30000, {
    min: MIN_TIMEOUT,
    fallback: 30000,
    onCommit: (v) => updateParams({ timeoutMs: v }),
  });
  const waitForMs = useSyncedNumber(params?.waitForMs ?? 0, {
    min: 0,
    onCommit: (v) => updateParams({ waitForMs: v || undefined }),
  });

  const activePaths = useMemo(() => parseFilePaths(fileText.value), [fileText.value]);

  return (
    <BaseNode selected={selected} icon={UploadCloud} iconClassName="text-pink-300" title="Upload File">
      <p className="text-[11px] text-gray-500 mb-3">Attach local files to file inputs</p>

      <div className="space-y-3 text-xs">
        <div>
          <label className="text-[11px] font-semibold text-gray-400">Target selector</label>
          <input
            type="text"
            value={selector.value}
            onChange={textInputHandler(selector.setValue)}
            onBlur={selector.commit}
            placeholder="#upload"
            className="w-full px-2 py-1 mt-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
          <p className="text-[10px] text-gray-500 mt-1">Must point to an &lt;input type="file"&gt; element.</p>
        </div>

        <div>
          <label className="text-[11px] font-semibold text-gray-400">File paths</label>
          <textarea
            value={fileText.value}
            onChange={textInputHandler(fileText.setValue)}
            onBlur={fileText.commit}
            placeholder="/home/me/Documents/example.pdf"
            rows={3}
            className="w-full px-2 py-1 mt-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
          <div className="mt-1 text-[10px] text-gray-500 flex items-center gap-1">
            <FileWarning size={12} className="text-amber-300" />
            Paths must be absolute on the execution machine. One per line for multiple files.
          </div>
          {activePaths.length > 0 && (
            <p
              className="text-[10px] text-emerald-300 mt-1"
              data-testid={selectors.workflowBuilder.nodes.upload.pathCount({ id })}
            >
              {activePaths.length === 1 ? '1 file selected' : `${activePaths.length} files selected`}
            </p>
          )}
        </div>

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-[11px] font-semibold text-gray-400">Timeout (ms)</label>
            <input
              type="number"
              min={MIN_TIMEOUT}
              value={timeoutMs.value}
              onChange={numberInputHandler(timeoutMs.setValue)}
              onBlur={timeoutMs.commit}
              className="w-full px-2 py-1 mt-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
          <div>
            <label className="text-[11px] font-semibold text-gray-400">Wait after (ms)</label>
            <input
              type="number"
              min={0}
              value={waitForMs.value}
              onChange={numberInputHandler(waitForMs.setValue)}
              onBlur={waitForMs.commit}
              className="w-full px-2 py-1 mt-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        </div>
      </div>
    </BaseNode>
  );
};

export default memo(UploadFileNode);
