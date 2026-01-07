import { elem, fragment } from "./templating.js";

const trashIconSvg = `<svg fill="currentColor" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20">
  <path fill-rule="evenodd" d="M8.75 1A2.75 2.75 0 006 3.75v.443c-.795.077-1.584.176-2.365.298a.75.75 0 10.23 1.482l.149-.022.841 10.518A2.75 2.75 0 007.596 19h4.807a2.75 2.75 0 002.742-2.53l.841-10.52.149.023a.75.75 0 00.23-1.482A41.03 41.03 0 0014 4.193V3.75A2.75 2.75 0 0011.25 1h-2.5zM10 4c.84 0 1.673.025 2.5.075V3.75c0-.69-.56-1.25-1.25-1.25h-2.5c-.69 0-1.25.56-1.25 1.25v.325C8.327 4.025 9.16 4 10 4zM8.58 7.72a.75.75 0 00-1.5.06l.3 7.5a.75.75 0 101.5-.06l-.3-7.5zm4.34.06a.75.75 0 10-1.5-.06l-.3 7.5a.75.75 0 101.5.06l.3-7.5z" clip-rule="evenodd" />
</svg>`;

export default function(element) {
    const widgetId = element.dataset.widgetId;
    if (!widgetId) return;

    initTodoistWidget(element, widgetId);
}

function initTodoistWidget(element, widgetId) {
    const apiBase = `/api/widgets/${widgetId}`;

    // Handle checkbox changes (task completion)
    const checkboxes = element.querySelectorAll('.todoist-checkbox');
    checkboxes.forEach(checkbox => {
        checkbox.addEventListener('change', async (e) => {
            const taskId = e.target.dataset.taskId;
            const taskElement = e.target.closest('.todoist-task');

            if (e.target.checked) {
                // Close task
                try {
                    await fetch(`${apiBase}/tasks/${taskId}/close`, {
                        method: 'POST'
                    });
                    taskElement.classList.add('todoist-task-completed');
                    // Show success indication
                    showNotification('Task completed');
                } catch (error) {
                    console.error('Failed to close task:', error);
                    e.target.checked = false; // Revert
                    showNotification('Failed to complete task', 'error');
                }
            } else {
                // Reopen task
                try {
                    await fetch(`${apiBase}/tasks/${taskId}/reopen`, {
                        method: 'POST'
                    });
                    taskElement.classList.remove('todoist-task-completed');
                    showNotification('Task reopened');
                } catch (error) {
                    console.error('Failed to reopen task:', error);
                    e.target.checked = true; // Revert
                    showNotification('Failed to reopen task', 'error');
                }
            }
        });
    });

    // Handle delete buttons
    const deleteButtons = element.querySelectorAll('.todoist-delete');
    deleteButtons.forEach(button => {
        button.addEventListener('click', async (e) => {
            const taskId = e.currentTarget.dataset.taskId;
            const taskElement = e.currentTarget.closest('.todoist-task');

            if (!confirm('Delete this task?')) return;

            try {
                await fetch(`${apiBase}/tasks/${taskId}`, {
                    method: 'DELETE'
                });
                // Animate removal
                taskElement.style.opacity = '0';
                taskElement.style.transform = 'translateX(-20px)';
                setTimeout(() => {
                    taskElement.remove();
                    // Check if empty
                    const tasksContainer = element.querySelector('.todoist-tasks');
                    if (tasksContainer && tasksContainer.children.length === 0) {
                        const emptyDiv = document.createElement('div');
                        emptyDiv.className = 'todoist-empty';
                        emptyDiv.textContent = 'No tasks found';
                        tasksContainer.appendChild(emptyDiv);
                    }
                }, 200);
                showNotification('Task deleted');
            } catch (error) {
                console.error('Failed to delete task:', error);
                showNotification('Failed to delete task', 'error');
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
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        content: content
                    })
                });

                if (!response.ok) {
                    throw new Error('Failed to create task');
                }

                const task = await response.json();

                // Add task to DOM
                const tasksContainer = element.querySelector('.todoist-tasks');
                const emptyDiv = element.querySelector('.todoist-empty');

                if (emptyDiv) {
                    emptyDiv.remove();
                }

                const taskElement = createTaskElement(task);
                if (tasksContainer) {
                    tasksContainer.insertBefore(taskElement, tasksContainer.firstChild);
                }

                input.value = '';
                showNotification('Task created');

                // Refresh widget to get updated data
                setTimeout(() => {
                    location.reload();
                }, 1000);

            } catch (error) {
                console.error('Failed to create task:', error);
                showNotification('Failed to create task', 'error');
            } finally {
                submitButton.disabled = false;
                submitButton.textContent = '+';
            }
        });
    }
}

function createTaskElement(task) {
    const taskDiv = document.createElement('div');
    taskDiv.className = 'todoist-task';
    taskDiv.dataset.taskId = task.id;
    taskDiv.dataset.content = task.content;

    taskDiv.innerHTML = `
        <div class="todoist-task-header">
            <input type="checkbox" class="todoist-checkbox" data-task-id="${task.id}" aria-label="Mark task as complete">
            <div class="todoist-task-content">
                <span class="todoist-task-title">${escapeHtml(task.content)}</span>
            </div>
            <div class="todoist-task-meta">
                ${task.due ? `<span class="todoist-due${task.due.is_recurring ? ' todoist-recurring' : ''}">${escapeHtml(task.due.string || task.due.date)}</span>` : ''}
                ${task.priority > 1 ? `<span class="todoist-priority todoist-priority-${task.priority}" title="Priority ${task.priority}"></span>` : ''}
            </div>
            <button class="todoist-delete" data-task-id="${task.id}" aria-label="Delete task">
                ${trashIconSvg}
            </button>
        </div>
    `;

    // Re-attach event listeners
    const checkbox = taskDiv.querySelector('.todoist-checkbox');
    checkbox.addEventListener('change', async (e) => {
        const taskId = e.target.dataset.taskId;
        const widgetId = taskDiv.closest('.todoist').dataset.widgetId;
        const apiBase = `/api/widgets/${widgetId}`;

        if (e.target.checked) {
            try {
                await fetch(`${apiBase}/tasks/${taskId}/close`, { method: 'POST' });
                taskDiv.classList.add('todoist-task-completed');
            } catch (error) {
                console.error('Failed to close task:', error);
                e.target.checked = false;
            }
        }
    });

    const deleteButton = taskDiv.querySelector('.todoist-delete');
    deleteButton.addEventListener('click', async (e) => {
        const taskId = e.currentTarget.dataset.taskId;
        const widgetId = taskDiv.closest('.todoist').dataset.widgetId;
        const apiBase = `/api/widgets/${widgetId}`;

        if (!confirm('Delete this task?')) return;

        try {
            await fetch(`${apiBase}/tasks/${taskId}`, { method: 'DELETE' });
            taskDiv.style.opacity = '0';
            setTimeout(() => taskDiv.remove(), 200);
        } catch (error) {
            console.error('Failed to delete task:', error);
        }
    });

    return taskDiv;
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function showNotification(message, type = 'success') {
    // Simple notification system
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
