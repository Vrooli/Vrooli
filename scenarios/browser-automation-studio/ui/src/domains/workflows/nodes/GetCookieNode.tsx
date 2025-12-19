import { FC, memo } from 'react';
import type { NodeProps } from 'reactflow';
import { Cookie } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import {
  useSyncedString,
  useSyncedNumber,
  useSyncedSelect,
} from '@hooks/useSyncedField';
import BaseNode from './BaseNode';
import {
  NodeTextField,
  NodeNumberField,
  NodeSelectField,
  FieldRow,
} from './fields';

// GetCookieParams interface for V2 native action params
interface GetCookieParams {
  name?: string;
  url?: string;
  domain?: string;
  path?: string;
  resultFormat?: string;
  storeResult?: string;
  timeoutMs?: number;
  waitForMs?: number;
}

const RESULT_OPTIONS = [
  { label: 'Store value only', value: 'value' },
  { label: 'Store full cookie object', value: 'object' },
];

const MIN_TIMEOUT = 100;

const GetCookieNode: FC<NodeProps> = ({ selected, id }) => {
  const { params, updateParams } = useActionParams<GetCookieParams>(id);

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
  const storeAs = useSyncedString(params?.storeResult ?? '', {
    onCommit: (v) => updateParams({ storeResult: v || undefined }),
  });

  // Select field
  const resultFormat = useSyncedSelect(params?.resultFormat ?? 'value', {
    onCommit: (v) => updateParams({ resultFormat: v }),
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
    <BaseNode selected={selected} icon={Cookie} iconClassName="text-lime-300" title="Get Cookie">
      <div className="space-y-3 text-xs">
        <FieldRow>
          <NodeTextField field={name} label="Cookie name" placeholder="authToken" />
          <NodeTextField field={storeAs} label="Store result as" placeholder="authToken" />
        </FieldRow>

        <FieldRow>
          <NodeTextField field={url} label="Target URL" placeholder="https://example.com" />
          <NodeTextField field={domain} label="Domain" placeholder=".example.com" />
        </FieldRow>

        <NodeTextField field={path} label="Path (optional)" placeholder="/dashboard" />

        <NodeSelectField field={resultFormat} label="Result format" options={RESULT_OPTIONS} />

        <FieldRow>
          <NodeNumberField field={timeoutMs} label="Timeout (ms)" min={MIN_TIMEOUT} />
          <NodeNumberField field={waitForMs} label="Post-fetch wait (ms)" min={0} />
        </FieldRow>
      </div>
    </BaseNode>
  );
};

export default memo(GetCookieNode);
