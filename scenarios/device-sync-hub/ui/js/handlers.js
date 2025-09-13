// Event handlers and operations for Device Sync Hub

// Extend the DeviceSyncApp class with handler methods
Object.assign(DeviceSyncApp.prototype, {
  // File upload handlers
  handleFileSelect(event) {
    const files = Array.from(event.target.files);
    if (files.length > 0) {
      this.uploadFiles(files);
    }
  },

  handleDragOver(event) {
    event.preventDefault();
    event.stopPropagation();
    this.elements.fileUploadArea.classList.add('dragover');
  },

  handleDrop(event) {
    event.preventDefault();
    event.stopPropagation();
    this.elements.fileUploadArea.classList.remove('dragover');
    
    const files = Array.from(event.dataTransfer.files);
    if (files.length > 0) {
      this.uploadFiles(files);
    }
  },

  // Upload multiple files
  async uploadFiles(files) {
    const maxSize = this.settings.maxFileSize;
    const oversizedFiles = files.filter(file => file.size > maxSize);
    
    if (oversizedFiles.length > 0) {
      this.showToast(`Some files are too large (max ${Utils.formatFileSize(maxSize)})`, 'error');
      return;
    }

    this.showUploadProgress(true);
    
    let uploaded = 0;
    const total = files.length;
    
    for (const file of files) {
      try {
        await this.apiClient.uploadFile(file, this.settings.defaultExpiry);
        uploaded++;
        
        // Update progress
        const progress = (uploaded / total) * 100;
        this.updateUploadProgress(progress, `Uploading ${uploaded}/${total} files...`);
        
      } catch (error) {
        console.error('File upload failed:', error);
        this.showToast(`Failed to upload ${file.name}: ${Utils.getErrorMessage(error)}`, 'error');
      }
    }
    
    this.showUploadProgress(false);
    
    if (uploaded > 0) {
      this.showToast(`Successfully uploaded ${uploaded} file${uploaded > 1 ? 's' : ''}`, 'success');
      this.refreshItems();
    }
    
    // Clear file input
    this.elements.fileInput.value = '';
  },

  // Text upload handler
  async handleTextUpload() {
    const text = this.elements.textContent.value.trim();
    
    if (!text) {
      this.showToast('Please enter some text to share', 'warning');
      return;
    }

    this.elements.uploadTextBtn.disabled = true;
    this.elements.uploadTextBtn.textContent = 'Sharing...';
    
    try {
      await this.apiClient.uploadText(text, 'text', this.settings.defaultExpiry);
      this.showToast('Text shared successfully', 'success');
      this.elements.textContent.value = '';
      this.refreshItems();
    } catch (error) {
      console.error('Text upload failed:', error);
      this.showToast(`Failed to share text: ${Utils.getErrorMessage(error)}`, 'error');
    } finally {
      this.elements.uploadTextBtn.disabled = false;
      this.elements.uploadTextBtn.textContent = 'Share Text';
    }
  },

  // Clipboard handlers
  async handleClipboardPaste() {
    try {
      const clipboardText = await Utils.readFromClipboard();
      
      if (!clipboardText.trim()) {
        this.showToast('Clipboard is empty', 'warning');
        return;
      }

      // Show preview
      const previewContent = this.elements.clipboardPreview.querySelector('.preview-content');
      previewContent.textContent = Utils.truncateText(clipboardText, 200);
      this.elements.clipboardPreview.classList.remove('hidden');
      
      // Store clipboard content for sharing
      this._clipboardContent = clipboardText;
      
    } catch (error) {
      console.error('Failed to read clipboard:', error);
      this.showToast('Could not access clipboard. Please paste manually.', 'error');
    }
  },

  async handleClipboardShare() {
    if (!this._clipboardContent) {
      this.showToast('No clipboard content to share', 'warning');
      return;
    }

    this.elements.shareClipboardBtn.disabled = true;
    this.elements.shareClipboardBtn.textContent = 'Sharing...';
    
    try {
      await this.apiClient.uploadText(this._clipboardContent, 'clipboard', this.settings.defaultExpiry);
      this.showToast('Clipboard content shared successfully', 'success');
      this.handleClipboardClear();
      this.refreshItems();
    } catch (error) {
      console.error('Clipboard share failed:', error);
      this.showToast(`Failed to share clipboard: ${Utils.getErrorMessage(error)}`, 'error');
    } finally {
      this.elements.shareClipboardBtn.disabled = false;
      this.elements.shareClipboardBtn.textContent = 'Share Clipboard';
    }
  },

  handleClipboardClear() {
    this.elements.clipboardPreview.classList.add('hidden');
    this._clipboardContent = null;
  },

  // Upload progress
  showUploadProgress(show) {
    this.elements.uploadProgress.classList.toggle('hidden', !show);
    if (!show) {
      this.updateUploadProgress(0, '');
    }
  },

  updateUploadProgress(percent, text) {
    const progressFill = this.elements.uploadProgress.querySelector('.progress-fill');
    const progressText = this.elements.uploadProgress.querySelector('.progress-text');
    
    progressFill.style.width = `${percent}%`;
    progressText.textContent = text;
  },

  // Items management
  async refreshItems() {
    try {
      const response = await this.apiClient.getSyncItems();
      this.syncItems = response.items || response; // Handle different response formats
      this.renderItems();
    } catch (error) {
      console.error('Failed to refresh items:', error);
      this.showToast('Failed to load sync items', 'error');
    }
  },

  handleFilterChange(event) {
    this.currentFilter = event.target.value;
    this.renderItems();
  },

  // Render items list
  renderItems() {
    const filteredItems = this.currentFilter === 'all' 
      ? this.syncItems 
      : this.syncItems.filter(item => item.type === this.currentFilter);

    if (filteredItems.length === 0) {
      this.elements.itemsList.innerHTML = '';
      this.elements.emptyState.classList.remove('hidden');
      return;
    }

    this.elements.emptyState.classList.add('hidden');

    const itemsHtml = filteredItems.map(item => this.createItemHtml(item)).join('');
    this.elements.itemsList.innerHTML = itemsHtml;

    // Add event listeners to item buttons
    this.elements.itemsList.querySelectorAll('[data-action]').forEach(button => {
      button.addEventListener('click', (e) => this.handleItemAction(e));
    });
  },

  // Create HTML for a sync item
  createItemHtml(item) {
    const icon = Utils.getContentTypeIcon(item.type);
    const fileName = item.content?.filename || item.content?.text?.substring(0, 30) || 'Unknown';
    const fileSize = item.content?.file_size ? Utils.formatFileSize(item.content.file_size) : '';
    const createdAt = Utils.formatDate(item.created_at);
    const expiresAt = Utils.formatExpiration(item.expires_at);

    return `
      <div class="sync-item" data-item-id="${item.id}">
        <div class="item-icon">${icon}</div>
        <div class="item-details">
          <div class="item-name">${Utils.escapeHtml(fileName)}</div>
          <div class="item-meta">
            <span class="item-type">${item.type}</span>
            ${fileSize ? `<span class="item-size">${fileSize}</span>` : ''}
            <span class="item-date">${createdAt}</span>
            <span class="item-expires ${expiresAt.includes('left') ? '' : 'expired'}">${expiresAt}</span>
          </div>
        </div>
        <div class="item-actions">
          ${item.type === 'file' ? `<button class="btn btn-sm" data-action="download" data-item-id="${item.id}" title="Download">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 10v6m0 0l-4-4m4 4l4-4m-4 4V3"/>
            </svg>
          </button>` : ''}
          ${item.type !== 'file' ? `<button class="btn btn-sm" data-action="copy" data-item-id="${item.id}" title="Copy to clipboard">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/>
            </svg>
          </button>` : ''}
          <button class="btn btn-sm btn-danger" data-action="delete" data-item-id="${item.id}" title="Delete">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
            </svg>
          </button>
        </div>
      </div>
    `;
  },

  // Handle item actions (download, copy, delete)
  async handleItemAction(event) {
    const button = event.target.closest('button');
    const action = button.dataset.action;
    const itemId = button.dataset.itemId;
    const item = this.syncItems.find(i => i.id === itemId);

    if (!item) {
      this.showToast('Item not found', 'error');
      return;
    }

    button.disabled = true;

    try {
      switch (action) {
        case 'download':
          await this.downloadItem(item);
          break;
        case 'copy':
          await this.copyItemToClipboard(item);
          break;
        case 'delete':
          await this.deleteItem(item);
          break;
      }
    } catch (error) {
      console.error(`${action} action failed:`, error);
      this.showToast(`Failed to ${action} item: ${Utils.getErrorMessage(error)}`, 'error');
    } finally {
      button.disabled = false;
    }
  },

  // Download item
  async downloadItem(item) {
    const download = await this.apiClient.downloadSyncItem(item.id);
    Utils.downloadBlob(download.blob, download.filename);
    this.showToast('File downloaded successfully', 'success');
  },

  // Copy item content to clipboard
  async copyItemToClipboard(item) {
    const text = item.content?.text || JSON.stringify(item.content);
    const success = await Utils.copyToClipboard(text);
    
    if (success) {
      this.showToast('Copied to clipboard', 'success');
    } else {
      this.showToast('Failed to copy to clipboard', 'error');
    }
  },

  // Delete item
  async deleteItem(item) {
    if (!confirm('Are you sure you want to delete this item?')) {
      return;
    }

    await this.apiClient.deleteSyncItem(item.id);
    this.showToast('Item deleted', 'success');
    this.refreshItems();
  },

  // WebSocket message handlers
  handleItemAdded(item) {
    // Add to local list if not already present
    if (!this.syncItems.find(i => i.id === item.id)) {
      this.syncItems.unshift(item);
      this.renderItems();
      this.showToast('New item synced from another device', 'info');
    }
  },

  handleItemDeleted(itemId) {
    const index = this.syncItems.findIndex(i => i.id === itemId);
    if (index !== -1) {
      this.syncItems.splice(index, 1);
      this.renderItems();
    }
  },

  handleItemUpdated(item) {
    const index = this.syncItems.findIndex(i => i.id === item.id);
    if (index !== -1) {
      this.syncItems[index] = item;
      this.renderItems();
    }
  }
});