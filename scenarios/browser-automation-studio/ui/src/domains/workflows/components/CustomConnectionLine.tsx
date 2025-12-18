import React from 'react';
import { ConnectionLineComponentProps } from 'reactflow';

const CustomConnectionLine: React.FC<ConnectionLineComponentProps> = ({
  fromX,
  fromY,
  toX,
  toY,
}) => {
  return (
    <g>
      <path
        fill="none"
        stroke="#4a5568"
        strokeWidth={2}
        className="animated"
        d={`M${fromX},${fromY} C ${fromX} ${fromY + 50} ${toX} ${toY - 50} ${toX},${toY}`}
      />
      <circle
        cx={toX}
        cy={toY}
        fill="#4a5568"
        r={5}
        stroke="#4a5568"
        strokeWidth={1.5}
      />
    </g>
  );
};

export default CustomConnectionLine;