# Feature: Todoist Integration for Glance Dashboard

## Feature Description

This feature adds a new widget type to Glance that integrates with Todoist, providing a comprehensive task management interface directly within the dashboard. The widget will enable users to view, create, and manage tasks, subtasks, projects, labels, and comments through Todoist's REST API v2, with all data synchronized back to the user's Todoist account.

The widget will support comprehensive Todoist API integration:

**Tasks Management:**
- View tasks with advanced filters (inbox, today, priority, project, section, label, custom filter)
- Create tasks with: content, description, project, section, parent, order, labels, priority, due date (string/date/datetime), assignee, duration, deadline
- Update tasks: modify all task properties
- Complete/close tasks (marks recurring tasks for next occurrence)
- Reopen completed tasks
- Delete tasks
- Add subtasks to existing tasks
- View task details: comments, attachments, assignments, collaborators

**Projects Management:**
- View all projects with properties (id, name, color, parent_id, order, is_shared, is_favorite, is_inbox_project, is_team_inbox, view_style)
- Create new projects (with name, parent_id, color, is_favorite, view_style)
- Update project properties
- Archive/unarchive projects
- Delete projects (with all sections and tasks)
- View project collaborators (for shared projects)

**Sections Management:**
- View all sections (with optional project filter)
- Create new sections (with name, project_id, order)
- Update section properties (name)
- Delete sections (with all tasks)

**Comments System:**
- View all comments for tasks or projects
- Create new comments (with optional attachment support)
- Update comment content
- Delete comments
- Display comment metadata (attachment, posted_at, author)

**Labels Management:**
- View all personal labels (with properties: id, name, color, order, is_favorite)
- Create personal labels (with name, order, color, is_favorite)
- Update label properties
- Delete personal labels
- View all shared labels (with omit_personal option)
- Rename shared labels
- Remove shared labels from all tasks

**Advanced Features:**
- Task duration tracking (amount and unit: minute/day)
- Task deadlines (separate from due dates)
- Task assignments (assignee_id for shared projects)
- Natural language due date parsing (due_string with language support)
- Recurring task detection and display
- Markdown rendering for task content and descriptions
- Real-time synchronization with Todoist
- Intelligent caching to respect API rate limits (1000 requests/15min)

## User Story

As a **productivity-focused dashboard user**
I want to **view and manage my Todoist tasks directly from Glance**
So that **I can maintain a centralized dashboard for all my feeds and tasks without switching applications**

## Problem Statement

Currently, Glance has a simple todo widget that stores tasks in browser localStorage, which is not suitable for users who:
1. Use Todoist as their primary task management system
2. Need to access and manage tasks across multiple devices
3. Want to leverage Todoist's advanced features (projects, labels, reminders, collaboration)
4. Require their task data to persist and sync reliably

The existing widget lacks:
- Cloud synchronization
- Advanced task features (subtasks, due dates, priorities, labels)
- Project/section organization
- Comment/attachment support
- Integration with external task management services

## Solution Statement

Create a new `todoist` widget type that integrates with Todoist's REST API v2 using Python's `todoist-api-python` SDK. The widget will:

1. **Backend Integration**: Use Python subprocess or embedded Python to interact with Todoist API
2. **Authentication**: Support Todoist API tokens via environment variables (following Glance's security patterns)
3. **Widget Configuration**: Provide flexible configuration options for filters, projects, and display modes
4. **Frontend Interaction**: Extend the existing todo widget UI with Todoist-specific features
5. **Caching**: Implement intelligent caching to respect API rate limits (1000 requests/15min per user)

The solution follows Glance's architecture patterns:
- Server-side rendering with Go templates
- Minimal JavaScript for interactivity
- Environment variable support for sensitive data
- YAML-based configuration
- No external dependencies beyond what's necessary

## Relevant Files

### Existing Files to Reference

- **`internal/glance/widget-todo.go`**
  - Current todo widget implementation with localStorage
  - Widget structure and interface implementation
  - Template rendering pattern
  - Use this as a template for the new Todoist widget

- **`internal/glance/widget.go`**
  - Widget interface definition
  - All widgets must implement `update()`, `Render()` methods
  - Widget base configuration structure

- **`internal/glance/widget-rss.go`**
  - Good example of a widget that fetches external data
  - HTTP request patterns and error handling
  - Configuration parsing with multiple options

- **`internal/glance/widget-custom-api.go`**
  - Shows how to fetch and render JSON data from external APIs
  - Template-based rendering with dynamic data
  - Configuration options for API endpoints

- **`internal/glance/config.go`**
  - Configuration parsing system with environment variable support
  - Widget registration and initialization
  - Cache management system

- **`internal/glance/widget-utils.go`**
  - Shared utilities for HTTP requests
  - User agent handling
  - Error handling patterns

- **`internal/glance/templates/todo.html`**
  - Current todo widget template
  - Minimal structure that can be extended

- **`internal/glance/static/js/todo.js`**
  - Client-side todo list management
  - Drag-and-drop reordering
  - localStorage persistence
  - Will need to be modified for Todoist sync

- **`internal/glance/static/css/widget-todo.css`**
  - Todo widget styling
  - May need extensions for Todoist-specific elements (priority indicators, labels, etc.)

- **`config/glance.yml`**
  - Example configuration that will include Todoist widget examples

- **`docs/configuration.md`**
  - Documentation that needs a new Todoist section
  - Follow existing widget documentation pattern

### New Files to Create

- **`internal/glance/widget-todoist.go`**
  - Main Todoist widget implementation
  - Todoist API client initialization
  - Task/project/label fetching logic
  - Configuration struct with all Todoist options

- **`internal/glance/templates/todoist.html`**
  - Todoist widget template
  - Project/section views
  - Task items with metadata display

- **`internal/glance/static/js/todoist.js`**
  - Todoist-specific client-side functionality
  - Task creation/editing forms
  - Subtask expansion/collapse
  - Comment display
  - AJAX calls for CRUD operations

- **`internal/glance/static/css/widget-todoist.css`**
  - Todoist-specific styling
  - Priority indicators
  - Label badges
  - Project/section styling

- **`internal/glance/todoist_api.py`` or similar**
  - Python bridge for Todoist API calls
  - Wrapper functions for common operations
  - Error handling and response formatting
  - Alternatively, could use direct HTTP requests from Go without Python

- **`.env` and `.example.env`**
  - Environment variable configuration for Todoist API token
  - Add `TODOIST_API_TOKEN` variable

## Implementation Plan

### Phase 1: Foundation

**Objective**: Set up the infrastructure for Todoist integration

1. **Research Decision Point**: Python Bridge vs Direct Go HTTP Client
   - Evaluate whether to use Python SDK or make direct HTTP requests from Go
   - Consider: Glance's "no new dependencies" principle
   - Direct HTTP from Go would be cleaner and avoid Python runtime dependency
   - Decision: Use Go's `net/http` to call Todoist REST API directly (no Python needed)

2. **Configuration Schema**
   - Define `todoistWidget` struct in `config-fields.go`
   - Support environment variable for API token
   - Configuration options:
     - **Authentication**:
       - `api-token`: Todoist API token (from `${TODOIST_API_TOKEN}` or `${secret:todoist_token}`)
     - **Task Filtering**:
       - `filter`: Custom Todoist filter string (e.g., "today & priority high")
       - `project-id`: Filter by specific project ID
       - `section-id`: Filter by specific section ID
       - `label`: Filter by label name
       - `priority-min`: Filter by minimum priority (1-4)
       - `due-filter`: Due date filter (today, overdue, upcoming, none, all)
       - `lang`: Language code for filter parsing (default: en)
       - `ids`: Comma-separated list of specific task IDs to display
       - `assignee`: Filter by assignee ID
     - **Display Options**:
       - `show-completed`: Include completed tasks (default: false)
       - `show-subtasks`: Display subtasks (default: true)
       - `show-projects`: Show project/section browser (default: true)
       - `show-labels`: Show label list (default: true)
       - `show-collaborators`: Show project collaborators (default: true)
       - `limit`: Maximum number of tasks to show (default: 20)
       - `collapse-after`: Collapse subtasks after N items (default: 3)
       - `group-by`: Group tasks by (none, project, priority, due_date)
       - `sort-by`: Sort tasks by (default, priority, due_date, created_at, manual)
       - `sort-order`: asc or desc (default: asc)
     - **Task Creation**:
       - `default-project-id`: Default project for new tasks
       - `default-section-id`: Default section for new tasks
       - `default-priority`: Default priority for new tasks (1-4)
       - `default-labels`: Default labels for new tasks (comma-separated)
       - `allow-natural-language`: Enable natural language parsing (default: true)
       - `default-due-time`: Enable due time by default (default: false)
     - **Advanced Options**:
       - `cache`: Cache duration (default: 5m)
       - `refresh-interval`: Auto-refresh interval (default: 0 = disabled)
       - `show-project-colors`: Show project color indicators (default: true)
       - `show-label-colors`: Show label color badges (default: true)
       - `markdown-rendering`: Enable markdown in task content (default: true)
       - `compact-mode`: Use compact list layout (default: false)
       - `hide-description`: Hide task descriptions by default (default: false)
       - `show-avatars`: Show assignee avatars (default: true)
       - `show-duration`: Show task duration badges (default: true)
       - `show-deadline`: Show deadline badges (default: true)
     - **UI Controls**:
       - `show-create-button`: Show task creation button (default: true)
       - `show-project-selector`: Show project selector in create form (default: true)
       - `show-label-selector`: Show label selector in create form (default: true)
       - `enable-drag-drop`: Enable drag-and-drop reordering (default: true)
       - `enable-inline-edit`: Enable inline task editing (default: true)
       - `show-bulk-actions`: Show bulk action buttons (default: true)

3. **Authentication Setup**
   - Document API token acquisition from Todoist integrations settings
   - Add to `.env.example`: `TODOIST_API_TOKEN=your_token_here`
   - Support `${TODOIST_API_TOKEN}` and `${secret:todoist_token}` patterns

4. **API Client Infrastructure**
   - Create `internal/glance/todoist/client.go`
   - Implement HTTP client with authorization header
   - Add request/response structures matching Todoist API v2
   - Implement rate limiting awareness (1000 requests/15min)
   - Error handling for unauthorized, rate limit, and network errors

### Phase 2: Core Widget Implementation

**Objective**: Build the basic Todoist widget with task display

1. **Widget Structure** (`widget-todoist.go`)
   - Create `todoistWidget` struct with configuration
   - Implement `widget` interface: `update()`, `Render()`, `initialize()`
   - Fetch tasks on update with caching
   - Support different filter modes (inbox, today, project, label)

2. **API Integration** (`todoist/client.go`)
   - **Tasks API Methods**:
     - `GetTasks(opts)` - Fetch active tasks with filters (project_id, section_id, label, filter, lang, ids)
     - `GetTask(taskId)` - Get single active task with all details
     - `CreateTask(request)` - Create task with: content, description, project_id, section_id, parent_id, order, labels, priority, due_string/due_date/due_datetime, due_lang, assignee_id, duration, duration_unit, deadline_date
     - `UpdateTask(taskId, request)` - Update task: content, description, labels, priority, due_string/due_date/due_datetime, due_lang, assignee_id, duration, duration_unit, deadline_date
     - `CloseTask(taskId)` - Mark task complete (handles recurring)
     - `ReopenTask(taskId)` - Reopen completed task
     - `DeleteTask(taskId)` - Delete task permanently

   - **Projects API Methods**:
     - `GetProjects()` - Fetch all user projects
     - `GetProject(projectId)` - Get single project by ID
     - `CreateProject(request)` - Create project with: name, parent_id, color, is_favorite, view_style
     - `UpdateProject(projectId, request)` - Update project: name, color, is_favorite, view_style
     - `ArchiveProject(projectId)` - Archive project
     - `UnarchiveProject(projectId)` - Unarchive project
     - `DeleteProject(projectId)` - Delete project with all sections and tasks
     - `GetProjectCollaborators(projectId)` - Get collaborators for shared projects

   - **Sections API Methods**:
     - `GetSections(projectId)` - Fetch all sections (optionally filtered by project_id)
     - `GetSection(sectionId)` - Get single section by ID
     - `CreateSection(request)` - Create section with: name, project_id, order
     - `UpdateSection(sectionId, request)` - Update section name
     - `DeleteSection(sectionId)` - Delete section with all tasks

   - **Comments API Methods**:
     - `GetComments(opts)` - Get comments for task_id or project_id
     - `GetComment(commentId)` - Get single comment by ID
     - `CreateComment(request)` - Create comment with: content, task_id/project_id, attachment
     - `UpdateComment(commentId, content)` - Update comment content
     - `DeleteComment(commentId)` - Delete comment

   - **Labels API Methods**:
     - `GetLabels()` - Get all personal labels
     - `GetLabel(labelId)` - Get single personal label
     - `CreateLabel(request)` - Create personal label with: name, order, color, is_favorite
     - `UpdateLabel(labelId, request)` - Update label: name, order, color, is_favorite
     - `DeleteLabel(labelId)` - Delete personal label (removes from all tasks)
     - `GetSharedLabels(omitPersonal)` - Get all shared label names
     - `RenameSharedLabel(name, newName)` - Rename all instances of shared label
     - `RemoveSharedLabel(name)` - Remove shared label from all tasks

3. **Template** (`templates/todoist.html`)
   - Base structure using `widget-base.html`
   - Task list display with:
     - Checkbox for completion
     - Task content with markdown support
     - Due date display (if set)
     - Priority indicator
     - Label badges
     - Project/section indicator
   - Subtask nesting (collapsible)
   - "Add task" input field

4. **Styling** (`css/widget-todoist.css`)
   - Extend existing todo widget styles
   - Add priority color coding (1=normal, 2=medium, 3=high, 4=urgent)
   - Label badge styling (small, colored tags)
   - Subtask indentation
   - Collapsed state indicators
   - Error state styling

5. **Client-Side JavaScript** (`js/todoist.js`)
   - Modify existing `todo.js` for Todoist data structure
   - Add task completion (calls close task endpoint)
   - Add task deletion
   - Implement collapsible subtasks
   - Real-time task creation via AJAX

### Phase 3: Advanced Features

**Objective**: Add advanced Todoist features and interactions

1. **Comprehensive Task Creation**
   - Frontend form with all Todoist task fields:
     - Content (required, markdown supported)
     - Description (optional, markdown)
     - Project selector (dropdown with color indicators)
     - Section selector (filtered by selected project)
     - Parent task selector (for subtasks)
     - Labels selector (multi-select with color badges)
     - Priority selector (1-4 with visual indicators)
     - Due date options:
       - Natural language input (due_string with language selector)
       - Date picker (due_date)
       - Date/time picker (due_datetime)
     - Deadline date picker (deadline_date)
     - Duration fields (amount + unit selector: minute/day)
     - Assignee selector (for shared projects)
   - Backend endpoint to handle all task creation parameters
   - JavaScript to submit form and refresh widget
   - Validation for required fields

2. **Subtask Management**
   - Display subtasks nested under parent task
   - Add subtask button on each parent task
   - Collapsible subtask sections
   - Visual distinction for subtask hierarchy (indentation, connecting lines)
   - Subtask count badge on parent tasks
   - Recursive subtask display (subtask of subtask)

3. **Comments System with Attachments**
   - Display comment count badge on each task
   - Expandable comments section
   - Comment list with:
     - Author name/avatar
     - Posted date (relative time)
     - Content (markdown rendered)
     - Attachment display (file type icon, filename, download link)
   - Add comment form:
     - Text area for content
     - Optional file attachment input
     - Submit button
   - Update comment functionality
   - Delete comment functionality

4. **Project/Section Management UI**
   - Project browser view:
     - List all projects with hierarchy
     - Project color indicators
     - Favorite star indicator
     - Shared project indicator
     - Task count per project
   - Section browser within projects
   - Create new project form (name, color, favorite toggle)
   - Create new section form (name, project selector)
   - Archive/unarchive project functionality
   - Delete project/section with confirmation
   - Collaborators list for shared projects

5. **Labels Management**
   - Label browser view:
     - All personal labels with colors
     - Favorite indicator
     - Task count per label
   - Shared labels section
   - Create new label form (name, color, favorite toggle)
   - Edit label functionality
   - Delete label functionality
   - Filter tasks by label
   - Multi-label filtering

6. **Advanced Task Display**
   - Task properties display:
     - Due date with recurring indicator
     - Deadline date (separate badge)
     - Duration badge (e.g., "15 min", "2 days")
     - Assignee avatar (for shared tasks)
     - Project/section breadcrumb
   - Priority indicators (color-coded borders or icons)
   - Label badges with colors
   - Subtask count and progress
   - Comment count
   - Attachment count

7. **Task Editing and Updates**
   - Inline task editing:
     - Double-click to edit content
     - Edit description in expanded view
     - Quick priority change
     - Quick due date change
   - Full edit modal with all fields
   - Update task via AJAX without page reload
   - Optimistic UI updates

8. **Filter and Search Capabilities**
   - Advanced filter panel:
     - Project/section filters
     - Label multi-select filters
     - Priority filter (>= X)
     - Due date filters (today, overdue, next 7 days, no date)
     - Assignee filter (for shared projects)
   - Custom filter string input (supports Todoist filter syntax)
   - Filter persistence across page reloads
   - Multiple saved filter presets

9. **Additional Todoist Features**
   - Natural language quick add (parse due_string)
   - Task reordering (drag and drop syncs order to Todoist)
   - Archive/delete tasks with confirmation
   - Recurring task detection and visual indicator
   - Task duplication
   - Move task to different project/section
   - Bulk operations (complete multiple, delete multiple, move multiple)

### Phase 4: Integration and Polish

**Objective**: Integrate with Glance systems and polish UX

1. **Documentation**
   - Add Todoist section to `docs/configuration.md`
   - Document all configuration options
   - Provide example configurations
   - Add troubleshooting section
   - Include screenshots in final documentation

2. **Error Handling**
   - Graceful degradation when API is unreachable
   - Display error messages in widget
   - Retry logic for failed requests
   - Rate limit handling (show message when limited)

3. **Caching Strategy**
   - Cache tasks for configurable duration (default 5m)
   - Cache projects and labels (longer duration, 1h)
   - Invalidate cache on create/update/delete operations
   - Respect Todoist's rate limits

4. **Testing**
   - Test with real Todoist account
   - Test various filter configurations
   - Test error conditions (invalid token, rate limit, network errors)
   - Test task creation, completion, deletion
   - Test subtasks and comments

5. **Mobile Responsiveness**
   - Ensure task list works on mobile
   - Responsive forms for task creation
   - Touch-friendly subtask expansion

## Step by Step Tasks

### 1. Setup and Configuration Foundation

- [ ] Read existing widget implementations (widget-rss.go, widget-custom-api.go, widget-todo.go)
- [ ] Create `internal/glance/todoist/` directory for Todoist-specific code
- [ ] Define `todoistWidget` configuration struct in `internal/glance/config-fields.go`
- [ ] Add configuration parsing for todoist widget type
- [ ] Register widget type in `internal/glance/config.go`
- [ ] Add `TODOIST_API_TOKEN` to `.env.example`
- [ ] Document environment variable setup

### 2. API Client Implementation

- [ ] Create `internal/glance/todoist/client.go`
- [ ] Define comprehensive API request/response structs:
  - **Task struct**: id, content, description, comment_count, is_completed, order, priority, project_id, section_id, parent_id, creator_id, created_at, assignee_id, assigner_id, due (object with date, is_recurring, datetime, string, timezone, lang), deadline (object with date), duration (object with amount, unit), labels ([]string), url
  - **Project struct**: id, name, comment_count, order, color, is_shared, is_favorite, is_inbox_project, is_team_inbox, view_style, url, parent_id
  - **Section struct**: id, project_id, order, name
  - **Label struct**: id, name, color, order, is_favorite
  - **Comment struct**: id, content, task_id, project_id, posted_at, attachment (object with file_name, file_type, file_url, resource_type)
  - **Collaborator struct**: id, name, email
  - **CreateTaskRequest struct**: content, description, project_id, section_id, parent_id, order, labels ([]string), priority, due_string, due_date, due_datetime, due_lang, assignee_id, duration, duration_unit, deadline_date
  - **UpdateTaskRequest struct**: same as CreateTaskRequest (all optional)
  - **CreateProjectRequest struct**: name, parent_id, color, is_favorite, view_style
  - **UpdateProjectRequest struct**: name, color, is_favorite, view_style
  - **CreateSectionRequest struct**: name, project_id, order
  - **UpdateSectionRequest struct**: name
  - **CreateLabelRequest struct**: name, order, color, is_favorite
  - **UpdateLabelRequest struct**: name, order, color, is_favorite
  - **CreateCommentRequest struct**: content, task_id, project_id, attachment (object with resource_type, file_url, file_type, file_name)

- [ ] Implement `NewClient(apiToken string)` function
- [ ] **Tasks API Implementation**:
  - [ ] `GetTasks(opts)` with project_id, section_id, label, filter, lang, ids parameters
  - [ ] `GetTask(taskId)` - returns single task with all details
  - [ ] `CreateTask(request)` - POST to /rest/v2/tasks
  - [ ] `UpdateTask(taskId, request)` - POST to /rest/v2/tasks/{id}
  - [ ] `CloseTask(taskId)` - POST to /rest/v2/tasks/{id}/close
  - [ ] `ReopenTask(taskId)` - POST to /rest/v2/tasks/{id}/reopen
  - [ ] `DeleteTask(taskId)` - DELETE to /rest/v2/tasks/{id}

- [ ] **Projects API Implementation**:
  - [ ] `GetProjects()` - GET /rest/v2/projects
  - [ ] `GetProject(projectId)` - GET /rest/v2/projects/{id}
  - [ ] `CreateProject(request)` - POST /rest/v2/projects
  - [ ] `UpdateProject(projectId, request)` - POST /rest/v2/projects/{id}
  - [ ] `ArchiveProject(projectId)` - POST /rest/v2/projects/{id}/archive
  - [ ] `UnarchiveProject(projectId)` - POST /rest/v2/projects/{id}/unarchive
  - [ ] `DeleteProject(projectId)` - DELETE /rest/v2/projects/{id}
  - [ ] `GetProjectCollaborators(projectId)` - GET /rest/v2/projects/{id}/collaborators

- [ ] **Sections API Implementation**:
  - [ ] `GetSections(projectId)` - GET /rest/v2/sections with optional project_id filter
  - [ ] `GetSection(sectionId)` - GET /rest/v2/sections/{id}
  - [ ] `CreateSection(request)` - POST /rest/v2/sections
  - [ ] `UpdateSection(sectionId, request)` - POST /rest/v2/sections/{id}
  - [ ] `DeleteSection(sectionId)` - DELETE /rest/v2/sections/{id}

- [ ] **Comments API Implementation**:
  - [ ] `GetComments(opts)` - GET /rest/v2/comments with task_id or project_id
  - [ ] `GetComment(commentId)` - GET /rest/v2/comments/{id}
  - [ ] `CreateComment(request)` - POST /rest/v2/comments
  - [ ] `UpdateComment(commentId, content)` - POST /rest/v2/comments/{id}
  - [ ] `DeleteComment(commentId)` - DELETE /rest/v2/comments/{id}

- [ ] **Labels API Implementation**:
  - [ ] `GetLabels()` - GET /rest/v2/labels (personal labels)
  - [ ] `GetLabel(labelId)` - GET /rest/v2/labels/{id}
  - [ ] `CreateLabel(request)` - POST /rest/v2/labels
  - [ ] `UpdateLabel(labelId, request)` - POST /rest/v2/labels/{id}
  - [ ] `DeleteLabel(labelId)` - DELETE /rest/v2/labels/{id}
  - [ ] `GetSharedLabels(omitPersonal)` - GET /rest/v2/labels/shared
  - [ ] `RenameSharedLabel(name, newName)` - POST /rest/v2/labels/shared/rename
  - [ ] `RemoveSharedLabel(name)` - POST /rest/v2/labels/shared/remove

- [ ] Add error handling for all HTTP requests (401 unauthorized, 429 rate limit, 5xx server errors)
- [ ] Add retry logic for transient failures (network errors, 5xx)
- [ ] Implement rate limiting awareness (track request count, respect 1000/15min limit)
- [ ] Handle 301 redirects for moved tasks/sections/projects

### 3. Widget Backend Implementation

- [ ] Create `internal/glance/widget-todoist.go`
- [ ] Define `todoistWidget` struct with configuration fields
- [ ] Implement `initialize()` method
- [ ] Implement `update()` method to fetch data from API
- [ ] Implement `Render()` method with template
- [ ] Add caching logic (check cache, fetch if expired)
- [ ] Handle different filter modes:
  - Filter by project ID
  - Filter by label
  - Filter by "today" (due date <= today)
  - Filter by "inbox" (inbox project)
  - Filter by priority (>= X)
- [ ] Pass data to template with proper structure
- [ ] Handle API errors gracefully (show error in widget)

### 4. Frontend Templates

- [ ] Create `internal/glance/templates/todoist.html`
- [ ] Extend `widget-base.html` structure
- [ ] Add task list container
- [ ] Add individual task item template with:
  - Checkbox input
  - Task content (markdown rendered)
  - Due date badge
  - Priority indicator
  - Label badges
  - Subtask toggle button
  - Delete button
- [ ] Add subtask list template (nested under parent task)
- [ ] Add comments section template (hidden by default)
- [ ] Add "Add task" form with:
  - Text input for task content
  - Due date input
  - Priority selector
  - Submit button
- [ ] Add loading state indicator
- [ ] Add error message display
- [ ] Add project/section header (if filtered by project)

### 5. Styling

- [ ] Create `internal/glance/static/css/widget-todoist.css`
- [ ] Copy base styles from `widget-todo.css`
- [ ] Add priority indicator styles:
  - Priority 1 (normal): neutral color
  - Priority 2 (medium): yellow/orange
  - Priority 3 (high): orange/red
  - Priority 4 (urgent): red
- [ ] Add label badge styles (small rounded tags)
- [ ] Add subtask indentation styles
- [ ] Add collapsed/expanded state indicators
- [ ] Add due date badge styles (overdue in red)
- [ ] Add comment icon with count
- [ ] Add project/section indicator styles
- [ ] Ensure responsive design for mobile
- [ ] Add loading spinner styles
- [ ] Add error alert styles

### 6. Client-Side JavaScript

- [ ] Create `internal/glance/static/js/todoist.js`
- [ ] Copy and adapt base functionality from `todo.js`
- [ ] Modify `Todo()` function to work with Todoist data structure
- [ ] Add AJAX request to create task:
  - POST to `/api/widgets/{widgetId}/tasks`
  - Handle response and add to list
  - Show error message on failure
- [ ] Add task completion handler:
  - POST to `/api/widgets/{widgetId}/tasks/{taskId}/close`
  - Remove from list or mark completed
- [ ] Add task deletion handler:
  - DELETE to `/api/widgets/{widgetId}/tasks/{taskId}`
  - Animate removal from list
- [ ] Add subtask toggle:
  - Expand/collapse subtask list
  - Save state to localStorage
- [ ] Add comment section toggle:
  - Fetch comments via AJAX
  - Display in expanded section
- [ ] Add comment submission:
  - POST to comment endpoint
  - Append to comment list
- [ ] Add drag-and-drop reordering (optional, complex)
- [ ] Initialize all Todoist widgets on page load
- [ ] Add auto-refresh on cache expiry

### 7. HTTP API Endpoints

- [ ] Add widget-specific API routes in `glance.go`:

  **Task Endpoints**:
  - [ ] `POST /api/widgets/:id/tasks` - Create task (with all parameters)
  - [ ] `PUT /api/widgets/:id/tasks/:taskId` - Update task
  - [ ] `POST /api/widgets/:id/tasks/:taskId/close` - Complete task
  - [ ] `POST /api/widgets/:id/tasks/:taskId/reopen` - Reopen task
  - [ ] `DELETE /api/widgets/:id/tasks/:taskId` - Delete task
  - [ ] `GET /api/widgets/:id/tasks/:taskId` - Get task with subtasks

  **Comment Endpoints**:
  - [ ] `GET /api/widgets/:id/tasks/:taskId/comments` - Get comments
  - [ ] `POST /api/widgets/:id/tasks/:taskId/comments` - Add comment
  - [ ] `GET /api/widgets/:id/comments/:commentId` - Get single comment
  - [ ] `PUT /api/widgets/:id/comments/:commentId` - Update comment
  - [ ] `DELETE /api/widgets/:id/comments/:commentId` - Delete comment
  - [ ] `GET /api/widgets/:id/projects/:projectId/comments` - Get project comments
  - [ ] `POST /api/widgets/:id/projects/:projectId/comments` - Add project comment

  **Project Endpoints**:
  - [ ] `GET /api/widgets/:id/projects` - Get all projects
  - [ ] `GET /api/widgets/:id/projects/:projectId` - Get single project
  - [ ] `POST /api/widgets/:id/projects` - Create project
  - [ ] `PUT /api/widgets/:id/projects/:projectId` - Update project
  - [ ] `POST /api/widgets/:id/projects/:projectId/archive` - Archive project
  - [ ] `POST /api/widgets/:id/projects/:projectId/unarchive` - Unarchive project
  - [ ] `DELETE /api/widgets/:id/projects/:projectId` - Delete project
  - [ ] `GET /api/widgets/:id/projects/:projectId/collaborators` - Get collaborators
  - [ ] `GET /api/widgets/:id/projects/:projectId/sections` - Get project sections

  **Section Endpoints**:
  - [ ] `GET /api/widgets/:id/sections` - Get all sections (with optional project filter)
  - [ ] `GET /api/widgets/:id/sections/:sectionId` - Get single section
  - [ ] `POST /api/widgets/:id/sections` - Create section
  - [ ] `PUT /api/widgets/:id/sections/:sectionId` - Update section
  - [ ] `DELETE /api/widgets/:id/sections/:sectionId` - Delete section

  **Label Endpoints**:
  - [ ] `GET /api/widgets/:id/labels` - Get all personal labels
  - [ ] `GET /api/widgets/:id/labels/:labelId` - Get single label
  - [ ] `POST /api/widgets/:id/labels` - Create label
  - [ ] `PUT /api/widgets/:id/labels/:labelId` - Update label
  - [ ] `DELETE /api/widgets/:id/labels/:labelId` - Delete label
  - [ ] `GET /api/widgets/:id/labels/shared` - Get shared labels (omit_personal param)
  - [ ] `POST /api/widgets/:id/labels/shared/rename` - Rename shared label
  - [ ] `POST /api/widgets/:id/labels/shared/remove` - Remove shared label

- [ ] Implement handlers for each endpoint in Go
- [ ] Add authentication checks (if auth is enabled)
- [ ] Return JSON responses with proper status codes (200, 201, 204, 400, 401, 429, 500)
- [ ] Handle errors and return appropriate messages:
  - 400 Bad Request - Invalid input
  - 401 Unauthorized - Invalid API token
  - 403 Forbidden - No permission
  - 404 Not Found - Resource doesn't exist
  - 429 Too Many Requests - Rate limit exceeded
  - 500 Server Error - API or internal error
- [ ] Add request ID tracking for idempotency (X-Request-Id header)
- [ ] Add CORS headers if needed
- [ ] Log all API calls for debugging (with sensitive data redacted)

### 8. Testing and Validation

**Tasks Management Testing**:
- [ ] Test widget displays tasks from default filter
- [ ] Test project filter configuration
- [ ] Test section filter configuration
- [ ] Test label filter configuration
- [ ] Test priority filter configuration
- [ ] Test custom filter string input
- [ ] Test task creation via form with all fields:
  - Content only (minimal task)
  - Task with description
  - Task with project
  - Task with section
  - Task with labels (multiple)
  - Task with priority
  - Task with due date (date picker)
  - Task with due datetime
  - Task with due_string natural language
  - Task with deadline
  - Task with duration (minutes and days)
  - Task with assignee (shared project)
  - Task as subtask (parent_id)
- [ ] Test task completion (close)
- [ ] Test task reopening
- [ ] Test task deletion
- [ ] Test task update:
  - Update content
  - Update priority
  - Update due date
  - Update labels
  - Move to different project/section
- [ ] Test subtask display and toggle
- [ ] Test subtask creation
- [ ] Test subtask completion (independent of parent)
- [ ] Test nested subtasks (subtask of subtask)

**Comments Testing**:
- [ ] Test comments display and creation
- [ ] Test comment update
- [ ] Test comment deletion
- [ ] Test comment with attachment
- [ ] Test project comments (different from task comments)

**Projects Testing**:
- [ ] Test projects list view
- [ ] Test project hierarchy (parent/child projects)
- [ ] Test project creation
- [ ] Test project update (name, color, favorite)
- [ ] Test project archival
- [ ] Test project unarchival
- [ ] Test project deletion (with confirmation)
- [ ] Test project collaborators display (shared projects)
- [ ] Test project sections view

**Sections Testing**:
- [ ] Test sections list
- [ ] Test sections filtered by project
- [ ] Test section creation
- [ ] Test section update (name)
- [ ] Test section deletion

**Labels Testing**:
- [ ] Test personal labels list
- [ ] Test label creation
- [ ] Test label update (name, color, favorite)
- [ ] Test label deletion
- [ ] Test shared labels list
- [ ] Test rename shared label
- [ ] Test remove shared label
- [ ] Test tasks displayed with label badges
- [ ] Test filtering by label

**Error Handling Testing**:
- [ ] Test with invalid API token (shows error)
- [ ] Test rate limiting (shows appropriate message)
- [ ] Test network error handling
- [ ] Test unauthorized (401) responses
- [ ] Test not found (404) responses
- [ ] Test server error (500) handling
- [ ] Test invalid input validation

**Performance Testing**:
- [ ] Test caching (doesn't refetch within cache duration)
- [ ] Test cache invalidation on create/update/delete
- [ ] Test performance with many tasks (100+)
- [ ] Test performance with many projects (50+)
- [ ] Test performance with many labels (100+)
- [ ] Test subtask rendering performance (deep nesting)

**UI/UX Testing**:
- [ ] Test mobile responsiveness
- [ ] Test responsive design on various screen sizes
- [ ] Test keyboard navigation
- [ ] Test screen reader accessibility
- [ ] Test drag-and-drop task reordering
- [ ] Test modal dialogs
- [ ] Test form validation
- [ ] Test loading states
- [ ] Test error message display
- [ ] Test toast notifications

**Integration Testing**:
- [ ] Test with various task configurations:
  - Tasks with due dates
  - Tasks with priorities (1-4)
  - Tasks with labels
  - Tasks with subtasks
  - Recurring tasks
  - Tasks in different projects/sections
  - Tasks with descriptions (markdown)
  - Tasks with attachments (via comments)
  - Assigned tasks (shared projects)
  - Tasks with duration
  - Tasks with deadlines
  - Overdue tasks
  - Tasks due today
  - Tasks without due dates
- [ ] Test multi-label filtering
- [ ] Test project hierarchy navigation
- [ ] Test collaborator assignment (shared projects)
- [ ] Test widget configuration reload

### 9. Documentation

- [ ] Add "Todoist" section to `docs/configuration.md`
- [ ] Document all configuration options with examples
- [ ] Document environment variable setup
- [ ] Document how to get Todoist API token
- [ ] Provide example configurations:
  - Show all tasks
  - Show today's tasks
  - Show tasks from specific project
  - Show tasks with specific label
  - Show high priority tasks
- [ ] Document API rate limits
- [ ] Add troubleshooting section
- [ ] Take screenshots of widget in various states
- [ ] Update README.md to mention Todoist widget (if appropriate)

### 10. Final Polish

- [ ] Review code for consistency with Glance patterns
- [ ] Ensure no unused dependencies
- [ ] Check error messages are user-friendly
- [ ] Verify responsive design on various screen sizes
- [ ] Test performance with many tasks (100+)
- [ ] Add loading states for better UX
- [ ] Verify no JavaScript console errors
- [ ] Check accessibility (keyboard navigation, screen readers)
- [ ] Run `go build` to ensure no build errors
- [ ] Run `go test ./...` to ensure no test regressions
- [ ] Test with Docker container
- [ ] Verify hot-reload works with config changes

## Testing Strategy

### Unit Tests

Create unit tests for:
- API client methods (mock HTTP responses)
- Configuration parsing
- Filter logic (today, priority, project, label)
- Data structure transformations
- Error handling paths

Test files:
- `internal/glance/todoist/client_test.go`
- `internal/glance/widget-todoist_test.go`

### Integration Tests

- Test widget with test Todoist account
- Test all API endpoints with real API
- Test caching behavior
- Test error scenarios (invalid token, rate limit)
- Test with various task configurations

### Edge Cases

Test these edge cases:
- Empty task list (no tasks match filter)
- Very long task content (truncate display)
- Tasks with no due date
- Tasks in the past (overdue)
- Nested subtasks (subtask of subtask)
- Tasks with many labels (display limit)
- Tasks with no project (inbox tasks)
- Recurring tasks
- Assigned tasks (from shared projects)
- Tasks with attachments (may not display)
- Unicode in task content
- Markdown in task content (render safely)
- Very deep subtask hierarchy (display limit)
- Many projects (dropdown performance)
- Many labels (multi-select performance)

## Acceptance Criteria

The feature is complete when:

1. **Configuration**
   - [ ] Widget can be configured via YAML with all documented options
   - [ ] API token can be provided via environment variable
   - [ ] Multiple Todoist widgets can coexist with different configurations
   - [ ] All filter options work correctly (project, section, label, priority, due date, custom filter)
   - [ ] Display options are configurable (compact mode, grouping, sorting, etc.)

2. **Task Display**
   - [ ] Tasks are displayed according to filter configuration
   - [ ] Task content is rendered with markdown support
   - [ ] Task descriptions are displayed (or hidden based on config)
   - [ ] Due dates are displayed in human-readable format
   - [ ] Due dates with time show datetime correctly
   - [ ] Priority is visually indicated (color/icon)
   - [ ] Labels are displayed as colored badges
   - [ ] Project/section is indicated (with color)
   - [ ] Overdue tasks are highlighted (red)
   - [ ] Recurring tasks show recurrence indicator
   - [ ] Duration badges are displayed (when configured)
   - [ ] Deadline badges are displayed (when configured)
   - [ ] Assignee avatars are shown (for shared tasks)
   - [ ] Subtask count is displayed
   - [ ] Comment count is displayed
   - [ ] Task order can be sorted by multiple criteria

3. **Task Management**
   - [ ] New tasks can be created with all fields:
     - Content (required)
     - Description (markdown)
     - Project selection
     - Section selection
     - Parent task selection (for subtasks)
     - Labels (multi-select)
     - Priority (1-4)
     - Due date (natural language or picker)
     - Due datetime
     - Deadline date
     - Duration (amount + unit)
     - Assignee (for shared projects)
   - [ ] Tasks can be marked as complete (close)
   - [ ] Completed tasks can be reopened
   - [ ] Tasks can be deleted
   - [ ] Tasks can be edited (inline or modal)
   - [ ] Tasks can be moved to different project/section
   - [ ] Changes sync to Todoist immediately
   - [ ] List refreshes after modifications
   - [ ] Tasks can be reordered (drag and drop)

4. **Subtasks**
   - [ ] Subtasks are displayed nested under parent task
   - [ ] Subtasks can be expanded/collapsed
   - [ ] Subtasks can be completed independently
   - [ ] New subtasks can be added
   - [ ] Subtasks can be reordered
   - [ ] Nested subtasks are supported (subtask of subtask)
   - [ ] Collapsed state is persisted (localStorage)
   - [ ] Visual hierarchy is clear (indentation, connecting lines)

5. **Comments**
   - [ ] Comment count is displayed on task
   - [ ] Comments can be viewed (expanded section)
   - [ ] New comments can be added
   - [ ] Comments can be updated
   - [ ] Comments can be deleted
   - [ ] Comment author is shown
   - [ ] Comment timestamp is shown (relative time)
   - [ ] Comment attachments are displayed with icon and filename
   - [ ] Comments support markdown rendering
   - [ ] Project comments are supported (separate from task comments)

6. **Projects**
   - [ ] All projects are visible in project browser
   - [ ] Project hierarchy is displayed (parent/child)
   - [ ] Project colors are shown
   - [ ] Favorite indicator is shown
   - [ ] Shared project indicator is shown
   - [ ] Task count per project is displayed
   - [ ] New projects can be created
   - [ ] Projects can be updated (name, color, favorite)
   - [ ] Projects can be archived
   - [ ] Projects can be unarchived
   - [ ] Projects can be deleted (with confirmation)
   - [ ] Project sections are visible
   - [ ] Collaborators are listed for shared projects

7. **Sections**
   - [ ] All sections are visible
   - [ ] Sections can be filtered by project
   - [ ] New sections can be created
   - [ ] Sections can be updated (name)
   - [ ] Sections can be deleted

8. **Labels**
   - [ ] All personal labels are visible in label browser
   - [ ] Label colors are shown
   - [ ] Label favorite indicator is shown
   - [ ] Task count per label is displayed
   - [ ] Shared labels are shown in separate section
   - [ ] New labels can be created
   - [ ] Labels can be updated (name, color, favorite)
   - [ ] Labels can be deleted
   - [ ] Shared labels can be renamed
   - [ ] Shared labels can be removed
   - [ ] Tasks can be filtered by label
   - [ ] Multiple labels can be selected

9. **Filtering and Navigation**
   - [ ] Tasks can be filtered by project
   - [ ] Tasks can be filtered by section
   - [ ] Tasks can be filtered by label (multiple)
   - [ ] Tasks can be filtered by priority (>= X)
   - [ ] Tasks can be filtered by due date (today, overdue, upcoming, none)
   - [ ] Tasks can be filtered by assignee
   - [ ] Custom filter string works (supports Todoist syntax)
   - [ ] Specific task IDs can be displayed
   - [ ] Tasks can be grouped by project/priority/due date
   - [ ] Tasks can be sorted by multiple criteria
   - [ ] Sort order (asc/desc) works correctly
   - [ ] Navigate between projects without page reload

10. **Error Handling**
    - [ ] Invalid API token shows user-friendly error
    - [ ] Network errors show user-friendly error
    - [ ] Rate limit shows appropriate message (429)
    - [ ] Widget doesn't crash on API errors
    - [ ] Failed operations show error toast
    - [ ] Validation errors are shown inline
    - [ ] Retry logic works for transient failures
    - [ ] 301 redirects are handled correctly

11. **Performance**
    - [ ] Initial page load completes in < 3 seconds (with 50 tasks)
    - [ ] Task creation completes in < 1 second
    - [ ] Task updates complete in < 1 second
    - [ ] Cache reduces API calls appropriately
    - [ ] Widget remains responsive with 100+ tasks
    - [ ] Project browser loads quickly with 50+ projects
    - [ ] Label browser loads quickly with 100+ labels
    - [ ] Subtask rendering is fast (even with deep nesting)
    - [ ] Rate limit isn't exceeded (1000/15min)

12. **UI/UX**
    - [ ] Interface is responsive on mobile devices
    - [ ] Forms work on mobile (responsive inputs)
    - [ ] Touch-friendly subtask expansion
    - [ ] Keyboard navigation works
    - [ ] Screen reader announcements work
    - [ ] Loading states are visible
    - [ ] Success/error toasts are shown
    - [ ] Drag-and-drop works smoothly
    - [ ] Modals are accessible
    - [ ] Color contrast meets accessibility standards
    - [ ] Focus indicators are visible

13. **Documentation**
    - [ ] All configuration options are documented
    - [ ] Setup instructions are clear
    - [ ] Examples cover common use cases:
      - Show all tasks
      - Show today's tasks
      - Show tasks from specific project
      - Show tasks with specific label
      - Show high priority tasks
      - Show overdue tasks
      - Show tasks assigned to me
      - Custom filter examples
    - [ ] Troubleshooting section is helpful
    - [ ] API rate limits are documented
    - [ ] Screenshots show various widget states

14. **Code Quality**
    - [ ] Code follows Glance patterns and conventions
    - [ ] No unnecessary dependencies added
    - [ ] Error handling is comprehensive
    - [ ] Code is commented where complex
    - [ ] No build or test errors

## Validation Commands

Execute every command to validate the feature works correctly with zero regressions.

### Build and Run Validation

```bash
# Build the application
go build -o build/glance .

# Run the application (will fail if config is invalid, which is expected)
./build/glance --config config/glance.yml

# Validate configuration
./build/glance config:validate --config config/glance.yml

# Build for multiple architectures
GOOS=linux GOARCH=amd64 go build -o build/glance-amd64 .
GOOS=linux GOARCH=arm64 go build -o build/glance-arm64 .
GOOS=darwin GOARCH=amd64 go build -o build/glance-darwin-amd64 .
```

### Testing Commands

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for specific package
go test ./internal/glance/todoist/...

# Run tests with verbose output
go test -v ./internal/glance/...

# Run race detector
go test -race ./...
```

### Docker Validation

```bash
# Build Docker image
docker build -t glance:latest .

# Run container with test config
docker run --rm -v $(pwd)/config:/app/config glance:latest

# Test with environment variable
docker run --rm -v $(pwd)/config:/app/config -e TODOIST_API_TOKEN=test glance:latest

# Verify container starts and serves page
docker run -d -p 8080:8080 -v $(pwd)/config:/app/config --name glance-test glance:latest
sleep 5
curl -f http://localhost:8080 || exit 1
docker stop glance-test
docker rm glance-test
```

### Manual Testing Checklist

```bash
# 1. Start the server
./build/glance --config config/glance.yml

# 2. Open browser to http://localhost:8080

# 3. Verify in browser console:
#    - No JavaScript errors
#    - Widget loads and displays tasks
#    - Network tab shows API calls

# 4. Test widget functionality:
#    - Tasks display correctly
#    - Can create new task
#    - Can complete task
#    - Can delete task
#    - Subtasks expand/collapse
#    - Comments display

# 5. Test configuration reload:
#    - Edit config/glance.yml
#    - Save file
#    - Verify widget updates without restart

# 6. Test error conditions:
#    - Stop server
#    - Set invalid TODOIST_API_TOKEN
#    - Start server
#    - Verify error message displays in widget
```

### Code Quality Validation

```bash
# Format code
go fmt ./...

# Run linter (if golangci-lint is installed)
golangci-lint run

# Check for unused dependencies
go mod tidy

# Verify no vendored dependencies
ls vendor/ && exit 1 || true

# Check for TODO/FIXME comments that should be addressed
grep -r "TODO\|FIXME" internal/glance/widget-todoist.go internal/glance/todoist/
```

### Documentation Validation

```bash
# Verify config example is valid YAML
./build/glance config:validate --config docs/glance.yml

# Verify documentation has no broken links (if link checker available)
# Check all examples in docs/configuration.md are syntactically correct
grep -A 20 "## Todoist" docs/configuration.md | grep -E "^- type: todoist"

# Verify all documented options exist in code
grep "todo:" docs/configuration.md | wc -l  # Count examples
```

## Notes

### Important Considerations

1. **Python vs Go Implementation**:
   - Initially considered Python SDK, but decided on pure Go implementation
   - Reasons: Glance's "no new dependencies" principle, simpler deployment, no runtime dependency
   - Todoist REST API v2 is straightforward to use with Go's `net/http`
   - This aligns better with Glance's architecture

2. **Authentication Security**:
   - API token should never be logged
   - Support for Docker secrets: `${secret:todoist_token}`
   - Token should be validated on widget initialization
   - Clear error message if token is invalid

3. **API Rate Limiting**:
   - Todoist allows 1000 requests per 15 minutes per user
   - Implement caching to minimize API calls
   - Consider background refresh for better UX
   - Handle 429 responses gracefully with retry-after

4. **Widget Uniqueness**:
   - Unlike the basic todo widget, this requires a valid API token
   - Should be clearly labeled as "Todoist" to distinguish from basic todo
   - Consider adding an icon to distinguish visually
   - Supports full Todoist feature set (projects, labels, comments, etc.)

5. **Data Synchronization**:
   - Todoist is the source of truth
   - Widget should reflect Todoist state, not try to fully replicate it
   - For advanced features, may need to link to Todoist web app
   - Consider adding "Open in Todoist" links (task.url property)
   - Handle 301 redirects when tasks/projects/sections are moved
   - Implement optimistic UI updates for better perceived performance

6. **Task Properties Overview**:
   - **Basic**: content, description, is_completed
   - **Organization**: project_id, section_id, parent_id, order, labels
   - **Priority**: 1=normal, 2=medium, 3=high, 4=urgent
   - **Scheduling**: due (with date, datetime, string, timezone, is_recurring, lang)
   - **Deadlines**: deadline.date (separate from due date)
   - **Duration**: duration.amount + duration.unit (minute or day)
   - **Assignment**: assignee_id, assigner_id (for shared projects)
   - **Metadata**: id, creator_id, created_at, comment_count, url

7. **Project Properties Overview**:
   - **Basic**: id, name, color, parent_id
   - **Status**: is_shared, is_favorite, is_inbox_project, is_team_inbox
   - **Display**: view_style (list or board), order
   - **Metadata**: comment_count, url
   - **Special projects**: Inbox (is_inbox_project: true), Team Inbox (is_team_inbox: true)

8. **Section Properties**:
   - Basic: id, name, project_id, order
   - Sections belong to a single project
   - Deleting a section deletes all its tasks

9. **Label Types**:
   - **Personal Labels**: User-created, customizable (name, color, order, is_favorite)
   - **Shared Labels**: From collaborators, can be renamed/removed across all tasks
   - Labels are stored as array of strings on tasks

10. **Comment System**:
    - Comments can be on tasks or projects
    - Support attachments (file_url, file_type, file_name, resource_type)
    - Markdown content support
    - Properties: id, content, posted_at, attachment, task_id/project_id

11. **Collaboration Features**:
    - Shared projects: is_shared flag
    - Assignees: assignee_id on tasks
    - Collaborators: list via /projects/{id}/collaborators
    - Project comments: separate from task comments

12. **Filter Language**:
    - Supports Todoist's filter syntax
    - Language parameter for natural language parsing (due_string)
    - Filter precedence: filter > ids > label/project_id/section_id
    - Examples: "today & priority high", "overdue", "no due date"

13. **Compatibility with Existing Todo Widget**:
    - Keep basic `todo` widget intact
    - Users can choose which widget to use based on needs
    - Consider migration path from basic todo to Todoist (export/import)

14. **Performance Considerations**:
    - Large task lists (100+) could slow down rendering
    - Consider pagination or virtual scrolling
    - Lazy load subtasks and comments
    - Debounce rapid state changes
    - Cache projects and labels longer than tasks (they change less often)
    - Implement efficient data structures for lookups

15. **Accessibility**:
    - Ensure keyboard navigation works
    - Add ARIA labels for interactive elements
    - Support screen reader announcements for task changes
    - Maintain focus management during updates
    - Use semantic HTML elements
    - Provide sufficient color contrast

16. **Browser Compatibility**:
    - Test on modern browsers (Chrome, Firefox, Safari, Edge)
    - Graceful degradation for older browsers
    - No reliance on bleeding-edge APIs
    - Use feature detection where appropriate

### References

- Todoist REST API v2: https://developer.todoist.com/rest/v2/
- Todoist API Python SDK: https://github.com/doist/todoist-api-python
- Todoist Integration Guide: https://developer.todoist.com/guides/
- Glance Contributing Guidelines: README.md section
- Glance Widget System: `internal/glance/widget.go`
- Todoist Help Center: https://todoist.com/help/articles/
- Todoist Filter Syntax: https://todoist.com/help/articles/205248802
- Todoist Colors: https://developer.todoist.com/sync/v9/#colors
- Todoist Text Formatting (Markdown): https://todoist.com/help/articles/205338345
