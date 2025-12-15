import { useEffect, useState, useMemo } from 'react';
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
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../../../shared/ui/select';
import {
  fetchFeedbackList,
  updateFeedbackStatus,
  deleteFeedback,
  deleteFeedbackBulk,
  type FeedbackRequest,
} from '../../../shared/api';

type FeedbackType = 'refund' | 'bug' | 'feature' | 'general';
type FeedbackStatus = 'pending' | 'in_progress' | 'resolved' | 'rejected';

const typeConfig: Record<FeedbackType, { icon: React.ReactNode; label: string; color: string }> = {
  refund: {
    icon: <RefreshCcw className="h-4 w-4" />,
    label: 'Refund Request',
    color: 'text-amber-400 bg-amber-500/10 border-amber-500/20',
  },
  bug: {
    icon: <Bug className="h-4 w-4" />,
    label: 'Bug Report',
    color: 'text-red-400 bg-red-500/10 border-red-500/20',
  },
  feature: {
    icon: <Lightbulb className="h-4 w-4" />,
    label: 'Feature Request',
    color: 'text-blue-400 bg-blue-500/10 border-blue-500/20',
  },
  general: {
    icon: <Heart className="h-4 w-4" />,
    label: 'General Feedback',
    color: 'text-purple-400 bg-purple-500/10 border-purple-500/20',
  },
};

const statusConfig: Record<FeedbackStatus, { icon: React.ReactNode; label: string; color: string }> = {
  pending: {
    icon: <Clock className="h-4 w-4" />,
    label: 'Pending',
    color: 'text-slate-400 bg-slate-500/10 border-slate-500/20',
  },
  in_progress: {
    icon: <AlertCircle className="h-4 w-4" />,
    label: 'In Progress',
    color: 'text-blue-400 bg-blue-500/10 border-blue-500/20',
  },
  resolved: {
    icon: <CheckCircle className="h-4 w-4" />,
    label: 'Resolved',
    color: 'text-green-400 bg-green-500/10 border-green-500/20',
  },
  rejected: {
    icon: <XCircle className="h-4 w-4" />,
    label: 'Rejected',
    color: 'text-red-400 bg-red-500/10 border-red-500/20',
  },
};

export function FeedbackManagement() {
  const [feedbackList, setFeedbackList] = useState<FeedbackRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [typeFilter, setTypeFilter] = useState<string>('all');
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [expandedId, setExpandedId] = useState<number | null>(null);
  const [actionLoading, setActionLoading] = useState<number | null>(null);
  const [bulkActionLoading, setBulkActionLoading] = useState(false);

  useEffect(() => {
    loadFeedback();
  }, []);

  const loadFeedback = async () => {
    try {
      setLoading(true);
      const data = await fetchFeedbackList();
      setFeedbackList(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load feedback');
    } finally {
      setLoading(false);
    }
  };

  const filteredFeedback = useMemo(() => {
    return feedbackList.filter((f) => {
      if (statusFilter !== 'all' && f.status !== statusFilter) return false;
      if (typeFilter !== 'all' && f.type !== typeFilter) return false;
      return true;
    });
  }, [feedbackList, statusFilter, typeFilter]);

  const handleStatusChange = async (id: number, newStatus: FeedbackStatus) => {
    try {
      setActionLoading(id);
      const updated = await updateFeedbackStatus(id, newStatus);
      setFeedbackList((prev) => prev.map((f) => (f.id === id ? updated : f)));
    } catch (err) {
      console.error('Failed to update status:', err);
    } finally {
      setActionLoading(null);
    }
  };

  const handleDelete = async (id: number) => {
    if (!confirm('Are you sure you want to delete this feedback?')) return;

    try {
      setActionLoading(id);
      await deleteFeedback(id);
      setFeedbackList((prev) => prev.filter((f) => f.id !== id));
      setSelectedIds((prev) => {
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
    } catch (err) {
      console.error('Failed to delete feedback:', err);
    } finally {
      setActionLoading(null);
    }
  };

  const handleBulkDelete = async () => {
    if (selectedIds.size === 0) return;
    if (!confirm(`Are you sure you want to delete ${selectedIds.size} feedback item(s)?`)) return;

    try {
      setBulkActionLoading(true);
      await deleteFeedbackBulk(Array.from(selectedIds));
      setFeedbackList((prev) => prev.filter((f) => !selectedIds.has(f.id)));
      setSelectedIds(new Set());
    } catch (err) {
      console.error('Failed to bulk delete:', err);
    } finally {
      setBulkActionLoading(false);
    }
  };

  const toggleSelect = (id: number) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const toggleSelectAll = () => {
    if (selectedIds.size === filteredFeedback.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(filteredFeedback.map((f) => f.id)));
    }
  };

  const handleReply = (email: string, subject: string) => {
    const mailtoUrl = `mailto:${email}?subject=Re: ${encodeURIComponent(subject)}`;
    window.open(mailtoUrl, '_blank');
  };

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: 'numeric',
      minute: '2-digit',
    });
  };

  const pendingCount = feedbackList.filter((f) => f.status === 'pending').length;
  const inProgressCount = feedbackList.filter((f) => f.status === 'in_progress').length;

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
    <AdminLayout>
      <div className="max-w-6xl mx-auto">
        {/* Header */}
        <div className="flex flex-col md:flex-row md:items-center md:justify-between mb-8 gap-4">
          <div>
            <h1 className="text-3xl font-semibold flex items-center gap-3">
              <MessageSquare className="h-8 w-8" />
              Feedback Management
            </h1>
            <p className="text-slate-400 mt-1">
              {feedbackList.length} total items
              {pendingCount > 0 && (
                <span className="text-amber-400 ml-2">({pendingCount} pending)</span>
              )}
              {inProgressCount > 0 && (
                <span className="text-blue-400 ml-2">({inProgressCount} in progress)</span>
              )}
            </p>
          </div>
          <Button onClick={loadFeedback} variant="outline" className="gap-2">
            <RefreshCcw className="h-4 w-4" />
            Refresh
          </Button>
        </div>

        {/* Filters and Bulk Actions */}
        <Card className="bg-white/5 border-white/10 mb-6">
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
                    onClick={handleBulkDelete}
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
          <Card className="bg-white/5 border-white/10">
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
                onClick={toggleSelectAll}
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
              const type = typeConfig[feedback.type as FeedbackType] || typeConfig.general;
              const status = statusConfig[feedback.status as FeedbackStatus] || statusConfig.pending;
              const isExpanded = expandedId === feedback.id;
              const isSelected = selectedIds.has(feedback.id);
              const isLoading = actionLoading === feedback.id;

              return (
                <Card
                  key={feedback.id}
                  className={`bg-white/5 border-white/10 transition-all ${isSelected ? 'ring-1 ring-blue-500/50' : ''}`}
                >
                  <CardContent className="p-0">
                    {/* Main Row */}
                    <div className="flex items-center gap-3 p-4">
                      <button
                        onClick={() => toggleSelect(feedback.id)}
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
                        <div className={`p-2 rounded-lg border ${type.color}`}>
                          {type.icon}
                        </div>

                        {/* Content */}
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 mb-1">
                            <span className="font-medium truncate">{feedback.subject}</span>
                            <span className={`text-xs px-2 py-0.5 rounded-full border ${status.color} flex items-center gap-1`}>
                              {status.icon}
                              {status.label}
                            </span>
                          </div>
                          <div className="flex items-center gap-3 text-sm text-slate-400">
                            <span>{feedback.email}</span>
                            <span className="text-slate-600">•</span>
                            <span>{formatDate(feedback.created_at)}</span>
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
                            onClick={() => handleDelete(feedback.id)}
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
