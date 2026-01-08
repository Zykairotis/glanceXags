import { elem, fragment } from "./templating.js";

const trashIconSvg = `<svg fill="currentColor" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20">
  <path fill-rule="evenodd" d="M8.75 1A2.75 2.75 0 006 3.75v.443c-.795.077-1.584.176-2.365.298a.75.75 0 10.23 1.482l.149-.022.841 10.518A2.75 2.75 0 007.596 19h4.807a2.75 2.75 0 002.742-2.53l.841-10.52.149.023a.75.75 0 00.23-1.482A41.03 41.03 0 0014 4.193V3.75A2.75 2.75 0 0011.25 1h-2.5zM10 4c.84 0 1.673.025 2.5.075V3.75c0-.69-.56-1.25-1.25-1.25h-2.5c-.69 0-1.25.56-1.25 1.25v.325C8.327 4.025 9.16 4 10 4zM8.58 7.72a.75.75 0 00-1.5.06l.3 7.5a.75.75 0 101.5-.06l-.3-7.5zm4.34.06a.75.75 0 10-1.5-.06l-.3 7.5a.75.75 0 101.5.06l.3-7.5z" clip-rule="evenodd" />
</svg>`;

/**
 * Lightweight markdown parser that supports common markdown syntax.
 * Only allows safe HTML tags to prevent XSS attacks.
 * @param {string} text - The markdown text to parse
 * @returns {string} - Sanitized HTML string
 */
function parseMarkdown(text) {
    if (!text) return '';

    // First, escape HTML to prevent XSS
    let html = escapeHtml(text);

    // Code blocks (must be processed before other markdown)
    html = html.replace(/```(\w*)([\s\S]*?)```/g, (match, lang, code) => {
        return `<pre><code>${code.trim()}</code></pre>`;
    });

    // Inline code
    html = html.replace(/`([^`]+)`/g, '<code>$1</code>');

    // Headers (must be at start of line)
    html = html.replace(/^### (.+)$/gm, '<h3>$1</h3>');
    html = html.replace(/^## (.+)$/gm, '<h2>$1</h2>');
    html = html.replace(/^# (.+)$/gm, '<h1>$1</h1>');

    // Strikethrough: ~~text~~
    html = html.replace(/~~([^~]+)~~/g, '<s>$1</s>');

    // Bold: **text** or __text__
    html = html.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
    html = html.replace(/__([^_]+)__/g, '<strong>$1</strong>');

    // Italic: *text* or _text_ (but not inside words)
    html = html.replace(/\*([^*]+)\*/g, '<em>$1</em>');
    html = html.replace(/_([^_]+)_/g, '<em>$1</em>');

    // Links: [text](url)
    html = html.replace(/\[([^\]]+)\]\(([^)]+)\)/g, (match, text, url) => {
        // Validate URL to prevent javascript: and data: schemes
        try {
            const parsed = new URL(url);
            if (parsed.protocol === 'javascript:' || parsed.protocol === 'data:') {
                return match; // Return original if unsafe
            }
        } catch (e) {
            return match; // Return original if invalid URL
        }
        return `<a href="${escapeHtml(url)}" target="_blank" rel="noopener noreferrer">${text}</a>`;
    });

    // Bullet lists
    html = html.replace(/^- (.+)$/gm, '<li>$1</li>');
    html = html.replace(/(<li>.*<\/li>\n?)+/g, '<ul>$&</ul>');

    // Numbered lists  
    html = html.replace(/^\d+\. (.+)$/gm, '<li>$1</li>');

    // Line breaks and paragraphs
    html = html.replace(/\n\n/g, '</p><p>');
    html = html.replace(/\n/g, '<br>');

    // Wrap in paragraph if not already wrapped
    if (!html.startsWith('<')) {
        html = '<p>' + html + '</p>';
    }

    return html;
}

/**
 * Rich Text Editor class for editing comments, titles, and descriptions
 */
class RichTextEditor {
    constructor(widget, apiBase) {
        this.widget = widget;
        this.apiBase = apiBase;
        this.modal = widget.querySelector('.todoist-editor-modal');
        if (!this.modal) return;
        
        this.input = this.modal.querySelector('.editor-input');
        this.preview = this.modal.querySelector('.editor-preview');
        this.titleEl = this.modal.querySelector('.editor-title');
        this.currentContext = null; // { type: 'comment'|'task'|'description', id: string, element: Element }
        
        this.setupToolbar();
        this.setupKeyboardShortcuts();
        this.setupLivePreview();
        this.setupButtons();
    }
    
    open(context, content, title) {
        if (!this.modal) return;
        this.currentContext = context;
        this.input.value = content || '';
        this.titleEl.textContent = title || 'Edit';
        this.updatePreview();
        this.modal.classList.remove('collapsed');
        this.input.focus();
    }
    
    close() {
        if (!this.modal) return;
        this.modal.classList.add('collapsed');
        this.currentContext = null;
        this.input.value = '';
        this.preview.innerHTML = '';
    }
    
    async save() {
        if (!this.currentContext) return;
        
        const content = this.input.value.trim();
        if (!content && this.currentContext.type !== 'description') {
            showNotification('Content cannot be empty', 'error');
            return;
        }
        
        const { type, id, element } = this.currentContext;
        
        try {
            switch (type) {
                case 'comment':
                    await this.saveComment(id, content, element);
                    break;
                case 'task':
                    await this.saveTaskContent(id, content, element);
                    break;
                case 'description':
                    await this.saveTaskDescription(id, content, element);
                    break;
                case 'new-comment':
                    await this.createNewComment(id, content, this.currentContext.inputEl, element);
                    break;
            }
            this.close();
        } catch (error) {
            console.error('Failed to save:', error);
            showNotification('Failed to save changes', 'error');
        }
    }
    
    async saveComment(commentId, content, element) {
        const response = await fetch(`${this.apiBase}/comments/${commentId}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ content })
        });
        
        if (!response.ok) throw new Error('Failed to update comment');
        
        // Update the comment element
        if (element) {
            element.dataset.rawContent = content;
            const contentEl = element.querySelector('.todoist-comment-content');
            if (contentEl) {
                contentEl.innerHTML = parseMarkdown(content);
            }
        }
        showNotification('Comment updated');
    }
    
    async saveTaskContent(taskId, content, element) {
        const response = await fetch(`${this.apiBase}/tasks/${taskId}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ content })
        });
        
        if (!response.ok) throw new Error('Failed to update task');
        
        // Update the task element
        if (element) {
            element.dataset.content = content;
            const titleEl = element.querySelector('.title');
            if (titleEl) {
                titleEl.textContent = content;
            }
        }
        showNotification('Task updated');
    }
    
    async saveTaskDescription(taskId, content, element) {
        const response = await fetch(`${this.apiBase}/tasks/${taskId}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ description: content })
        });
        
        if (!response.ok) throw new Error('Failed to update description');
        
        // Update the description element
        if (element) {
            element.dataset.description = content;
            const descEl = element.querySelector('.todoist-task-description');
            if (descEl) {
                descEl.textContent = content;
            }
        }
        showNotification('Description updated');
    }
    
    async createNewComment(taskId, content, inputEl, section) {
        const response = await fetch(`${this.apiBase}/tasks/${taskId}/comments`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ content })
        });
        
        if (!response.ok) throw new Error('Failed to create comment');
        
        // Clear the input field
        if (inputEl) {
            inputEl.value = '';
        }
        
        // Reload comments to show the new one
        if (section) {
            const list = section.querySelector('.todoist-comments-list');
            if (list) {
                await loadComments(section, taskId, this.apiBase);
            }
        }
        
        showNotification('Comment added');
    }
    
    insertFormat(format) {
        const formats = {
            bold: { prefix: '**', suffix: '**', placeholder: 'bold text' },
            italic: { prefix: '*', suffix: '*', placeholder: 'italic text' },
            strikethrough: { prefix: '~~', suffix: '~~', placeholder: 'strikethrough' },
            link: { prefix: '[', suffix: '](url)', placeholder: 'link text' },
            code: { prefix: '`', suffix: '`', placeholder: 'code' },
            codeblock: { prefix: '```\n', suffix: '\n```', placeholder: 'code block' },
            h1: { prefix: '# ', suffix: '', placeholder: 'Heading 1', newline: true },
            h2: { prefix: '## ', suffix: '', placeholder: 'Heading 2', newline: true },
            h3: { prefix: '### ', suffix: '', placeholder: 'Heading 3', newline: true },
            ul: { prefix: '- ', suffix: '', placeholder: 'list item', newline: true },
            ol: { prefix: '1. ', suffix: '', placeholder: 'list item', newline: true },
        };
        
        const fmt = formats[format];
        if (!fmt) return;
        
        const start = this.input.selectionStart;
        const end = this.input.selectionEnd;
        const text = this.input.value;
        const selectedText = text.substring(start, end) || fmt.placeholder;
        
        let prefix = fmt.prefix;
        // Add newline before if needed and not at start of line
        if (fmt.newline && start > 0 && text[start - 1] !== '\n') {
            prefix = '\n' + prefix;
        }
        
        const newText = text.substring(0, start) + prefix + selectedText + fmt.suffix + text.substring(end);
        this.input.value = newText;
        
        // Set cursor position
        const newStart = start + prefix.length;
        const newEnd = newStart + selectedText.length;
        this.input.setSelectionRange(newStart, newEnd);
        this.input.focus();
        this.updatePreview();
    }
    
    setupToolbar() {
        if (!this.modal) return;
        this.modal.querySelectorAll('.editor-toolbar button[data-format]').forEach(button => {
            button.addEventListener('click', (e) => {
                e.preventDefault();
                const format = e.currentTarget.dataset.format;
                this.insertFormat(format);
            });
        });
    }
    
    setupKeyboardShortcuts() {
        if (!this.input) return;
        this.input.addEventListener('keydown', (e) => {
            if (e.ctrlKey || e.metaKey) {
                switch(e.key.toLowerCase()) {
                    case 'b':
                        e.preventDefault();
                        this.insertFormat('bold');
                        break;
                    case 'i':
                        e.preventDefault();
                        this.insertFormat('italic');
                        break;
                    case 'k':
                        e.preventDefault();
                        this.insertFormat('link');
                        break;
                    case '`':
                        e.preventDefault();
                        this.insertFormat('code');
                        break;
                    case 'enter':
                        e.preventDefault();
                        this.save();
                        break;
                }
            }
            if (e.key === 'Escape') {
                e.preventDefault();
                this.close();
            }
        });
    }
    
    setupLivePreview() {
        if (!this.input) return;
        this.input.addEventListener('input', () => {
            this.updatePreview();
        });
    }
    
    updatePreview() {
        if (!this.preview) return;
        this.preview.innerHTML = parseMarkdown(this.input.value) || '<span style="color: var(--color-text-subdue)">Preview will appear here...</span>';
    }
    
    setupButtons() {
        if (!this.modal) return;
        
        // Close button
        this.modal.querySelector('.editor-close')?.addEventListener('click', () => this.close());
        
        // Cancel button
        this.modal.querySelector('.editor-cancel')?.addEventListener('click', () => this.close());
        
        // Save button
        this.modal.querySelector('.editor-save')?.addEventListener('click', () => this.save());
        
        // Overlay click
        this.modal.querySelector('.editor-overlay')?.addEventListener('click', () => this.close());
        
        // Preview toggle button
        const toggleBtn = this.modal.querySelector('.editor-preview-toggle');
        const editorBody = this.modal.querySelector('.editor-body');
        if (toggleBtn && editorBody) {
            toggleBtn.addEventListener('click', () => {
                editorBody.classList.toggle('preview-hidden');
                toggleBtn.classList.toggle('active');
            });
        }
    }
}

export default function(element) {
    const widgetId = element.dataset.widgetId;
    if (!widgetId) return;

    initTodoistWidget(element, widgetId);
}

function initTodoistWidget(element, widgetId) {
    const apiBase = `/api/widgets/${widgetId}`;
    
    // Initialize Rich Text Editor
    const editor = new RichTextEditor(element, apiBase);
    
    // Handle double-click on task titles to edit with rich editor
    element.querySelectorAll('.todoist-task .title').forEach(titleEl => {
        titleEl.style.cursor = 'pointer';
        titleEl.title = 'Double-click to edit';
        titleEl.addEventListener('dblclick', (e) => {
            const taskEl = e.currentTarget.closest('.todoist-task');
            const taskId = taskEl.dataset.taskId;
            const content = taskEl.dataset.content || e.currentTarget.textContent;
            editor.open(
                { type: 'task', id: taskId, element: taskEl },
                content,
                'Edit Task Title'
            );
        });
    });
    
    // Handle click on description toggle to open editor
    element.querySelectorAll('.todoist-task-description').forEach(descEl => {
        descEl.style.cursor = 'pointer';
        descEl.title = 'Double-click to edit';
        descEl.addEventListener('dblclick', (e) => {
            const taskEl = e.currentTarget.closest('.todoist-task');
            const taskId = taskEl.dataset.taskId;
            const content = taskEl.dataset.description || e.currentTarget.textContent;
            editor.open(
                { type: 'description', id: taskId, element: taskEl },
                content,
                'Edit Description'
            );
        });
    });
    
    // Handle double-click on comment input to open editor for new comment
    element.querySelectorAll('.todoist-comment-input').forEach(inputEl => {
        inputEl.title = 'Double-click for rich editor';
        inputEl.addEventListener('dblclick', (e) => {
            const section = e.currentTarget.closest('.todoist-comments-section');
            const taskEl = section?.closest('.todoist-task');
            const taskId = taskEl?.dataset.taskId;
            const content = e.currentTarget.value || '';
            if (taskId) {
                editor.open(
                    { type: 'new-comment', id: taskId, element: section, inputEl: e.currentTarget },
                    content,
                    'Add Comment'
                );
            }
        });
    });
    
    // Store editor reference for comments to use
    element._richTextEditor = editor;

    // Handle checkbox changes (task completion)
    element.querySelectorAll('.todoist-checkbox').forEach(checkbox => {
        checkbox.addEventListener('change', async (e) => {
            const taskId = e.target.dataset.taskId;
            const taskElement = e.target.closest('li');

            if (e.target.checked) {
                try {
                    await fetch(`${apiBase}/tasks/${taskId}/close`, { method: 'POST' });
                    taskElement.classList.add('todoist-task-completed');
                    const titleElement = taskElement.querySelector('.title');
                    if (titleElement) titleElement.classList.add('text-strikethrough');
                    showNotification('Task completed');
                } catch (error) {
                    console.error('Failed to close task:', error);
                    e.target.checked = false;
                    showNotification('Failed to complete task', 'error');
                }
            } else {
                try {
                    await fetch(`${apiBase}/tasks/${taskId}/reopen`, { method: 'POST' });
                    taskElement.classList.remove('todoist-task-completed');
                    const titleElement = taskElement.querySelector('.title');
                    if (titleElement) titleElement.classList.remove('text-strikethrough');
                    showNotification('Task reopened');
                } catch (error) {
                    console.error('Failed to reopen task:', error);
                    e.target.checked = true;
                    showNotification('Failed to reopen task', 'error');
                }
            }
        });
    });

    // Handle delete buttons
    element.querySelectorAll('.todoist-delete').forEach(button => {
        button.addEventListener('click', async (e) => {
            const taskId = e.currentTarget.dataset.taskId;
            const taskElement = e.currentTarget.closest('li');

            if (!confirm('Delete this task?')) return;

            try {
                await fetch(`${apiBase}/tasks/${taskId}`, { method: 'DELETE' });
                taskElement.style.opacity = '0';
                taskElement.style.transform = 'translateX(-20px)';
                setTimeout(() => {
                    taskElement.remove();
                    checkEmptyState(element);
                }, 200);
                showNotification('Task deleted');
            } catch (error) {
                console.error('Failed to delete task:', error);
                showNotification('Failed to delete task', 'error');
            }
        });
    });

    // Handle edit buttons
    element.querySelectorAll('.todoist-edit').forEach(button => {
        button.addEventListener('click', (e) => {
            const taskElement = e.currentTarget.closest('li');
            showEditModal(element, taskElement, apiBase);
        });
    });

    // Handle description toggle
    element.querySelectorAll('.todoist-desc-toggle').forEach(button => {
        button.addEventListener('click', (e) => {
            const taskId = e.currentTarget.dataset.taskId;
            const taskContent = e.currentTarget.closest('.todoist-task-content');
            const descElement = taskContent.querySelector('.todoist-task-description');
            if (descElement) {
                descElement.classList.toggle('collapsed');
                e.currentTarget.classList.toggle('todoist-expanded');
            }
        });
    });

    // Handle subtask button
    element.querySelectorAll('.todoist-subtask-btn').forEach(button => {
        button.addEventListener('click', (e) => {
            const taskId = e.currentTarget.dataset.taskId;
            const taskElement = e.currentTarget.closest('li');
            const form = taskElement.querySelector('.todoist-subtask-form');
            if (form) {
                form.classList.toggle('collapsed');
                if (!form.classList.contains('collapsed')) {
                    form.querySelector('.todoist-subtask-input').focus();
                }
            }
        });
    });

    // Handle subtask form submit
    element.querySelectorAll('.todoist-subtask-submit').forEach(button => {
        button.addEventListener('click', async (e) => {
            const form = e.currentTarget.closest('.todoist-subtask-form');
            const parentId = form.dataset.parentId;
            const input = form.querySelector('.todoist-subtask-input');
            const content = input.value.trim();

            if (!content) return;

            try {
                const response = await fetch(`${apiBase}/tasks`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ content, parent_id: parentId })
                });

                if (!response.ok) throw new Error('Failed to create subtask');

                input.value = '';
                form.classList.add('collapsed');
                showNotification('Subtask created');
                setTimeout(() => location.reload(), 500);
            } catch (error) {
                console.error('Failed to create subtask:', error);
                showNotification('Failed to create subtask', 'error');
            }
        });
    });

    // Handle subtask form cancel
    element.querySelectorAll('.todoist-subtask-cancel').forEach(button => {
        button.addEventListener('click', (e) => {
            const form = e.currentTarget.closest('.todoist-subtask-form');
            form.querySelector('.todoist-subtask-input').value = '';
            form.classList.add('collapsed');
        });
    });

    // Handle comments button
    element.querySelectorAll('.todoist-comments-btn').forEach(button => {
        button.addEventListener('click', async (e) => {
            const taskId = e.currentTarget.dataset.taskId;
            const taskElement = e.currentTarget.closest('.todoist-task');
            const section = taskElement.querySelector('.todoist-comments-section');

            if (!section) {
                console.error('Comments section not found for task', taskId);
                return;
            }
            if (section.classList.contains('collapsed')) {
                section.classList.remove('collapsed');
                await loadComments(section, taskId, apiBase);
            } else {
                section.classList.add('collapsed');
            }
        });
    });

    // Handle comment submit
    element.querySelectorAll('.todoist-comment-submit').forEach(button => {
        button.addEventListener('click', async (e) => {
            const section = e.currentTarget.closest('.todoist-comments-section');
            const taskId = section.dataset.taskId;
            const input = section.querySelector('.todoist-comment-input');
            const content = input.value.trim();

            if (!content) return;

            try {
                const response = await fetch(`${apiBase}/tasks/${taskId}/comments`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ content })
                });

                if (!response.ok) throw new Error('Failed to create comment');

                const comment = await response.json();
                input.value = '';
                appendComment(section.querySelector('.todoist-comments-list'), comment);
                updateCommentCount(element, taskId, 1);
                showNotification('Comment added');
            } catch (error) {
                console.error('Failed to create comment:', error);
                showNotification('Failed to add comment', 'error');
            }
        });
    });

    // Handle add task form
    const addForm = element.querySelector('.todoist-add-form');
    if (addForm) {
        addForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            const input = addForm.querySelector('.todoist-add-input');
            const content = input.value.trim();

            if (!content) return;

            const submitButton = addForm.querySelector('.todoist-add-button');
            submitButton.disabled = true;
            submitButton.textContent = '...';

            try {
                const response = await fetch(`${apiBase}/tasks`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ content })
                });

                if (!response.ok) throw new Error('Failed to create task');

                input.value = '';
                showNotification('Task created');
                setTimeout(() => location.reload(), 500);
            } catch (error) {
                console.error('Failed to create task:', error);
                showNotification('Failed to create task', 'error');
            } finally {
                submitButton.disabled = false;
                submitButton.textContent = '+';
            }
        });
    }

    // Initialize modal
    initEditModal(element, apiBase);
}

function showEditModal(widget, taskElement, apiBase) {
    const modal = widget.querySelector('.todoist-modal');
    if (!modal) return;

    const taskId = taskElement.dataset.taskId;
    const content = taskElement.dataset.content || '';
    const description = taskElement.dataset.description || '';
    const priority = taskElement.dataset.priority || '1';
    const due = taskElement.dataset.due || '';

    modal.querySelector('.todoist-edit-task-id').value = taskId;
    modal.querySelector('.todoist-edit-content').value = content;
    modal.querySelector('.todoist-edit-description').value = description;
    modal.querySelector('.todoist-edit-priority').value = priority;
    modal.querySelector('.todoist-edit-due').value = due;

    modal.classList.remove('collapsed');
    modal.querySelector('.todoist-edit-content').focus();
}

function hideEditModal(widget) {
    const modal = widget.querySelector('.todoist-modal');
    if (modal) {
        modal.classList.add('collapsed');
    }
}

function initEditModal(widget, apiBase) {
    const modal = widget.querySelector('.todoist-modal');
    if (!modal) return;

    // Close button
    modal.querySelector('.modal-close')?.addEventListener('click', () => hideEditModal(widget));
    modal.querySelector('.todoist-modal-cancel')?.addEventListener('click', () => hideEditModal(widget));

    // Overlay click
    modal.querySelector('.modal-overlay')?.addEventListener('click', () => hideEditModal(widget));

    // Form submit
    const form = modal.querySelector('.todoist-edit-form');
    if (form) {
        form.addEventListener('submit', async (e) => {
            e.preventDefault();

            const taskId = form.querySelector('.todoist-edit-task-id').value;
            const content = form.querySelector('.todoist-edit-content').value.trim();
            const description = form.querySelector('.todoist-edit-description').value.trim();
            const priority = parseInt(form.querySelector('.todoist-edit-priority').value);
            const dueString = form.querySelector('.todoist-edit-due').value.trim();

            if (!content) {
                showNotification('Task content is required', 'error');
                return;
            }

            const saveBtn = form.querySelector('.todoist-modal-save');
            saveBtn.disabled = true;
            saveBtn.textContent = 'Saving...';

            try {
                const body = { content, priority };
                if (description) body.description = description;
                if (dueString) body.due_string = dueString;

                const response = await fetch(`${apiBase}/tasks/${taskId}`, {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(body)
                });

                if (!response.ok) throw new Error('Failed to update task');

                hideEditModal(widget);
                showNotification('Task updated');
                setTimeout(() => location.reload(), 500);
            } catch (error) {
                console.error('Failed to update task:', error);
                showNotification('Failed to update task', 'error');
            } finally {
                saveBtn.disabled = false;
                saveBtn.textContent = 'Save Changes';
            }
        });
    }
}

async function loadComments(section, taskId, apiBase) {
    const list = section.querySelector('.todoist-comments-list');
    list.innerHTML = '<div class="todoist-loading">Loading comments...</div>';

    try {
        const response = await fetch(`${apiBase}/tasks/${taskId}/comments`);
        if (!response.ok) throw new Error('Failed to fetch comments');

        const comments = await response.json();
        list.innerHTML = '';

        if (!comments || comments.length === 0) {
            list.innerHTML = '<div class="todoist-no-comments">No comments yet</div>';
        } else {
            comments.forEach(comment => appendComment(list, comment, apiBase));
        }
    } catch (error) {
        console.error('Failed to load comments:', error);
        list.innerHTML = '<div class="todoist-error">Failed to load comments</div>';
    }
}

function appendComment(list, comment, apiBase) {
    const commentEl = document.createElement('div');
    commentEl.className = 'todoist-comment';
    commentEl.dataset.commentId = comment.id;
    commentEl.dataset.rawContent = comment.content; // Store raw content for editing

    const date = comment.posted_at ? new Date(comment.posted_at).toLocaleDateString() : '';

    // Parse markdown for comment content
    const renderedContent = parseMarkdown(comment.content);

    commentEl.innerHTML = `
        <div class="todoist-comment-content" title="Double-click to edit">${renderedContent}</div>
        <div class="todoist-comment-meta">${date}</div>
    `;

    // Add double-click to open rich text editor
    const contentEl = commentEl.querySelector('.todoist-comment-content');
    contentEl.addEventListener('dblclick', () => {
        // Find the widget element and get the editor
        const widget = list.closest('.todoist');
        const editor = widget?._richTextEditor;
        if (editor) {
            editor.open(
                { type: 'comment', id: comment.id, element: commentEl },
                comment.content,
                'Edit Comment'
            );
        }
    });

    list.appendChild(commentEl);
}

function enterCommentEditMode(commentEl, apiBase) {
    if (commentEl.classList.contains('editing')) return;
    
    commentEl.classList.add('editing');
    const contentEl = commentEl.querySelector('.todoist-comment-content');
    const rawContent = commentEl.dataset.rawContent;
    const commentId = commentEl.dataset.commentId;
    
    // Create edit textarea
    const textarea = document.createElement('textarea');
    textarea.className = 'todoist-comment-edit-input';
    textarea.value = rawContent;
    textarea.rows = 3;
    
    // Replace content with textarea
    contentEl.style.display = 'none';
    commentEl.insertBefore(textarea, contentEl);
    textarea.focus();
    textarea.setSelectionRange(rawContent.length, rawContent.length);
    
    // Add edit hint
    const hint = document.createElement('div');
    hint.className = 'todoist-edit-hint';
    hint.textContent = 'Enter to save, Escape to cancel';
    commentEl.insertBefore(hint, contentEl);
    
    const saveEdit = async () => {
        const newContent = textarea.value.trim();
        if (!newContent) {
            cancelEdit();
            return;
        }
        
        try {
            const response = await fetch(`${apiBase}/comments/${commentId}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ content: newContent })
            });
            
            if (!response.ok) throw new Error('Failed to update comment');
            
            // Update display
            commentEl.dataset.rawContent = newContent;
            contentEl.innerHTML = parseMarkdown(newContent);
            showNotification('Comment updated');
        } catch (error) {
            console.error('Failed to update comment:', error);
            showNotification('Failed to update comment', 'error');
        }
        
        cancelEdit();
    };
    
    const cancelEdit = () => {
        textarea.remove();
        hint.remove();
        contentEl.style.display = '';
        commentEl.classList.remove('editing');
    };
    
    textarea.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            saveEdit();
        } else if (e.key === 'Escape') {
            cancelEdit();
        }
    });
    
    textarea.addEventListener('blur', () => {
        // Small delay to allow Enter key to fire first
        setTimeout(() => {
            if (commentEl.classList.contains('editing')) {
                cancelEdit();
            }
        }, 100);
    });
}

function updateCommentCount(widget, taskId, delta) {
    const btn = widget.querySelector(`.todoist-comments-btn[data-task-id="${taskId}"]`);
    if (btn) {
        const countEl = btn.querySelector('.todoist-comment-count');
        if (countEl) {
            const current = parseInt(countEl.textContent) || 0;
            countEl.textContent = current + delta;
        }
    }
}

function checkEmptyState(widget) {
    const tasksContainer = widget.querySelector('.todoist-tasks');
    if (tasksContainer && tasksContainer.querySelectorAll('.todoist-task').length === 0) {
        const emptyDiv = document.createElement('div');
        emptyDiv.className = 'todoist-empty';
        emptyDiv.textContent = 'No tasks found';
        tasksContainer.appendChild(emptyDiv);
    }
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function showNotification(message, type = 'success') {
    const notification = document.createElement('div');
    notification.className = `todoist-notification todoist-notification-${type}`;
    notification.textContent = message;
    notification.style.cssText = `
        position: fixed;
        bottom: 20px;
        right: 20px;
        background: ${type === 'success' ? '#4caf50' : '#f44336'};
        color: white;
        padding: 12px 24px;
        border-radius: 4px;
        z-index: 10000;
        animation: slideIn 0.3s ease-out;
    `;

    document.body.appendChild(notification);

    setTimeout(() => {
        notification.style.opacity = '0';
        setTimeout(() => notification.remove(), 300);
    }, 3000);
}

// Add CSS animation for notifications
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from { transform: translateY(100%); opacity: 0; }
        to { transform: translateY(0); opacity: 1; }
    }
`;
document.head.appendChild(style);
