import { logger } from '../utils/logger.js';

export class TagMultiSelect {
    constructor({
        selectElement,
        wrapperElement,
        selectionElement,
        tagsContainer,
        searchInput,
        dropdownElement,
        optionsContainer,
        statusElement,
        placeholder = ''
    }) {
        this.select = selectElement;
        this.wrapper = wrapperElement;
        this.selection = selectionElement;
        this.tagsContainer = tagsContainer;
        this.searchInput = searchInput;
        this.dropdown = dropdownElement;
        this.optionsContainer = optionsContainer;
        this.statusElement = statusElement;
        this.placeholder = placeholder;

        this.optionElements = new Map();
        this.optionsData = new Map();
        this.filteredOptions = [];
        this.highlightIndex = -1;
        this.disabled = Boolean(this.select?.disabled);
        this.isOpen = false;
        this.emptyMessage = 'No options available';
        this.noResultsMessage = 'No options match that search';
        this.stateMessage = '';
        this.stateTone = 'info';
        this.showingStateMessage = false;
        this.stateIsBusy = false;
        this.dropdownParent = this.dropdown ? this.dropdown.parentElement : null;
        this.portalActive = false;
        this.portalMargin = 8;
        this.boundRepositionDropdown = this.repositionDropdown.bind(this);

        this.handleDocumentClick = this.handleDocumentClick.bind(this);

        this.init();
    }

    init() {
        if (!this.select || !this.wrapper || !this.selection || !this.searchInput || !this.dropdown || !this.optionsContainer) {
            logger.warn('TagMultiSelect initialization skipped due to missing elements');
            return;
        }

        this.select.classList.add('tag-multiselect-hidden-select');
        this.select.setAttribute('aria-hidden', 'true');
        this.select.tabIndex = -1;

        this.wrapper.classList.add('tag-multiselect-enhanced');
        this.searchInput.placeholder = this.placeholder;
        this.searchInput.disabled = this.disabled;
        this.selection.setAttribute('aria-expanded', 'false');
        this.selection.setAttribute('aria-disabled', this.disabled ? 'true' : 'false');
        this.selection.setAttribute('aria-busy', 'false');
        this.statusElement.hidden = true;
        this.dropdown.hidden = true;

        this.selection.addEventListener('click', (event) => {
            if (event.target.closest('.tag-multiselect-remove')) {
                return;
            }

            if (this.disabled) {
                return;
            }

            if (!this.isOpen) {
                this.openDropdown();
            }
            this.focusSearchInput();
        });

        this.searchInput.addEventListener('focus', () => {
            if (!this.isOpen && !this.disabled) {
                this.openDropdown();
            }
        });

        this.searchInput.addEventListener('input', (event) => {
            this.filterOptions(event.target.value);
        });

        this.searchInput.addEventListener('keydown', (event) => {
            this.handleKeyDown(event);
        });

        document.addEventListener('click', this.handleDocumentClick);

        this.updateTags();
        this.refreshFromSelect();
    }

    setEmptyMessage(message) {
        this.emptyMessage = message || 'No options available';
    }

    setNoResultsMessage(message) {
        this.noResultsMessage = message || 'No options match that search';
    }

    setStatus(message, tone = 'info') {
        if (!this.statusElement) {
            return;
        }

        if (!message) {
            this.statusElement.textContent = '';
            this.statusElement.dataset.tone = 'info';
            this.statusElement.hidden = true;
            return;
        }

        this.statusElement.textContent = message;
        this.statusElement.dataset.tone = tone;
        this.statusElement.hidden = false;
    }

    setDisabled(disabled, options = {}) {
        this.disabled = Boolean(disabled);
        this.searchInput.disabled = this.disabled;
        this.wrapper.classList.toggle('is-disabled', this.disabled);
        this.selection.setAttribute('aria-disabled', this.disabled ? 'true' : 'false');

        if (this.disabled && options.message) {
            this.stateMessage = options.message;
            this.stateTone = options.tone || 'info';
            this.showingStateMessage = true;
            this.stateIsBusy = options.reason === 'loading';
        } else if (!this.disabled) {
            this.stateMessage = '';
            this.stateTone = 'info';
            this.showingStateMessage = false;
            this.stateIsBusy = false;
        } else if (this.disabled && !options.message) {
            this.showingStateMessage = false;
            this.stateIsBusy = false;
        }

        this.selection.setAttribute('aria-busy', this.stateIsBusy ? 'true' : 'false');
        this.wrapper.classList.toggle('is-busy', this.stateIsBusy);
        this.updateTags();

        if (this.disabled) {
            this.closeDropdown();
        }
    }

    clearOptions() {
        this.optionElements.clear();
        this.optionsData.clear();
        this.filteredOptions = [];
        this.optionsContainer.innerHTML = '';
        this.highlightIndex = -1;
        if (this.searchInput) {
            this.searchInput.value = '';
        }
        this.updateTags();
        this.repositionDropdown();
    }

    refreshFromSelect(preserveHighlightValue = null) {
        if (!this.select) {
            return;
        }

        const currentQuery = this.searchInput.value || '';
        const highlightValue = preserveHighlightValue || this.getHighlightedValue();

        this.optionElements.clear();
        this.optionsData.clear();
        this.filteredOptions = [];
        this.optionsContainer.innerHTML = '';

        const options = Array.from(this.select.options || []).filter(option => option.value);

        if (options.length === 0) {
            this.highlightIndex = -1;
            this.updateTags();
            if (!this.showingStateMessage) {
                this.setStatus(this.emptyMessage, 'empty');
            }
            this.repositionDropdown();
            return;
        }

        this.setStatus('', 'info');

        options.forEach((option, index) => {
            const data = {
                value: option.value,
                label: option.textContent,
                disabled: option.disabled,
                selected: option.selected,
                searchText: `${option.textContent || ''} ${option.value}`.toLowerCase()
            };

            this.optionsData.set(option.value, data);

            const optionElement = document.createElement('div');
            optionElement.className = 'tag-multiselect-option';
            optionElement.dataset.value = option.value;
            optionElement.id = `${this.select.id}-option-${index}`;
            optionElement.setAttribute('role', 'option');
            optionElement.setAttribute('aria-selected', String(option.selected));

            if (data.disabled) {
                optionElement.classList.add('is-disabled');
                optionElement.setAttribute('aria-disabled', 'true');
            } else {
                optionElement.tabIndex = -1;
                optionElement.addEventListener('mousedown', (event) => event.preventDefault());
                optionElement.addEventListener('click', (event) => {
                    event.preventDefault();
                    this.toggleOption(data.value);
                });
            }

            if (data.selected) {
                optionElement.classList.add('is-selected');
            }

            if (option.classList.contains('target-unavailable')) {
                optionElement.classList.add('is-unavailable');
            }

            const check = document.createElement('span');
            check.className = 'tag-multiselect-option-check';
            if (data.selected && !data.disabled) {
                const icon = document.createElement('i');
                icon.className = 'fas fa-check';
                check.appendChild(icon);
            }

            const label = document.createElement('span');
            label.className = 'tag-multiselect-option-label';
            label.textContent = data.label;

            optionElement.appendChild(check);
            optionElement.appendChild(label);

            this.optionsContainer.appendChild(optionElement);
            this.optionElements.set(option.value, optionElement);
        });

        this.filterOptions(currentQuery, { preserveHighlightValue: highlightValue });
        this.updateTags();
    }

    filterOptions(query, options = {}) {
        const normalizedQuery = (query || '').trim().toLowerCase();
        const filtered = [];

        this.optionElements.forEach((element, value) => {
            const optionData = this.optionsData.get(value);
            if (!optionData) {
                element.classList.add('is-hidden');
                return;
            }

            const matches = !normalizedQuery || optionData.searchText.includes(normalizedQuery);
            element.classList.toggle('is-hidden', !matches);

            if (matches) {
                filtered.push({
                    value,
                    element,
                    disabled: optionData.disabled
                });
            }
        });

        this.filteredOptions = filtered;

        if (filtered.length === 0) {
            if (this.optionsData.size === 0) {
                this.setStatus(this.emptyMessage, 'empty');
            } else if (normalizedQuery) {
                this.setStatus(this.noResultsMessage, 'empty');
            } else {
                this.setStatus('', 'info');
            }

            this.highlightIndex = -1;
            this.applyHighlight();
            return;
        }

        this.setStatus('', 'info');

        if (options.preserveHighlightValue) {
            const existingIndex = filtered.findIndex(item => item.value === options.preserveHighlightValue && !item.disabled);
            this.highlightIndex = existingIndex >= 0 ? existingIndex : this.findFirstSelectableIndex(filtered);
        } else if (this.highlightIndex === -1 || this.highlightIndex >= filtered.length || filtered[this.highlightIndex]?.disabled) {
            this.highlightIndex = this.findFirstSelectableIndex(filtered);
        }

        this.applyHighlight();
        this.repositionDropdown();
    }

    findFirstSelectableIndex(list) {
        return list.findIndex(item => !item.disabled);
    }

    updateTags() {
        if (!this.tagsContainer) {
            return;
        }

        this.tagsContainer.innerHTML = '';

        if (this.showingStateMessage) {
            this.tagsContainer.classList.remove('is-empty');
            this.tagsContainer.classList.add('is-placeholder');

            const placeholder = document.createElement('span');
            const toneSuffix = this.stateTone ? ` tag-multiselect-placeholder--${this.stateTone}` : '';
            placeholder.className = `tag-multiselect-placeholder${toneSuffix}`;
            placeholder.textContent = this.stateMessage || this.placeholder;
            this.tagsContainer.appendChild(placeholder);
            this.searchInput.placeholder = '';
            if (this.searchInput) {
                this.searchInput.style.display = 'none';
            }
            return;
        }

        if (this.searchInput) {
            this.searchInput.style.display = '';
        }
        this.tagsContainer.classList.remove('is-placeholder');

        const selected = Array.from(this.select.selectedOptions || [])
            .filter(option => option.value && !option.disabled);

        if (!selected.length) {
            this.tagsContainer.classList.add('is-empty');
            this.searchInput.placeholder = this.placeholder;
            return;
        }

        this.tagsContainer.classList.remove('is-empty');
        this.searchInput.placeholder = '';

        selected.forEach(option => {
            const tag = document.createElement('span');
            tag.className = 'tag-multiselect-tag';
            tag.dataset.value = option.value;
            tag.title = option.textContent;

            const label = document.createElement('span');
            label.className = 'tag-multiselect-tag-label';
            label.textContent = option.textContent;

            const removeButton = document.createElement('button');
            removeButton.type = 'button';
            removeButton.className = 'tag-multiselect-remove';
            removeButton.setAttribute('aria-label', `Remove ${option.textContent}`);

            const icon = document.createElement('i');
            icon.className = 'fas fa-times';
            removeButton.appendChild(icon);

            removeButton.addEventListener('click', (event) => {
                event.preventDefault();
                this.removeOption(option.value);
            });

            tag.appendChild(label);
            tag.appendChild(removeButton);
            this.tagsContainer.appendChild(tag);
        });
    }

    getSelectedValues() {
        return Array.from(this.select.selectedOptions || [])
            .filter(option => option.value)
            .map(option => option.value);
    }

    toggleOption(value) {
        const option = Array.from(this.select.options || []).find(opt => opt.value === value);
        if (!option || option.disabled) {
            return;
        }

        const highlightValue = this.getHighlightedValue();
        option.selected = !option.selected;
        this.refreshFromSelect(highlightValue);
        this.dispatchChange();
    }

    removeOption(value) {
        const option = Array.from(this.select.options || []).find(opt => opt.value === value);
        if (!option || option.disabled) {
            return;
        }

        const highlightValue = this.getHighlightedValue();
        option.selected = false;
        this.refreshFromSelect(highlightValue);
        this.dispatchChange();
        this.focusSearchInput();
    }

    handleKeyDown(event) {
        switch (event.key) {
            case 'ArrowDown':
                event.preventDefault();
                if (!this.isOpen) {
                    this.openDropdown();
                }
                this.moveHighlight(1);
                break;
            case 'ArrowUp':
                event.preventDefault();
                if (!this.isOpen) {
                    this.openDropdown();
                }
                this.moveHighlight(-1);
                break;
            case 'Enter':
                if (this.isOpen && this.highlightIndex !== -1) {
                    const highlighted = this.filteredOptions[this.highlightIndex];
                    if (highlighted && !highlighted.disabled) {
                        event.preventDefault();
                        this.toggleOption(highlighted.value);
                    }
                }
                break;
            case 'Escape':
                if (this.isOpen) {
                    event.preventDefault();
                    this.closeDropdown(true);
                }
                break;
            case 'Backspace':
                if (!this.searchInput.value) {
                    const selectedValues = this.getSelectedValues();
                    const lastValue = selectedValues[selectedValues.length - 1];
                    if (lastValue) {
                        event.preventDefault();
                        this.removeOption(lastValue);
                    }
                }
                break;
            case 'Tab':
                this.closeDropdown();
                break;
            default:
                break;
        }
    }

    moveHighlight(step) {
        if (!this.filteredOptions.length) {
            this.highlightIndex = -1;
            this.applyHighlight();
            return;
        }

        const length = this.filteredOptions.length;
        let newIndex = this.highlightIndex;
        let attempts = 0;

        do {
            if (newIndex === -1) {
                newIndex = step > 0 ? 0 : length - 1;
            } else {
                newIndex = (newIndex + step + length) % length;
            }
            attempts++;
        } while (this.filteredOptions[newIndex]?.disabled && attempts <= length);

        if (attempts > length) {
            this.highlightIndex = -1;
        } else {
            this.highlightIndex = newIndex;
        }

        this.applyHighlight();
    }

    getHighlightedValue() {
        if (this.highlightIndex === -1) {
            return null;
        }

        return this.filteredOptions[this.highlightIndex]?.value || null;
    }

    setHighlightByValue(value) {
        if (!value) {
            this.highlightIndex = -1;
            this.applyHighlight();
            return;
        }

        const index = this.filteredOptions.findIndex(item => item.value === value && !item.disabled);
        this.highlightIndex = index;
        this.applyHighlight();
    }

    applyHighlight() {
        this.optionElements.forEach((element) => {
            element.classList.remove('is-highlighted');
        });

        const highlighted = this.filteredOptions[this.highlightIndex];
        if (!highlighted) {
            this.searchInput.removeAttribute('aria-activedescendant');
            return;
        }

        highlighted.element.classList.add('is-highlighted');
        this.searchInput.setAttribute('aria-activedescendant', highlighted.element.id);
        this.ensureOptionInView(highlighted.element);
    }

    ensureOptionInView(element) {
        if (!element || !this.dropdown || this.dropdown.hidden) {
            return;
        }

        const container = this.dropdown;
        const elementTop = element.offsetTop;
        const elementBottom = elementTop + element.offsetHeight;
        const viewTop = container.scrollTop;
        const viewBottom = viewTop + container.clientHeight;

        if (elementTop < viewTop) {
            container.scrollTop = elementTop;
        } else if (elementBottom > viewBottom) {
            container.scrollTop = elementBottom - container.clientHeight;
        }
    }

    ensureDropdownPortal() {
        if (this.portalActive || !this.dropdown) {
            return;
        }

        if (!this.dropdownParent) {
            this.dropdownParent = this.dropdown.parentElement;
        }

        document.body.appendChild(this.dropdown);
        this.dropdown.classList.add('tag-multiselect-dropdown-portal');
        this.portalActive = true;
        window.addEventListener('resize', this.boundRepositionDropdown);
        window.addEventListener('scroll', this.boundRepositionDropdown, true);
    }

    teardownDropdownPortal() {
        if (!this.portalActive || !this.dropdownParent) {
            return;
        }

        window.removeEventListener('resize', this.boundRepositionDropdown);
        window.removeEventListener('scroll', this.boundRepositionDropdown, true);

        this.dropdownParent.appendChild(this.dropdown);
        this.dropdown.classList.remove('tag-multiselect-dropdown-portal', 'is-flipped');
        this.dropdown.style.top = '';
        this.dropdown.style.left = '';
        this.dropdown.style.width = '';
        this.dropdown.style.maxHeight = '';
        this.portalActive = false;
        if (this.wrapper) {
            this.wrapper.classList.remove('dropdown-open-top');
        }
    }

    repositionDropdown() {
        if (!this.portalActive || !this.dropdown || !this.selection) {
            return;
        }

        const rect = this.selection.getBoundingClientRect();
        const viewportWidth = window.innerWidth;
        const viewportHeight = window.innerHeight;
        const margin = this.portalMargin;

        const maxWidth = Math.max(200, viewportWidth - margin * 2);
        const targetWidth = Math.min(rect.width, maxWidth);
        let left = rect.left;

        if (left + targetWidth > viewportWidth - margin) {
            left = viewportWidth - targetWidth - margin;
        }
        left = Math.max(margin, left);

        this.dropdown.style.width = `${Math.round(targetWidth)}px`;
        this.dropdown.style.maxHeight = `${Math.max(180, Math.round(viewportHeight - margin * 2))}px`;

        // Force layout to get accurate height after width adjustment
        const dropdownRect = this.dropdown.getBoundingClientRect();
        let dropdownHeight = dropdownRect.height || 0;

        if (!dropdownHeight) {
            dropdownHeight = Math.min(320, viewportHeight - margin * 2);
        }

        let top = rect.bottom + margin;
        let placement = 'bottom';

        if (top + dropdownHeight > viewportHeight - margin) {
            const spaceBelow = viewportHeight - rect.bottom - margin;
            const spaceAbove = rect.top - margin;
            if (spaceAbove > spaceBelow) {
                top = Math.max(margin, rect.top - margin - dropdownHeight);
                placement = 'top';
            } else {
                top = Math.max(margin, Math.min(viewportHeight - margin - dropdownHeight, rect.bottom + margin));
            }
        }

        this.dropdown.style.top = `${Math.round(top)}px`;
        this.dropdown.style.left = `${Math.round(left)}px`;

        if (placement === 'top') {
            this.dropdown.classList.add('is-flipped');
            this.wrapper.classList.add('dropdown-open-top');
        } else {
            this.dropdown.classList.remove('is-flipped');
            this.wrapper.classList.remove('dropdown-open-top');
        }
    }

    openDropdown() {
        if (this.disabled || this.isOpen) {
            return;
        }

        this.isOpen = true;
        this.ensureDropdownPortal();
        this.dropdown.hidden = false;
        this.wrapper.classList.add('is-open');
        this.selection.setAttribute('aria-expanded', 'true');
        this.filterOptions(this.searchInput.value || '');
        this.dropdown.scrollTop = 0;
        this.repositionDropdown();
    }

    closeDropdown(focusInput = false) {
        if (!this.isOpen) {
            return;
        }

        this.isOpen = false;
        this.dropdown.hidden = true;
        this.wrapper.classList.remove('is-open');
        this.selection.setAttribute('aria-expanded', 'false');
        this.wrapper.classList.remove('dropdown-open-top');
        this.teardownDropdownPortal();

        if (focusInput) {
            this.focusSearchInput();
        }
    }

    handleDocumentClick(event) {
        if (!this.wrapper.contains(event.target) && !(this.dropdown && this.dropdown.contains(event.target))) {
            this.closeDropdown();
        }
    }

    focusSearchInput() {
        if (this.searchInput.disabled) {
            return;
        }

        this.searchInput.focus({ preventScroll: true });
    }

    dispatchChange() {
        // Keep the native select in sync with observers
        if (this.select) {
            this.select.dispatchEvent(new Event('change', { bubbles: true }));
        }
    }
}