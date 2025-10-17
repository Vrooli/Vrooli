import React from 'react';
import { DragDropContext, Droppable, Draggable } from 'react-beautiful-dnd';
import { BiUpvote, BiDownvote } from 'react-icons/bi';
import { MdDragIndicator, MdComment } from 'react-icons/md';
import { format } from 'date-fns';
import clsx from 'clsx';

const PriorityBadge = ({ priority }) => {
  const colors = {
    critical: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
    high: 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200',
    medium: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
    low: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
  };

  return (
    <span className={clsx('priority-badge', colors[priority] || colors.medium)}>
      {priority}
    </span>
  );
};

const FeatureCard = ({ request, onVote, onClick, user, isDragging, provided }) => {
  const handleVote = (e, value) => {
    e.stopPropagation();
    onVote(request.id, value);
  };

  return (
    <div
      ref={provided.innerRef}
      {...provided.draggableProps}
      className={clsx(
        'feature-card',
        isDragging && 'opacity-50 rotate-2 scale-105'
      )}
      onClick={() => onClick(request)}
    >
      <div className="flex items-start justify-between mb-2">
        <div className="flex-1">
          <div className="flex items-center space-x-2 mb-1">
            <div
              {...provided.dragHandleProps}
              className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 cursor-move"
            >
              <MdDragIndicator />
            </div>
            <h3 className="font-semibold text-gray-900 dark:text-white line-clamp-2">
              {request.title}
            </h3>
          </div>
          <p className="text-sm text-gray-600 dark:text-gray-400 line-clamp-2 ml-6">
            {request.description}
          </p>
        </div>
      </div>

      <div className="flex items-center justify-between mt-3">
        <div className="flex items-center space-x-2">
          <PriorityBadge priority={request.priority} />
          {request.tags && request.tags.length > 0 && (
            <div className="flex space-x-1">
              {request.tags.slice(0, 2).map((tag, index) => (
                <span
                  key={index}
                  className="px-2 py-0.5 bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 text-xs rounded"
                >
                  {tag}
                </span>
              ))}
              {request.tags.length > 2 && (
                <span className="text-xs text-gray-500">+{request.tags.length - 2}</span>
              )}
            </div>
          )}
        </div>

        <div className="flex items-center space-x-3">
          {request.comment_count > 0 && (
            <div className="flex items-center space-x-1 text-gray-500 dark:text-gray-400">
              <MdComment className="text-sm" />
              <span className="text-xs">{request.comment_count}</span>
            </div>
          )}
          
          <div className="flex items-center space-x-1">
            <button
              onClick={(e) => handleVote(e, 1)}
              className={clsx(
                'vote-button',
                request.user_vote === 1 ? 'vote-button-active-up' : 'vote-button-up'
              )}
              title="Upvote"
            >
              <BiUpvote />
            </button>
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300 min-w-[2ch] text-center">
              {request.vote_count}
            </span>
            <button
              onClick={(e) => handleVote(e, -1)}
              className={clsx(
                'vote-button',
                request.user_vote === -1 ? 'vote-button-active-down' : 'vote-button-down'
              )}
              title="Downvote"
            >
              <BiDownvote />
            </button>
          </div>
        </div>
      </div>

      <div className="mt-2 text-xs text-gray-500 dark:text-gray-400">
        {format(new Date(request.created_at), 'MMM d, yyyy')}
      </div>
    </div>
  );
};

const KanbanColumn = ({ status, requests, onVote, onRequestClick, user }) => {
  return (
    <div className="flex-1 min-w-[300px]">
      <div className="flex items-center justify-between mb-3">
        <h3 className="font-semibold text-gray-700 dark:text-gray-300">
          {status.label}
        </h3>
        <span className={clsx('status-badge', status.color)}>
          {requests.length}
        </span>
      </div>

      <Droppable droppableId={status.id}>
        {(provided, snapshot) => (
          <div
            ref={provided.innerRef}
            {...provided.droppableProps}
            className={clsx(
              'kanban-column',
              snapshot.isDraggingOver && 'bg-primary-50 dark:bg-primary-900/20 border-2 border-dashed border-primary-300 dark:border-primary-700'
            )}
          >
            {requests.map((request, index) => (
              <Draggable
                key={request.id}
                draggableId={request.id}
                index={index}
              >
                {(provided, snapshot) => (
                  <FeatureCard
                    request={request}
                    onVote={onVote}
                    onClick={onRequestClick}
                    user={user}
                    isDragging={snapshot.isDragging}
                    provided={provided}
                  />
                )}
              </Draggable>
            ))}
            {provided.placeholder}
            
            {requests.length === 0 && (
              <div className="text-center py-8 text-gray-400 dark:text-gray-500">
                No requests in this status
              </div>
            )}
          </div>
        )}
      </Droppable>
    </div>
  );
};

const KanbanBoard = ({ statuses, groupedRequests, onDragEnd, onVote, onRequestClick, user }) => {
  return (
    <DragDropContext onDragEnd={onDragEnd}>
      <div className="flex gap-4 overflow-x-auto pb-4">
        {statuses.map(status => (
          <KanbanColumn
            key={status.id}
            status={status}
            requests={groupedRequests[status.id] || []}
            onVote={onVote}
            onRequestClick={onRequestClick}
            user={user}
          />
        ))}
      </div>
    </DragDropContext>
  );
};

export default KanbanBoard;