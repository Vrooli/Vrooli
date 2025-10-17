import React, { useState } from 'react';
import { FiX, FiEdit2, FiTrash2, FiSave, FiMessageCircle } from 'react-icons/fi';
import { BiUpvote, BiDownvote } from 'react-icons/bi';
import { format } from 'date-fns';
import clsx from 'clsx';

const FeatureRequestModal = ({ request, isOpen, onClose, onVote, onUpdate, user }) => {
  const [isEditing, setIsEditing] = useState(false);
  const [editData, setEditData] = useState({
    title: request?.title || '',
    description: request?.description || '',
    status: request?.status || 'proposed',
    priority: request?.priority || 'medium',
  });
  const [comment, setComment] = useState('');

  const handleSave = () => {
    onUpdate(editData);
    setIsEditing(false);
  };

  const handleDelete = () => {
    if (window.confirm('Are you sure you want to delete this feature request?')) {
      // onDelete would be implemented
      onClose();
    }
  };

  const handleComment = () => {
    // Comment functionality would be implemented
    console.log('Comment:', comment);
    setComment('');
  };

  if (!isOpen || !request) return null;

  const statusColors = {
    proposed: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
    under_review: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
    in_development: 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200',
    shipped: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
    wont_fix: 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300',
  };

  const priorityColors = {
    critical: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
    high: 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200',
    medium: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
    low: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg max-w-3xl w-full max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700">
          <div className="flex items-center space-x-3">
            {isEditing ? (
              <input
                type="text"
                value={editData.title}
                onChange={(e) => setEditData({ ...editData, title: e.target.value })}
                className="text-xl font-semibold bg-transparent border-b-2 border-primary-500 text-gray-900 dark:text-white focus:outline-none"
              />
            ) : (
              <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                {request.title}
              </h2>
            )}
          </div>
          
          <div className="flex items-center space-x-2">
            {user && (
              <>
                {isEditing ? (
                  <button
                    onClick={handleSave}
                    className="p-2 text-green-600 hover:bg-green-50 dark:hover:bg-green-900/20 rounded"
                    title="Save changes"
                  >
                    <FiSave />
                  </button>
                ) : (
                  <button
                    onClick={() => setIsEditing(true)}
                    className="p-2 text-gray-600 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
                    title="Edit"
                  >
                    <FiEdit2 />
                  </button>
                )}
                <button
                  onClick={handleDelete}
                  className="p-2 text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 rounded"
                  title="Delete"
                >
                  <FiTrash2 />
                </button>
              </>
            )}
            <button
              onClick={onClose}
              className="p-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
            >
              <FiX size={20} />
            </button>
          </div>
        </div>

        <div className="p-6 space-y-4">
          {/* Status and Priority */}
          <div className="flex items-center space-x-4">
            {isEditing ? (
              <>
                <select
                  value={editData.status}
                  onChange={(e) => setEditData({ ...editData, status: e.target.value })}
                  className="px-3 py-1 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-sm"
                >
                  <option value="proposed">Proposed</option>
                  <option value="under_review">Under Review</option>
                  <option value="in_development">In Development</option>
                  <option value="shipped">Shipped</option>
                  <option value="wont_fix">Won't Fix</option>
                </select>
                <select
                  value={editData.priority}
                  onChange={(e) => setEditData({ ...editData, priority: e.target.value })}
                  className="px-3 py-1 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-sm"
                >
                  <option value="low">Low</option>
                  <option value="medium">Medium</option>
                  <option value="high">High</option>
                  <option value="critical">Critical</option>
                </select>
              </>
            ) : (
              <>
                <span className={clsx('status-badge', statusColors[request.status])}>
                  {request.status.replace('_', ' ')}
                </span>
                <span className={clsx('priority-badge', priorityColors[request.priority])}>
                  {request.priority} priority
                </span>
              </>
            )}
          </div>

          {/* Description */}
          <div>
            <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Description
            </h3>
            {isEditing ? (
              <textarea
                value={editData.description}
                onChange={(e) => setEditData({ ...editData, description: e.target.value })}
                rows={6}
                className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
            ) : (
              <p className="text-gray-600 dark:text-gray-400 whitespace-pre-wrap">
                {request.description}
              </p>
            )}
          </div>

          {/* Voting */}
          <div className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
            <div className="flex items-center space-x-3">
              <button
                onClick={() => onVote(request.id, 1)}
                className={clsx(
                  'flex items-center space-x-2 px-4 py-2 rounded-lg transition-colors',
                  request.user_vote === 1
                    ? 'bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-300'
                    : 'bg-white dark:bg-gray-700 hover:bg-green-50 dark:hover:bg-green-900/20 text-gray-700 dark:text-gray-300'
                )}
              >
                <BiUpvote size={20} />
                <span>Upvote</span>
              </button>
              
              <div className="text-center">
                <div className="text-2xl font-bold text-gray-900 dark:text-white">
                  {request.vote_count}
                </div>
                <div className="text-xs text-gray-500 dark:text-gray-400">votes</div>
              </div>
              
              <button
                onClick={() => onVote(request.id, -1)}
                className={clsx(
                  'flex items-center space-x-2 px-4 py-2 rounded-lg transition-colors',
                  request.user_vote === -1
                    ? 'bg-red-100 dark:bg-red-900 text-red-700 dark:text-red-300'
                    : 'bg-white dark:bg-gray-700 hover:bg-red-50 dark:hover:bg-red-900/20 text-gray-700 dark:text-gray-300'
                )}
              >
                <BiDownvote size={20} />
                <span>Downvote</span>
              </button>
            </div>
          </div>

          {/* Tags */}
          {request.tags && request.tags.length > 0 && (
            <div>
              <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Tags
              </h3>
              <div className="flex flex-wrap gap-2">
                {request.tags.map((tag, index) => (
                  <span
                    key={index}
                    className="px-3 py-1 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 text-sm rounded-full"
                  >
                    {tag}
                  </span>
                ))}
              </div>
            </div>
          )}

          {/* Comments Section */}
          <div className="border-t border-gray-200 dark:border-gray-700 pt-4">
            <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3 flex items-center">
              <FiMessageCircle className="mr-2" />
              Comments ({request.comment_count || 0})
            </h3>
            
            {user && (
              <div className="mb-4">
                <textarea
                  value={comment}
                  onChange={(e) => setComment(e.target.value)}
                  placeholder="Add a comment..."
                  rows={3}
                  className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
                <button
                  onClick={handleComment}
                  disabled={!comment.trim()}
                  className="mt-2 px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Post Comment
                </button>
              </div>
            )}
            
            <div className="text-gray-500 dark:text-gray-400 text-sm text-center py-4">
              Comments will be available in a future update
            </div>
          </div>

          {/* Metadata */}
          <div className="text-sm text-gray-500 dark:text-gray-400 space-y-1 pt-4 border-t border-gray-200 dark:border-gray-700">
            <p>Created: {format(new Date(request.created_at), 'PPP')}</p>
            {request.updated_at !== request.created_at && (
              <p>Updated: {format(new Date(request.updated_at), 'PPP')}</p>
            )}
            {request.creator_id && (
              <p>Created by: User {request.creator_id.slice(0, 8)}</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default FeatureRequestModal;