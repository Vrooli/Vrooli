import { FC, memo } from 'react';
import type { NodeProps } from 'reactflow';
import { Trash2 } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import {
  useSyncedString,
  useSyncedNumber,
  useSyncedBoolean,
  useSyncedSelect,
} from '@hooks/useSyncedField';
import BaseNode from './BaseNode';
import {
  NodeTextField,
  NodeNumberField,
  NodeSelectField,
  NodeCheckbox,
  FieldRow,
} from './fields';

// ClearStorageParams interface for V2 native action params
interface ClearStorageParams {
  storageType?: string;
  clearAll?: boolean;
  key?: string;
  timeoutMs?: number;
  waitForMs?: number;
}

const STORAGE_OPTIONS = [
  { label: 'localStorage', value: 'localStorage' },
  { label: 'sessionStorage', value: 'sessionStorage' },
];

const MIN_TIMEOUT = 100;

const ClearStorageNode: FC<NodeProps> = ({ selected, id }) => {
  const { params, updateParams } = useActionParams<ClearStorageParams>(id);

  // Select field
  const storageType = useSyncedSelect(params?.storageType ?? 'localStorage', {
    onCommit: (v) => updateParams({ storageType: v }),
  });

  // Boolean field
  const clearAll = useSyncedBoolean(params?.clearAll ?? false, {
    onCommit: (v) => updateParams({ clearAll: v || undefined }),
  });

  // String field
  const keyValue = useSyncedString(params?.key ?? '', {
    onCommit: (v) => updateParams({ key: v || undefined }),
  });

  // Number fields
  const timeoutMs = useSyncedNumber(params?.timeoutMs ?? 15000, {
    min: MIN_TIMEOUT,
    fallback: 15000,
    onCommit: (v) => updateParams({ timeoutMs: v }),
  });
  const waitForMs = useSyncedNumber(params?.waitForMs ?? 0, {
    min: 0,
    onCommit: (v) => updateParams({ waitForMs: v || undefined }),
  });

  return (
    <BaseNode selected={selected} icon={Trash2} iconClassName="text-red-300" title="Clear Storage">
      <div className="space-y-3 text-xs">
        <NodeSelectField field={storageType} label="Storage" options={STORAGE_OPTIONS} />

        <NodeCheckbox field={clearAll} label="Remove every entry" />

        {!clearAll.value && (
          <NodeTextField field={keyValue} label="Key" placeholder="profile" />
        )}

        <FieldRow>
          <NodeNumberField field={timeoutMs} label="Timeout (ms)" min={MIN_TIMEOUT} />
          <NodeNumberField field={waitForMs} label="Post-clear wait (ms)" min={0} />
        </FieldRow>
      </div>
    </BaseNode>
  );
};

export default memo(ClearStorageNode);
