import {
  MessageSquare,
  RefreshCcw,
  Bug,
  Lightbulb,
  Heart,
  Trash2,
  Mail,
  CheckCircle,
  Clock,
  XCircle,
  AlertCircle,
  Filter,
  ChevronDown,
  ChevronUp,
  Square,
  CheckSquare,
  ExternalLink,
} from 'lucide-react';
import { AdminLayout } from '../components/AdminLayout';
import { PageHeader } from '../components/PageHeader';
import { LAYOUT } from '../config/layout.constants';
import { Card, CardContent } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../../../shared/ui/select';
import { useFeedbackManagement } from '../hooks/useFeedbackManagement';
import {
  type FeedbackType,
  type FeedbackStatus,
  TYPE_CONFIG,
  STATUS_CONFIG,
  getTypeColor,
  getStatusColor,
} from '../services/feedback.service';
import { formatFeedbackTimestamp } from '../../../shared/lib/dateFormatters';

// Icons are component-specific (JSX), so they live in the route
const typeIcons: Record<FeedbackType, React.ReactNode> = {
  refund: <RefreshCcw className="h-4 w-4" />,
  bug: <Bug className="h-4 w-4" />,
  feature: <Lightbulb className="h-4 w-4" />,
  general: <Heart className="h-4 w-4" />,
};

const statusIcons: Record<FeedbackStatus, React.ReactNode> = {
  pending: <Clock className="h-4 w-4" />,
  in_progress: <AlertCircle className="h-4 w-4" />,
  resolved: <CheckCircle className="h-4 w-4" />,
  rejected: <XCircle className="h-4 w-4" />,
};

export function FeedbackManagement() {
  const {
    feedbackList,
    filteredFeedback,
    pendingCount,
    inProgressCount,
    statusFilter,
    typeFilter,
    selectedIds,
    expandedId,
    loading,
    error,
    actionLoading,
    bulkActionLoading,
    setStatusFilter,
    setTypeFilter,
    handleToggleSelect,
    handleToggleSelectAll,
    setExpandedId,
    loadFeedback,
    handleStatusChange,
    handleDelete,
    handleBulkDelete,
    handleReply,
  } = useFeedbackManagement();

  const onDelete = async (id: number) => {
    if (!confirm('Are you sure you want to delete this feedback?')) return;
    await handleDelete(id);
  };

  const onBulkDelete = async () => {
    if (!confirm(`Are you sure you want to delete ${selectedIds.size} feedback item(s)?`)) return;
    await handleBulkDelete();
  };

  if (loading) {
    return (
      <AdminLayout>
        <div className="flex items-center justify-center min-h-[400px]">
          <p className="text-slate-400">Loading feedback...</p>
        </div>
      </AdminLayout>
    );
  }

  if (error) {
    return (
      <AdminLayout>
        <div className="rounded-xl border border-red-500/20 bg-red-500/10 p-4">
          <p className="text-red-400">Error: {error}</p>
          <Button onClick={loadFeedback} variant="outline" className="mt-4">
            Retry
          </Button>
        </div>
      </AdminLayout>
    );
  }

  return (
    <AdminLayout maxWidth="wide">
      <div className={LAYOUT.pageSpacing}>
        <PageHeader
          variant="icon-title"
          title="Feedback Management"
          description={`${feedbackList.length} total items${pendingCount > 0 ? ` (${pendingCount} pending)` : ''}${inProgressCount > 0 ? ` (${inProgressCount} in progress)` : ''}`}
          icon={MessageSquare}
          iconBgClass="bg-blue-500/10"
          iconColorClass="text-blue-400"
          actions={
            <Button onClick={loadFeedback} variant="outline" className="gap-2">
              <RefreshCcw className="h-4 w-4" />
              Refresh
            </Button>
          }
          testId="feedback-management-header"
        />

        {/* Filters and Bulk Actions */}
        <Card className={`${LAYOUT.card.base} mb-6`}>
          <CardContent className="py-4">
            <div className="flex flex-col md:flex-row md:items-center gap-4 justify-between">
              <div className="flex items-center gap-3">
                <Filter className="h-4 w-4 text-slate-400" />
                <Select value={statusFilter} onValueChange={setStatusFilter}>
                  <SelectTrigger className="w-[150px] bg-white/5 border-white/10">
                    <SelectValue placeholder="Status" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Status</SelectItem>
                    <SelectItem value="pending">Pending</SelectItem>
                    <SelectItem value="in_progress">In Progress</SelectItem>
                    <SelectItem value="resolved">Resolved</SelectItem>
                    <SelectItem value="rejected">Rejected</SelectItem>
                  </SelectContent>
                </Select>

                <Select value={typeFilter} onValueChange={setTypeFilter}>
                  <SelectTrigger className="w-[160px] bg-white/5 border-white/10">
                    <SelectValue placeholder="Type" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Types</SelectItem>
                    <SelectItem value="refund">Refund Request</SelectItem>
                    <SelectItem value="bug">Bug Report</SelectItem>
                    <SelectItem value="feature">Feature Request</SelectItem>
                    <SelectItem value="general">General</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              {selectedIds.size > 0 && (
                <div className="flex items-center gap-3">
                  <span className="text-sm text-slate-400">{selectedIds.size} selected</span>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={onBulkDelete}
                    disabled={bulkActionLoading}
                    className="gap-2 text-red-400 border-red-500/20 hover:bg-red-500/10"
                  >
                    <Trash2 className="h-4 w-4" />
                    Delete Selected
                  </Button>
                </div>
              )}
            </div>
          </CardContent>
        </Card>

        {/* Feedback List */}
        {filteredFeedback.length === 0 ? (
          <Card className={LAYOUT.card.base}>
            <CardContent className="py-12 text-center">
              <MessageSquare className="h-12 w-12 mx-auto text-slate-500 mb-4" />
              <p className="text-slate-400">No feedback found</p>
              <p className="text-sm text-slate-500 mt-1">
                {statusFilter !== 'all' || typeFilter !== 'all'
                  ? 'Try adjusting your filters'
                  : 'Feedback submissions will appear here'}
              </p>
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-3">
            {/* Select All Header */}
            <div className="flex items-center gap-3 px-4 py-2 text-sm text-slate-400">
              <button
                onClick={handleToggleSelectAll}
                className="hover:text-white transition-colors"
                aria-label={selectedIds.size === filteredFeedback.length ? 'Deselect all' : 'Select all'}
              >
                {selectedIds.size === filteredFeedback.length ? (
                  <CheckSquare className="h-5 w-5" />
                ) : (
                  <Square className="h-5 w-5" />
                )}
              </button>
              <span>
                {selectedIds.size === filteredFeedback.length
                  ? 'Deselect all'
                  : `Select all (${filteredFeedback.length})`}
              </span>
            </div>

            {filteredFeedback.map((feedback) => {
              const typeConfig = TYPE_CONFIG[feedback.type as FeedbackType] || TYPE_CONFIG.general;
              const statusConfig = STATUS_CONFIG[feedback.status as FeedbackStatus] || STATUS_CONFIG.pending;
              const typeIcon = typeIcons[feedback.type as FeedbackType] || typeIcons.general;
              const statusIcon = statusIcons[feedback.status as FeedbackStatus] || statusIcons.pending;
              const isExpanded = expandedId === feedback.id;
              const isSelected = selectedIds.has(feedback.id);
              const isLoading = actionLoading === feedback.id;

              return (
                <Card
                  key={feedback.id}
                  className={`${LAYOUT.card.base} transition-all ${isSelected ? 'ring-1 ring-blue-500/50' : ''}`}
                >
                  <CardContent className="p-0">
                    {/* Main Row */}
                    <div className="flex items-center gap-3 p-4">
                      <button
                        onClick={() => handleToggleSelect(feedback.id)}
                        className="hover:text-white transition-colors text-slate-400"
                        aria-label={isSelected ? 'Deselect' : 'Select'}
                      >
                        {isSelected ? (
                          <CheckSquare className="h-5 w-5 text-blue-400" />
                        ) : (
                          <Square className="h-5 w-5" />
                        )}
                      </button>

                      <button
                        onClick={() => setExpandedId(isExpanded ? null : feedback.id)}
                        className="flex-1 flex items-start gap-4 text-left hover:bg-white/5 -m-2 p-2 rounded-lg transition-colors"
                      >
                        {/* Type Badge */}
                        <div className={`p-2 rounded-lg border ${getTypeColor(feedback.type)}`}>
                          {typeIcon}
                        </div>

                        {/* Content */}
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 mb-1">
                            <span className="font-medium truncate">{feedback.subject}</span>
                            <span className={`text-xs px-2 py-0.5 rounded-full border ${getStatusColor(feedback.status)} flex items-center gap-1`}>
                              {statusIcon}
                              {statusConfig.label}
                            </span>
                          </div>
                          <div className="flex items-center gap-3 text-sm text-slate-400">
                            <span>{feedback.email}</span>
                            <span className="text-slate-600">•</span>
                            <span>{formatFeedbackTimestamp(feedback.created_at)}</span>
                            {feedback.order_id && (
                              <>
                                <span className="text-slate-600">•</span>
                                <span className="text-amber-400">Order: {feedback.order_id}</span>
                              </>
                            )}
                          </div>
                        </div>

                        {/* Expand Icon */}
                        {isExpanded ? (
                          <ChevronUp className="h-5 w-5 text-slate-400" />
                        ) : (
                          <ChevronDown className="h-5 w-5 text-slate-400" />
                        )}
                      </button>
                    </div>

                    {/* Expanded Content */}
                    {isExpanded && (
                      <div className="border-t border-white/10 p-4 space-y-4">
                        {/* Message */}
                        <div>
                          <p className="text-xs text-slate-500 uppercase tracking-wider mb-2">Message</p>
                          <p className="text-slate-200 whitespace-pre-wrap bg-slate-800/50 rounded-lg p-4">
                            {feedback.message}
                          </p>
                        </div>

                        {/* Actions */}
                        <div className="flex flex-wrap items-center gap-3 pt-2">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleReply(feedback.email, feedback.subject)}
                            className="gap-2"
                          >
                            <Mail className="h-4 w-4" />
                            Reply via Email
                            <ExternalLink className="h-3 w-3" />
                          </Button>

                          <div className="flex items-center gap-2">
                            <span className="text-xs text-slate-500">Status:</span>
                            <Select
                              value={feedback.status}
                              onValueChange={(value) => handleStatusChange(feedback.id, value as FeedbackStatus)}
                              disabled={isLoading}
                            >
                              <SelectTrigger className="w-[140px] h-8 bg-white/5 border-white/10">
                                <SelectValue />
                              </SelectTrigger>
                              <SelectContent>
                                <SelectItem value="pending">Pending</SelectItem>
                                <SelectItem value="in_progress">In Progress</SelectItem>
                                <SelectItem value="resolved">Resolved</SelectItem>
                                <SelectItem value="rejected">Rejected</SelectItem>
                              </SelectContent>
                            </Select>
                          </div>

                          <div className="flex-1" />

                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => onDelete(feedback.id)}
                            disabled={isLoading}
                            className="gap-2 text-red-400 border-red-500/20 hover:bg-red-500/10"
                          >
                            <Trash2 className="h-4 w-4" />
                            Delete
                          </Button>
                        </div>
                      </div>
                    )}
                  </CardContent>
                </Card>
              );
            })}
          </div>
        )}
      </div>
    </AdminLayout>
  );
}
