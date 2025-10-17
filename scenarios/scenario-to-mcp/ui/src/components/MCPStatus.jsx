import React from 'react';
import { Circle, CheckCircle, XCircle, Clock } from 'lucide-react';
import './MCPStatus.css';

function MCPStatus({ status }) {
  const getStatusIcon = () => {
    switch (status) {
      case 'active':
        return <CheckCircle className="status-icon active" />;
      case 'inactive':
        return <XCircle className="status-icon inactive" />;
      case 'pending':
        return <Clock className="status-icon pending" />;
      case 'error':
        return <XCircle className="status-icon error" />;
      default:
        return <Circle className="status-icon unknown" />;
    }
  };

  const getStatusLabel = () => {
    switch (status) {
      case 'active':
        return 'Active';
      case 'inactive':
        return 'Inactive';
      case 'pending':
        return 'Pending';
      case 'error':
        return 'Error';
      default:
        return 'No MCP';
    }
  };

  return (
    <div className={`mcp-status status-${status || 'none'}`}>
      {getStatusIcon()}
      <span className="status-label">{getStatusLabel()}</span>
    </div>
  );
}

export default MCPStatus;