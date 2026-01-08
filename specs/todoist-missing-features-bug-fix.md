# Bug: Todoist Widget Missing Task Edit, Subtask Creation, Description Display, and Comments Features

## Bug Description

The Todoist widget is functional for basic task viewing and creation, but lacks several critical user interaction features:

1. **No task editing capability** - Users cannot edit existing task content, due dates, priority, or other properties
2. **No subtask creation UI** - While the widget shows tasks can have subtasks, there's no UI to create subtasks for a parent task
3. **Description visibility issue** - Task descriptions are only shown if `hide-description: false` is set, but there's no way to toggle visibility inline
4. **No comments viewing/adding** - The widget shows comment count badges but provides no UI to view existing comments or add new ones

**Expected Behavior:**
- Users should be able to click a task to edit it
- Users should be able to add subtasks to existing tasks
- Task descriptions should be expandable/collapsible inline
- Comments should be viewable and users should be able to add new comments

**Actual Behavior:**
- Tasks are read-only (except for completion/deletion)
- No UI for creating subtasks
- Descriptions are either always shown or always hidden based on config
- Comments are displayed as a count badge only, with no way to view or add them

## Problem Statement

The Todoist widget implementation focuses on task viewing and basic CRUD (create, complete, delete) but lacks the interactive UI components for:

1. **Inline task editing** - No edit button or modal to modify task properties
2. **Subtask creation** - No UI to add subtasks to a parent task (parent_id field exists in API but not exposed)
3. **Description toggle** - Descriptions are statically shown/hidden based on config, not user-controlled per task
4. **Comments interface** - Comment count is displayed but there's no expandable comments section or form to add new comments

The backend API client supports all these operations (UpdateTask, CreateTask with parent_id, GetComments, CreateComment), but the frontend UI doesn't expose these capabilities.

## Solution Statement

Add the missing UI components and JavaScript handlers to expose the existing backend API capabilities:

1. **Add edit button and modal** - Allow inline editing of task content, description, due date, priority
2. **Add subtask creation UI** - Add a button to create subtasks for each parent task
3. **Add description toggle** - Make descriptions collapsible/expandable with a "show more" indicator
4. **Add comments section** - Create expandable comments panel with ability to view and add comments

The solution will:
- Reuse existing backend API endpoints (no backend changes needed)
- Add new UI elements to the template
- Add JavaScript handlers for new interactions
- Add CSS styling for new components
- Maintain existing functionality without breaking changes

## Steps to Reproduce

1. Start the Glance server with Todoist widget configured
2. Navigate to the Todoist page
3. Observe a task with a description
4. Try to edit the task - **no edit button or interaction exists**
5. Try to add a subtask to a task - **no UI for this exists**
6. Try to toggle description visibility - **not possible without config change**
7. Click on a task with comments - **comments are not viewable**

## Root Cause Analysis

The Todoist widget was implemented with a focus on basic task management (view, create, complete, delete) but the more advanced interactive features were not included in the initial implementation. The backend API client (`internal/glance/todoist/client.go`) already supports all necessary operations:

- `UpdateTask()` - exists but not exposed via UI
- `CreateTask()` with `parent_id` - supported but no UI to set parent
- `GetComments()` / `CreateComment()` - implemented but no frontend interface

The root cause is that the frontend components (template and JavaScript) only implement the basic use cases, missing:
- Edit button/trigger in the task template
- Edit modal/form in the template
- Subtask creation button
- Description expand/collapse UI
- Comments section with toggle
- JavaScript handlers for these interactions

## Relevant Files

### Files to Modify

- **`internal/glance/templates/todoist.html`** (86 lines)
  - Add edit button to each task
  - Add subtask creation button
  - Add description toggle button
  - Add comments section with toggle
  - Add edit modal dialog
  - Add subtask creation form

- **`internal/glance/static/js/todoist.js`** (253 lines)
  - Add edit button click handler
  - Add task update functionality (PUT to existing endpoint)
  - Add subtask creation handler (use existing create endpoint with parent_id)
  - Add description toggle handler
  - Add comments section toggle handler
  - Add comment viewing (fetch from API)
  - Add comment creation handler
  - Add modal show/hide logic

- **`internal/glance/widget-todoist.go`** (420 lines)
  - Add PUT endpoint handler for task updates (path detection for `/tasks/{id}` without `/close` or `/reopen`)
  - Add GET endpoint handler for comments (`/tasks/{id}/comments`)
  - Add POST endpoint handler for comments (`/tasks/{id}/comments`)
  - Note: Create task with parent_id already works, just needs frontend to send it

- **`internal/glance/static/css/widget-todoist.css`** (265 lines)
  - Add styles for edit button
  - Add styles for subtask button
  - Add styles for description toggle
  - Add styles for comments section
  - Add styles for edit modal
  - Add styles for collapsible sections

### New Files

None required - all changes will be to existing files.

## Step by Step Tasks

### 1. Backend API Endpoint Enhancements

- [ ] **Update `widget-todoist.go` handleRequest method** to handle task updates
  - Add case in `http.MethodPut` for updating task content (detect when path doesn't end in `/close` or `/reopen`)
  - Parse request body as `UpdateTaskRequest`
  - Call `widget.client.UpdateTask()`
  - Schedule early update and return 204 No Content

- [ ] **Add GET endpoint for comments** in `widget-todoist.go`
  - Handle GET requests to `/tasks/{id}/comments`
  - Call `widget.client.GetComments()` with task ID
  - Return JSON response with comments array

- [ ] **Add POST endpoint for comments** in `widget-todoist.go`
  - Handle POST requests to `/tasks/{id}/comments`
  - Parse request body with content and optional attachment
  - Call `widget.client.CreateComment()`
  - Return 201 Created with the new comment

### 2. Template UI Enhancements

- [ ] **Add action buttons to task template** in `todoist.html`
  - Add edit button next to delete button (hidden by default, shown on hover)
  - Add subtask creation button (show for all tasks)
  - Add description toggle button (show when description exists and is hidden)

- [ ] **Add collapsible sections to task template** in `todoist.html`
  - Add expandable description section (hidden by default if HideDescription is true)
  - Add expandable comments section (hidden by default)
  - Add comment count indicator that opens comments section

- [ ] **Add edit modal** to `todoist.html`
  - Create modal dialog at widget level (not per task)
  - Include fields: content (textarea), description (textarea), due date (input), priority (select)
  - Include save and cancel buttons
  - Include hidden field for task ID

- [ ] **Add subtask creation form** to `todoist.html`
  - Add inline form below each task (hidden by default)
  - Include input for subtask content
  - Include submit button
  - Include hidden field for parent task ID

- [ ] **Add comments section** to `todoist.html`
  - Add container for comments list (empty initially)
  - Add form to add new comment
  - Include textarea for comment content
  - Include submit button

### 3. JavaScript Functionality

- [ ] **Add edit functionality** to `todoist.js`
  - Add click handler for edit buttons
  - Populate modal with current task data
  - Show modal
  - Add save handler that sends PUT request to `/tasks/{id}`
  - Add cancel handler that hides modal
  - Update task in DOM on successful save

- [ ] **Add subtask creation** to `todoist.js`
  - Add toggle handler for subtask form visibility
  - Add submit handler for subtask form
  - Send POST request to `/tasks` with `parent_id` set
  - Add new subtask to DOM or reload widget

- [ ] **Add description toggle** to `todoist.js`
  - Add click handler for description toggle button
  - Toggle visibility of description element
  - Update button text/icon based on state

- [ ] **Add comments functionality** to `todoist.js`
  - Add click handler for comment count badge to open comments section
  - Fetch comments via GET request to `/tasks/{id}/comments`
  - Render comments in the comments section
  - Add submit handler for new comment form
  - Send POST request to `/tasks/{id}/comments`
  - Append new comment to comments list
  - Update comment count badge

- [ ] **Add modal utility functions** to `todoist.js`
  - `showEditModal(task)` - populate and show modal
  - `hideEditModal()` - hide modal
  - `saveTask(taskId, data)` - send update request

- [ ] **Update `createTaskElement` function** in `todoist.js`
  - Include new buttons (edit, subtask, description toggle)
  - Include collapsible sections (description, comments)
  - Attach event listeners to new elements

### 4. CSS Styling

- [ ] **Add action button styles** to `widget-todoist.css`
  - Edit button styling (similar to delete but different color)
  - Subtask button styling (distinct from delete)
  - Description toggle button styling
  - Hover effects for all action buttons

- [ ] **Add modal styles** to `widget-todoist.css`
  - Modal overlay (fixed, semi-transparent background)
  - Modal dialog (centered, max-width, padding)
  - Modal form fields (full width, proper spacing)
  - Modal buttons (save, cancel)

- [ ] **Add collapsible section styles** to `widget-todoist.css`
  - Description section (margin, padding, border)
  - Comments section (margin, padding, border)
  - Comment item styling (avatar, content, timestamp)
  - Transition animations for expand/collapse

- [ ] **Add subtask form styles** to `widget-todoist.css`
  - Inline form (flex layout, gap)
  - Input field styling
  - Button styling

- [ ] **Add comments form styles** to `widget-todoist.css`
  - Textarea styling
  - Submit button styling
  - Comment list styling

### 5. Testing and Validation

- [ ] **Test task editing**
  - Click edit button on a task
  - Modify task content
  - Save changes
  - Verify task is updated in UI and in Todoist

- [ ] **Test subtask creation**
  - Click subtask button on a task
  - Enter subtask content
  - Submit form
  - Verify subtask is created and appears

- [ ] **Test description toggle**
  - Click description toggle on a task with hidden description
  - Verify description becomes visible
  - Click toggle again
  - Verify description is hidden

- [ ] **Test comments viewing**
  - Click comment count badge on a task
  - Verify comments section expands
  - Verify existing comments are displayed

- [ ] **Test comment creation**
  - Open comments section for a task
  - Enter comment content
  - Submit form
  - Verify comment appears in list
  - Verify comment count badge updates

- [ ] **Test mobile responsiveness**
  - Verify modal works on mobile
  - Verify action buttons are tappable
  - Verify collapsible sections work on touch

### 6. Documentation

- [ ] **Update configuration documentation** if needed
  - Document any new configuration options (if any)
  - Update usage examples

- [ ] **Add comments to code** for complex interactions
  - Document the edit flow
  - Document the comments API integration
  - Document the subtask creation flow

## Validation Commands

Execute every command to validate the bug is fixed with zero regressions.

```bash
# Build the application
go build -o build/glance .

# Validate configuration
./build/glance config:validate --config config/glance.yml

# Run tests (if any exist)
go test ./...

# Start the server
./build/glance --config config/glance.yml

# In browser, test the following:
# 1. Navigate to Todoist page
# 2. Click edit button on a task, modify content, save
# 3. Click subtask button, create subtask
# 4. Click description toggle, verify visibility
# 5. Click comment count, view comments, add new comment
# 6. Verify all existing functionality still works (complete, delete, create)
```

## Notes

### Implementation Considerations

1. **Backward Compatibility** - All changes are additive. Existing configurations will continue to work without modification.

2. **API Efficiency** - Comments are fetched on-demand when the user clicks to view them, not during initial page load. This keeps the widget fast.

3. **Modal Implementation** - A single modal at the widget level is reused for editing any task, rather than creating a modal per task.

4. **Progressive Enhancement** - If JavaScript fails to load, the widget still displays tasks. The interactive features gracefully degrade.

5. **Security** - All API calls go through the backend widget handler, which validates requests and applies the widget's authentication configuration.

6. **Performance** - Task updates use optimistic UI updates for better perceived performance, with server validation happening in the background.

7. **Mobile Considerations** - All new UI elements must be touch-friendly and work well on mobile devices.

### Future Enhancements (Out of Scope)

- Task drag-and-drop reordering
- Bulk task operations
- Advanced filtering UI
- Task attachments
- Project/section management UI
- Label management UI

### API Endpoints Used

All endpoints already exist in the backend:

- `PUT /api/widgets/{id}/tasks/{taskId}` - Update task (to be added to handler)
- `GET /api/widgets/{id}/tasks/{taskId}/comments` - Get comments (to be added to handler)
- `POST /api/widgets/{id}/tasks/{taskId}/comments` - Create comment (to be added to handler)
- `POST /api/widgets/{id}/tasks` - Create task (with parent_id for subtasks, already exists)
