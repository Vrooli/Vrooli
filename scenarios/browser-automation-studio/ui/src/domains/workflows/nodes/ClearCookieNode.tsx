import { FC, memo } from 'react';
import type { NodeProps } from 'reactflow';
import { Cookie } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import {
  useSyncedString,
  useSyncedNumber,
  useSyncedBoolean,
} from '@hooks/useSyncedField';
import BaseNode from './BaseNode';
import {
  NodeTextField,
  NodeCheckbox,
  FieldRow,
  TimeoutFields,
} from './fields';

// ClearCookieParams interface for V2 native action params
interface ClearCookieParams {
  clearAll?: boolean;
  name?: string;
  url?: string;
  domain?: string;
  path?: string;
  timeoutMs?: number;
  waitForMs?: number;
}

const MIN_TIMEOUT = 100;

const ClearCookieNode: FC<NodeProps> = ({ selected, id }) => {
  const { params, updateParams } = useActionParams<ClearCookieParams>(id);

  // Boolean field
  const clearAll = useSyncedBoolean(params?.clearAll ?? false, {
    onCommit: (v) => updateParams({ clearAll: v || undefined }),
  });

  // String fields
  const name = useSyncedString(params?.name ?? '', {
    onCommit: (v) => updateParams({ name: v || undefined }),
  });
  const url = useSyncedString(params?.url ?? '', {
    onCommit: (v) => updateParams({ url: v || undefined }),
  });
  const domain = useSyncedString(params?.domain ?? '', {
    onCommit: (v) => updateParams({ domain: v || undefined }),
  });
  const path = useSyncedString(params?.path ?? '', {
    onCommit: (v) => updateParams({ path: v || undefined }),
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

  return (
    <BaseNode selected={selected} icon={Cookie} iconClassName="text-rose-300" title="Clear Cookie">
      <div className="space-y-3 text-xs">
        <NodeCheckbox field={clearAll} label="Remove every cookie in the current context" />

        {!clearAll.value && (
          <>
            <FieldRow>
              <NodeTextField field={name} label="Cookie name" placeholder="authToken" />
              <NodeTextField field={domain} label="Domain" placeholder=".example.com" />
            </FieldRow>

            <FieldRow>
              <NodeTextField field={url} label="Target URL" placeholder="https://example.com" />
              <NodeTextField field={path} label="Path" placeholder="/app" />
            </FieldRow>
          </>
        )}

        <TimeoutFields
          timeoutMs={timeoutMs}
          waitForMs={waitForMs}
          waitLabel="Post-clear wait (ms)"
          minTimeout={MIN_TIMEOUT}
        />
      </div>
    </BaseNode>
  );
};

export default memo(ClearCookieNode);
