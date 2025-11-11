import { memo } from 'react';
import type { NodeProps } from 'reactflow';
import { BlurNode as BlurNodeComponent } from './FocusNode';

const WrappedBlurNode = (props: NodeProps) => <BlurNodeComponent {...props} />;

export default memo(WrappedBlurNode);
